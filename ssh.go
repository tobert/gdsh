// Copyright 2013 Albert P. Tobey. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// references: http://dave.cheney.net/tag/golang

package main

import (
	"bufio"
	"code.google.com/p/go.crypto/ssh"
	"io"
	"log"
	"net"
	"os"
)

// SshCmd bundles a command and its environment together over channels
type SshCmd struct {
	command string
	env map[string]string
}

// SshConn is an ssh connection and channels for communicating with it,
// IO pipes are proxied through the channels, so to write to
// a process's stdin, write to the stdin channel in this struct
type sshConn struct {
	address string
	ssh     *ssh.ClientConn
	command chan SshCmd
	stdin   chan string
	stdout  chan string
	stderr  chan string
	ready   chan string
	done    chan string
}

// SshConnMgr represents a collection of ssh connections
type SshConnMgr struct {
	conns map[string]*sshConn
	auth *[]ssh.ClientAuth
}

// NewSshConn returns a new sshConn with channels pre-made
func NewSshConn(address string, ssh *ssh.ClientConn) (*sshConn) {
	return &sshConn{
		address,             // address
		ssh,                 // ssh
		make(chan SshCmd),   // command
		make(chan string),   // stdin
		make(chan string),   // stdout
		make(chan string),   // stderr
		make(chan string),   // ready
		make(chan string),   // done
	}
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

// runCmdAll runs an SshCmd on all nodes connected by the manager. It waits for
// and ack from all of the connections before returning.
// TODO: timeouts
func (mgr *SshConnMgr) RunCmdAll(command SshCmd) {
	for _, conn := range mgr.conns {
		conn.command <-command
	}
	for _, conn := range mgr.conns {
		<-conn.done
	}
}

// runCmdOne runs an SshCmd on one node that must already be connected by the
// connection manager
func (mgr *SshConnMgr) RunCmdOne(address string, command SshCmd) {
	log.Printf("[%s] runCmdOne %s\n", address, command)
	conn := mgr.conns[address]
	conn.command <-command
	<-conn.done
}

// stopAll sends a 'done' message to each connections' goroutine so
// it will exit cleanly
func (mgr *SshConnMgr) StopAll() {
	for name, conn := range mgr.conns {
		log.Printf("[%s] sending done message ...\n", name)
		conn.done <- "done"
	}

	// wait for confirmation
	for name, conn := range mgr.conns {
		<-conn.done
		log.Printf("[%s] done message acked. Done!\n", name)
	}
}

// connectList tells the connection manager to connect to all nodes in the
// provided list of addresses
// expects a list of ssh addresses in address:port format
func (mgr *SshConnMgr) ConnectList(list []string, user string, key string) {
	conns := []*sshConn{}

	for _, address := range list {
		conn := NewSshConn(address, nil)

		config := &ssh.ClientConfig{
			User: user,
			Auth: *mgr.auth,
		}

		go conn.handleConnection(config)

		mgr.conns[address] = conn
		conns = append(conns, conn)
	}

	// now wait for them all to become ready, use a local list
	// to allow connectList() to be called multiple times safely
	for _, conn := range conns {
		<-conn.ready
	}
}

// handleConnection actually connects to the remote machine then sends
// a "ready" on the ready channel so this can be run in goroutines
// and safely waited upon after firing off a bunch of them
func (conn *sshConn) handleConnection(config *ssh.ClientConfig) {
	ssh, err := ssh.Dial("tcp", conn.address, config)
	if err != nil {
		log.Fatal("ssh connection to ", conn.address, " failed: ", err)
	}

	conn.ssh = ssh
	conn.ready <-"ready"

	for {
		log.Printf("going to wait [%s]\n", conn.address)

		select {
		case command, _ := <-conn.command:
			conn.runCommandSession(command)
		case <-conn.done:
			conn.done <-"ack"
			break
		}
	}
}

// runCommandSession runs a single command on the remote host. A session
// is set up, IO wired to the connection object's channels, environment
// pushed, then execution & wait.
func (conn *sshConn) runCommandSession(command SshCmd) {
	sess, err := conn.ssh.NewSession()
	if err != nil {
		// maybe not fatal in the future?
		log.Fatal("[", conn.address, "] session creation failed: ", err)
	}
	defer sess.Close()
	log.Printf("session started\n")

	for k, v := range command.env {
		sess.Setenv(k, v)
	}
	log.Printf("session environment set\n")

	stdin, _ := sess.StdinPipe()
	go forwardToFd(stdin, conn.stdin)
	defer stdin.Close()
	log.Printf("stdin forwarder started\n")

	stdout, _ := sess.StdoutPipe()
	go forwardFromFd(stdout, conn.stdout, "stdout")
	log.Printf("stdout forwarder started\n")

	stderr, _ := sess.StderrPipe()
	go forwardFromFd(stderr, conn.stderr, "stderr")
	log.Printf("stderr forwarder started\n")


	if err := sess.Start(command.command); err != nil {
		log.Fatal("[", conn.address, "] command failed: ", err)
	} else {
		log.Printf("[%s] executed command: %v %v\n", conn.address, command, sess)
	}

	sess.Wait()
	conn.done <-command.command
}

func forwardFromFd(fd io.Reader, to chan string, name string) {
	bfd := bufio.NewReader(fd)
	log.Printf("[%s] forwarding\n", name)

	for {
		line, err := bfd.ReadString('\n')
		if err == io.EOF {
			log.Printf("[%s] EOF\n", name)
			return
		}
		log.Printf("[%s] %s", name, line)
		to <- string(line)
	}
}

func forwardToFd(fd io.Writer, from chan string) {
	for {
		stdin := <-from
		fd.Write([]byte(stdin))
	}
}

/*
	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()
	}()
	if err := session.Run("/usr/bin/scp -qrt ./"); err != nil {
		panic("Failed to run: " + err.Error())
	}
}

*/

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
