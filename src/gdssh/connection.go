// Copyright 2013 Albert P. Tobey. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// references: http://dave.cheney.net/tag/golang

package gdssh

import (
	"code.google.com/p/go.crypto/ssh"
	"fmt"
	"log"
	"net"
	"os"
	"path"
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
func (conn *Conn) Scp(buf []byte, mode string, remoteFile string) {
	sess, err := conn.client.NewSession()
	if err != nil {
		log.Fatal("[", conn.Host, "] session creation failed: ", err)
	}
	defer sess.Close()

	stdin, _ := sess.StdinPipe()
	stdout, _ := sess.StdoutPipe()
	//stderr, _ := sess.StderrPipe()

	cmd := fmt.Sprintf("/usr/bin/scp -t %s", remoteFile)
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

// TODO: add ScpFile

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
