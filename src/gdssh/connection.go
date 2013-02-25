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
)

type SshConn struct {
	Host    string
	Port    int
	User    string
	Key     string
	address string
	auth    []ssh.ClientAuth
	client  *ssh.ClientConn
}

func Connect(host string, port int, user string, key string) (conn *SshConn, err error) {
	conn = &SshConn{
		Host:    host,
		Port:    port,
		User:    user,
		Key:     key,
		address: fmt.Sprintf("%s:%d", host, port),
		auth:    []ssh.ClientAuth{},
	}

	// only load a private key if requested ~/.ssh/id_rsa is _not_ loaded automatically
	// ssh-agent should be the usual path
	if key != "" {
		kr := new(keyring)
		if err := kr.loadPEM(key); err != nil {
			log.Fatal("Couldn't load specified private key '", key, "': ", err)
		}
		conn.auth = append(conn.auth, ssh.ClientAuthKeyring(kr))
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
		conn.auth = append(conn.auth, ssh.ClientAuthAgent(agent))
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: conn.auth,
	}

	conn.client, err = ssh.Dial("tcp", conn.address, config)

	return
}

// scp a buffer to a file on the remote machine
func Scp(conn *SshConn, buf []byte, mode string, remoteFile string) {
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
