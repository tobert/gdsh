// Copyright 2013 Albert P. Tobey. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// references: http://dave.cheney.net/tag/golang

package gdssh

import (
	"bufio"
	"code.google.com/p/go.crypto/ssh"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"strings"
	"time"
)

type Conn struct {
	Host      string
	Port      int
	User      string
	Key       string
	Retries   int
	Started   time.Time // last time the connection was made, reset by each retry
	connected bool      // for tracking whether the connection is alive
	done      chan bool // for notifying goroutines to stop retrying
	address   string    // host:port formatted connection address
	netconn   net.Conn
	config    *ssh.ClientConfig
	client    *ssh.ClientConn
}

func NewConn(host string, port int, user string, key string) (conn *Conn) {
	return &Conn{
		Host:      host,
		Port:      port,
		User:      user,
		Key:       key,
		Retries:   0,
		connected: false,
		done:      make(chan bool),
		address:   fmt.Sprintf("%s:%d", host, port),
	}
}

func (conn *Conn) Connect() error {
	var auth []ssh.ClientAuth

	// only load a private key if requested ~/.ssh/id_rsa is _not_ loaded automatically
	// ssh-agent should be the usual path
	if conn.Key != "" {
		kr := new(keyring)
		if err := kr.loadPEM(conn.Key); err != nil {
			log.Fatal("Couldn't load specified private key '", conn.Key, "': ", err)
		}
		auth = append(auth, ssh.ClientAuthKeyring(kr))
	}

	agentSock := os.Getenv("SSH_AUTH_SOCK")

	// ssh-agent support, might need to reuse this in the future?
	// how bad are 100's or 1000's of connections to the agent?
	if agentSock != "" {
		sock, err := net.Dial("unix", agentSock)
		if err != nil {
			log.Fatal("Could not connect to SSH_AUTH_SOCK. Is ssh-agent running?")
		}

		agent := ssh.NewAgentClient(sock)
		auth = append(auth, ssh.ClientAuthAgent(agent))
	}

	conn.config = &ssh.ClientConfig{
		User: conn.User,
		Auth: auth,
	}

	return conn.connect()
}

func (conn *Conn) connect() (err error) {
	if conn.connected {
		panic("BUG: connect() called on connected socket/client!")
	}

	// dial manually so the tcp socket can be closed directly since it's hidden
	// if you use ssh.Dial, might also be handy for tuning?
	conn.netconn, err = net.Dial("tcp", conn.address)
	if err != nil {
		conn.connected = false
		return
	}

	conn.client, err = ssh.Client(conn.netconn, conn.config)
	if err != nil {
		conn.connected = false
		return
	}

	conn.Started = time.Now()
	conn.connected = true

	return
}

func (conn *Conn) Reconnect() error {
	conn.Close()
	return conn.connect()
}

// TODO: some kind of keepalive, possibly open & close ssh channels
func (conn *Conn) Alive() bool {
	return conn.connected
}

func (conn *Conn) Close() {
	// TODO: watch for memory leaks on long-running programs in case
	// ssh.Client() doesn't have a Close()
	conn.netconn.Close() // close the underlying TCP connection, ignore errors
	conn.connected = false
	conn.client = nil
}

// scp a buffer to a file on the remote machine
func (conn *Conn) ScpBuf(buf []byte, mode string, remoteFile string) {
	sess, err := conn.client.NewSession()
	if err != nil {
		log.Fatal("[", conn.Host, "] session creation failed: ", err)
	}
	defer sess.Close()

	stdin, _ := sess.StdinPipe()
	stdout, _ := sess.StdoutPipe()
	//stderr, _ := sess.StderrPipe()

	cmd := fmt.Sprintf("/usr/bin/scp -t -- %s", remoteFile)
	if err := sess.Start(cmd); err != nil {
		log.Fatal("[", conn.address, "] command failed: ", err)
	}

	stdin.Write([]byte(fmt.Sprintf("C%s %d %s\n", mode, len(buf), path.Base(remoteFile))))
	response := make([]byte, 1, 1)
	stdout.Read(response)
	stdin.Write(buf)
	stdin.Close()
	// TODO: pull back stdout / stderr for errors
	err = sess.Wait()
}

func (conn *Conn) ScpPush(localFile string, remoteFile string) {
	f, err := os.Open(localFile)
	if err != nil {
		log.Fatal("Could not read local file '", localFile, "': ", err)
	}
	defer f.Close()
	fi, err := f.Stat()

	sess, err := conn.client.NewSession()
	if err != nil {
		log.Fatal("[", conn.Host, "] session creation failed: ", err)
	}
	defer sess.Close()

	stdin, _ := sess.StdinPipe()
	stdout, _ := sess.StdoutPipe()

	cmd := fmt.Sprintf("/usr/bin/scp -t -- %s", remoteFile)

	if err := sess.Start(cmd); err != nil {
		log.Fatal("[", conn.address, "] command failed: ", err)
	}

	mode := fi.Mode().Perm()
	size := fi.Size()
	base := path.Base(remoteFile)

	stdin.Write([]byte(fmt.Sprintf("C%04o %d %s\n", mode, size, base)))
	response := make([]byte, 1, 1)
	stdout.Read(response)
	io.Copy(stdin, f)
	stdin.Close()
	err = sess.Wait()
}

func (conn *Conn) ScpPull(localFile string, remoteFile string) {
	// overwrite in place, no tempfiles for now
	f, err := os.Create(localFile)
	if err != nil {
		log.Fatal("Could not open local file '", localFile, "' for writing: ", err)
	}

	sess, err := conn.client.NewSession()
	if err != nil {
		log.Fatal("[", conn.Host, "] session creation failed: ", err)
	}
	defer sess.Close()

	stdin, _ := sess.StdinPipe()
	stdout, _ := sess.StdoutPipe()

	cmd := fmt.Sprintf("/usr/bin/scp -f -- %s", remoteFile)
	if err := sess.Start(cmd); err != nil {
		log.Fatal("[", conn.address, "] command failed: ", err)
	}

	stdin.Write([]byte{0}) // ack

	var mode int
	var size int64
	var name string
	bs := bufio.NewReader(stdout)
	for {
		line, err := bs.ReadString('\n')
		stdin.Write([]byte{0}) // ack every message
		if err != nil {
			fmt.Printf("BUG: %s\n", err)
			break
		}
		// for now, we only care about the final line with filename/mode/size
		if strings.HasPrefix(line, "C") {
			_, err := fmt.Sscanf(line, "C%04o %d %s\n", &mode, &size, &name)
			if err != nil {
				fmt.Printf("BUG: %s\n", err)
				break
			}
			break
		} else {
			fmt.Printf("ignoring: '%s'\n", line)
		}
	}

	written, err := io.CopyN(f, bs, size)
	if err != nil {
		fmt.Printf("BUG(%d/%d bytes): %s\n", written, size, err)
	}

	stdin.Write([]byte{0}) // ack the data

	stdin.Close()
	err = sess.Wait()
	if err != nil {
		fmt.Printf("BUG(%d/%d bytes): %s\n", written, size, err)
	}
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
