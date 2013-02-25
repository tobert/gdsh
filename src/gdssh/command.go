// Copyright 2013 Albert P. Tobey. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gdssh

import (
	"bufio"
	"code.google.com/p/go.crypto/ssh"
	"io"
)

type SshCmd struct {
	Command string
	Env     map[string]string
	stdin   io.Writer
	stdout  io.Reader
	stderr  io.Reader
	bstdout *bufio.Reader
	bstderr *bufio.Reader
	running bool
	done    chan int
	quit    chan bool
	conn    *SshConn
	session *ssh.Session
}

func Command(conn *SshConn, command string, env map[string]string) *SshCmd {
	return &SshCmd{
		Command: command,
		Env:     env,
		running: false,
		done:    make(chan int),
		quit:    make(chan bool),
	}
}

func (cmd *SshCmd) Start() (err error) {
	sess, err := cmd.conn.client.NewSession()
	if err != nil {
		return
	}
	defer sess.Close()

	for k, v := range cmd.Env {
		sess.Setenv(k, v)
	}

	if cmd.stdin, err = sess.StdinPipe(); err != nil {
		return
	}

	if cmd.stdout, err = sess.StdoutPipe(); err != nil {
		return
	}

	if cmd.stderr, err = sess.StderrPipe(); err != nil {
		return
	}

	cmd.bstdout = bufio.NewReader(cmd.stdout)
	cmd.bstderr = bufio.NewReader(cmd.stderr)

	if err = sess.Start(cmd.Command); err != nil {
		return
	}

	cmd.session = sess
	cmd.running = true
	return nil
}

func (cmd *SshCmd) Running() bool {
	return cmd.running
}

func (cmd *SshCmd) Wait() (rc int) {
	cmd.quit <- true
	return <-cmd.done
}

func (cmd *SshCmd) Run() (rc int, err error) {
	// TODO: get rc for return
	if err = cmd.Start(); err != nil {
		return
	}
	rc = cmd.Wait()
	return
}

func (cmd *SshCmd) Signal(sig ssh.Signal) error {
	return cmd.session.Signal(sig)
}

func (cmd *SshCmd) ReadStdout(p []byte) (int, error) {
	return cmd.bstdout.Read(p)
}

func (cmd *SshCmd) ReadStringStdout(delim byte) (string, error) {
	return cmd.bstdout.ReadString(delim)
}

func (cmd *SshCmd) ReadStderr(p []byte) (int, error) {
	return cmd.bstderr.Read(p)
}

func (cmd *SshCmd) ReadStringStderr(delim byte) (string, error) {
	return cmd.bstderr.ReadString(delim)
}

func (cmd *SshCmd) WriteStdin(p []byte) (int, error) {
	return cmd.stdin.Write(p)
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
