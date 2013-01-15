// Copyright 2013 Albert P. Tobey. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// references: http://dave.cheney.net/tag/golang

package main

import (
	"bufio"
	"code.google.com/p/go.crypto/ssh"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
)

// SshCmd bundles a command and its environment together over channels
type SshCmd struct {
	command string
	env     map[string]string
}

// SshConn is an ssh connection and channels for communicating with it,
// IO pipes are proxied through the channels, so to write to
// a process's stdin, write to the stdin channel in this struct
type sshConn struct {
	address string
	ssh     *ssh.ClientConn
	command chan SshCmd
	stdin   chan []byte
	stdout  chan []byte
	stderr  chan []byte
	ready   chan bool
	done    chan bool
}

// SshConnMgr represents a collection of ssh connections
type SshConnMgr struct {
	conns map[string]*sshConn
	auth  *[]ssh.ClientAuth
}

// NewSshConn returns a new sshConn with channels pre-made
func NewSshConn(address string, ssh *ssh.ClientConn) *sshConn {
	return &sshConn{
		address,           // address
		ssh,               // ssh
		make(chan SshCmd), // command
		make(chan []byte), // stdin
		make(chan []byte), // stdout
		make(chan []byte), // stderr
		make(chan bool),   // ready
		make(chan bool),   // done
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
		conn.command <- command
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
	conn.command <- command
	<-conn.done
}

// ScpToAll scp's a file to all nodes in the manager
func (mgr *SshConnMgr) ScpAll(local string, remote string) {
	wg := sync.WaitGroup{}
	for _, conn := range mgr.conns {
		wg.Add(1)
		go func() {
			fmt.Print("starting!\n")
			mgr.scpTo(conn, local, remote)
			wg.Done()
			fmt.Print("done!\n")
		}()
	}
	wg.Wait()
}

// scp a file to one node on the given connection
func (mgr *SshConnMgr) scpTo(conn *sshConn, local string, remote string) {
	scp := SshCmd{fmt.Sprintf("/usr/bin/scp -t %s", remote), map[string]string{}}
	stat, _ := os.Stat(local)
	scpCommand := fmt.Sprintf("scp: C0500 %d %s\n", stat.Size(), remote)

	conn.command <- scp

	conn.stdin <- []byte(scpCommand)
	log.Printf("pushed %s to remote stdin\n", scpCommand)

	fd, err := os.Open(local)
	if err != nil {
		log.Printf("Failed to open local file '%s' for read: %s\n", local, err)
		return
	}
	defer fd.Close()

	var buf []byte
	sz, err := fd.Read(buf)
	for {
		if sz == 0 || err == io.EOF {
			break
		} else if err != nil {
			log.Fatal("file read failed after ", sz, " bytes: ", err)
		}
		fmt.Print("Pushing data to connection stdin ...")
		conn.stdin <- buf[0:sz]
		sz, err = fd.Read(buf)
	}
	fmt.Print("--- 5\n")
}

// stopAll sends a 'done' message to each connections' goroutine so
// it will exit cleanly
func (mgr *SshConnMgr) StopAll() {
	for name, conn := range mgr.conns {
		log.Printf("[%s] sending done message ...\n", name)
		conn.done <- true
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
	wg := sync.WaitGroup{}

	for _, address := range list {
		conn := NewSshConn(address, nil)

		config := &ssh.ClientConfig{
			User: user,
			Auth: *mgr.auth,
		}

		wg.Add(1)
		go conn.handleConnection(config, &wg)

		mgr.conns[address] = conn
		conns = append(conns, conn)
	}

	wg.Wait()
}

// handleConnection actually connects to the remote machine then sends
// a "ready" on the ready channel so this can be run in goroutines
// and safely waited upon after firing off a bunch of them
func (conn *sshConn) handleConnection(config *ssh.ClientConfig, wg *sync.WaitGroup) {
	var err error
	conn.ssh, err = ssh.Dial("tcp", conn.address, config)
	if err != nil {
		log.Fatal("ssh connection to ", conn.address, " failed: ", err)
	}

	wg.Done()

	for {
		log.Printf("going to wait [%s]\n", conn.address)

		select {
		case command, _ := <-conn.command:
			go conn.runCommandSession(command)
			<-conn.ready // wait for complete connection setup
		case <-conn.done:
			log.Print("All done!")
			conn.done <- true
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

	for k, v := range command.env {
		sess.Setenv(k, v)
	}

	stdin, _ := sess.StdinPipe()
	go forwardToFd(stdin, conn.stdin)

	stdout, _ := sess.StdoutPipe()
	go forwardFromFd(stdout, conn.stdout, "stdout")

	stderr, _ := sess.StderrPipe()
	go forwardFromFd(stderr, conn.stderr, "stderr")

	if err := sess.Start(command.command); err != nil {
		log.Fatal("[", conn.address, "] command failed: ", err)
	} else {
		log.Printf("[%s] executed command: %v %v\n", conn.address, command, sess)
	}

	conn.ready <-true
	sess.Wait()
	conn.done <-true
}

func forwardFromFd(fd io.Reader, to chan []byte, name string) {
	bfd := bufio.NewReader(fd)

	for {
		line, err := bfd.ReadBytes('\n')
		if err == io.EOF {
			log.Printf("[%s] EOF\n", name)
			return
		}
		log.Printf("[%s] %s", name, line)
		to <- line
	}
}

func forwardToFd(fd io.Writer, from chan []byte) {
	for {
		stdin := <-from
		log.Printf("[stdin]: %s", stdin)
		sz, err := fd.Write([]byte(stdin))
		if sz != len(stdin) {
			log.Fatal("[stdin] truncated write")
		}

		if err == io.EOF {
			log.Print("[stdin] EOF\n")
			break
		}
	}
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
