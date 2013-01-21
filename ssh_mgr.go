// Copyright 2013 Albert P. Tobey. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// references: http://dave.cheney.net/tag/golang

package main

import (
	"io"
	"bufio"
	"code.google.com/p/go.crypto/ssh"
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"sync"
)

// SshConn is an ssh connection and a channel for creating sessions/ssh channels
type sshConn struct {
	address string
	ssh     *ssh.ClientConn
	session chan *SshSess
	done    chan bool
}

// NewSshConn returns a new sshConn with channels pre-made
func newSshConn(address string, ssh *ssh.ClientConn) *sshConn {
	return &sshConn{
		address,             // address
		ssh,                 // ssh
		make(chan *SshSess), // command
		make(chan bool),     // done
	}
}

// SshConnMgr represents a collection of ssh connections
type SshConnMgr struct {
	conns map[string]*sshConn
	auth  *[]ssh.ClientAuth
}

// NewSshConnMgr initialize a new connection manager with a keyring and ssh-agent
func NewSshConnMgr(key string) (mgr *SshConnMgr) {
	mgr = &SshConnMgr{}
	mgr.conns = map[string]*sshConn{}
	auth := []ssh.ClientAuth{}
	agentSock := os.Getenv("SSH_AUTH_SOCK")

	// only load a private key if requested ~/.ssh/id_rsa is _not_ loaded automatically
	// ssh-agent should be the usual path
	if key != "" {
		kr := new(keyring)
		if err := kr.loadPEM(key); err != nil {
			log.Fatal("Couldn't load specified private key '", key, "': ", err)
		}
		auth = append(auth, ssh.ClientAuthKeyring(kr))
	}

	// ssh-agent support
	if agentSock != "" {
		sock, err := net.Dial("unix", agentSock)
		if err != nil {
			log.Fatal("Could not connect to SSH_AUTH_SOCK. Is ssh-agent running?")
		}

		agent := ssh.NewAgentClient(sock)
		auth = append(auth, ssh.ClientAuthAgent(agent))
	}

	mgr.auth = &auth
	return
}

// runCmdAll runs an SshCmd on all nodes connected by the manager.
// TODO: timeouts
func (mgr *SshConnMgr) RunCmdAll(command string, env map[string]string) {
	sessions := []*SshSess{}

	for _, conn := range mgr.conns {
		s := NewSshSess(command, env)
		conn.session <-s
		sessions = append(sessions, s)
	}

	for _, cmd := range sessions {
		<-cmd.done
	}
}

// ScpToAll scp's a file to all nodes in the manager
func (mgr *SshConnMgr) ScpAll(buf []byte, mode string, remote string) {
	wg := sync.WaitGroup{}
	for _, conn := range mgr.conns {
		wg.Add(1)
		go func() {
			mgr.scpTo(conn, buf, mode, remote)
			wg.Done()
		}()
	}
	wg.Wait()
}

// scp a buffer to a file on the remote machine
func (mgr *SshConnMgr) scpTo(conn *sshConn, buf []byte, mode string, remote string) {
	sess, err := conn.ssh.NewSession()
	if err != nil {
		log.Fatal("[", conn.address, "] session creation failed: ", err)
	}
	defer sess.Close()

	stdin, _ := sess.StdinPipe()
	stdout, _ := sess.StdoutPipe()
	//stderr, _ := sess.StderrPipe()

	cmd := fmt.Sprintf("/usr/bin/scp -t %s", remote)
	if err := sess.Start(cmd); err != nil {
		log.Fatal("[", conn.address, "] command failed: ", err)
	}

	stdin.Write([]byte(fmt.Sprintf("C%s %d %s\n", mode, len(buf), path.Base(remote))))
	response := make([]byte, 1, 1)
	stdout.Read(response)
	stdin.Write(buf)
	stdin.Close()
	// TODO: pull back stdout / stderr for errors
	err = sess.Wait()
}

// stopAll sends a 'done' message to each connections' goroutine so
// it will exit cleanly
func (mgr *SshConnMgr) StopAll() {
	for _, conn := range mgr.conns {
		conn.done <- true
	}

	// wait for confirmation
	for _, conn := range mgr.conns {
		<-conn.done
	}
}

// connectList tells the connection manager to connect to all nodes in the
// provided list of addresses
// expects a list of ssh addresses in address:port format
func (mgr *SshConnMgr) ConnectList(list []string, user string, key string) {
	conns := []*sshConn{}
	wg := sync.WaitGroup{}

	for _, address := range list {
		conn := newSshConn(address, nil)

		config := &ssh.ClientConfig{
			User: user,
			Auth: *mgr.auth,
		}

		wg.Add(1)
		go func() {
			var err error
			conn.ssh, err = ssh.Dial("tcp", conn.address, config)
			if err != nil {
				log.Fatal("ssh connection to ", conn.address, " failed: ", err)
			}

			wg.Done()

			for {
				select {
				case session, _ := <-conn.session:
					go conn.runCommandSession(session)
				}
			}
		}()

		mgr.conns[address] = conn
		conns = append(conns, conn)
	}

	wg.Wait()
}

// runCommandSession runs a single command on the remote host. A session
// is set up, IO wired to the connection object's channels, environment
// pushed, then execution & wait.
func (conn *sshConn) runCommandSession(s *SshSess) {
	rawSess, err := conn.ssh.NewSession()
	if err != nil {
		// maybe not fatal in the future?
		log.Fatal("[", conn.address, "] session creation failed: ", err)
	}
	defer rawSess.Close()

	for k, v := range s.env {
		rawSess.Setenv(k, v)
	}

	stdin, _ := rawSess.StdinPipe()
	go forwardToFd(stdin, s.stdin)

	stdout, _ := rawSess.StdoutPipe()
	go forwardFromFd(stdout, s.stdout)

	stderr, _ := rawSess.StderrPipe()
	go forwardFromFd(stderr, s.stderr)

	if err := rawSess.Start(s.command); err != nil {
		log.Fatal("[", conn.address, "] command failed: ", err)
	}

	err = rawSess.Wait()
	if err != nil {
		log.Fatal("[", conn.address, "] ssh session failed: ", err)
	}
	s.done <-true
}

func forwardFromFd(fd io.Reader, to chan []byte) {
	bfd := bufio.NewReader(fd)

	for {
		line, err := bfd.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		to <- line
	}
}

func forwardToFd(fd io.Writer, from chan []byte) {
	for {
		buf := <-from
		sz, err := fd.Write(buf)
		if sz != len(buf) {
			log.Fatal("[stdin] truncated write, sent ", len(buf), " but only wrote ", sz)
		}
		if err == io.EOF {
			break
		}
	}
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
