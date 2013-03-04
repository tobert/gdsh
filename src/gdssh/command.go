// Copyright 2013 Albert P. Tobey. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gdssh

import (
	"bytes"
	"code.google.com/p/go.crypto/ssh"
	"io"
	"log"
)

type SshCmd struct {
	Command string
	Env     map[string]string
	Stdin   chan []byte
	Stdout  chan []byte
	Stderr  chan []byte
	stdin   io.Writer
	stdout  io.Reader
	stderr  io.Reader
	running bool
	conn    *Conn
	session *ssh.Session
}

func (conn *Conn) Command(command string, env map[string]string) *SshCmd {
	return &SshCmd{
		conn:    conn,
		Command: command,
		Env:     env,
		Stdin:   make(chan []byte),
		Stdout:  make(chan []byte),
		Stderr:  make(chan []byte),
		running: false,
	}
}

func (cmd *SshCmd) Start() (err error) {
	sess, err := cmd.conn.client.NewSession()
	if err != nil {
		return
	}

	for k, v := range cmd.Env {
		sess.Setenv(k, v)
	}

	if cmd.stdin, err = sess.StdinPipe(); err != nil {
		log.Println("failed to acquire stdin pipe: %s", err)
		return
	}
	go cmd.fwdStdin()

	if cmd.stdout, err = sess.StdoutPipe(); err != nil {
		log.Println("failed to acquire stdout pipe: %s", err)
		return
	}
	go cmd.fwdStdout()

	if cmd.stderr, err = sess.StderrPipe(); err != nil {
		log.Println("failed to acquire stderr pipe: %s", err)
		return
	}
	go cmd.fwdStderr()

	if err = sess.Start(cmd.Command); err != nil {
		log.Printf("FAILED: '%s': %s\n", cmd.Command, err)
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
	err := cmd.session.Wait()
	cmd.running = false
	cmd.session.Close()
	if err == nil {
		return 0
	}
	// TODO: figure out how to get at ExitError.Waitmsg
	log.Printf("\nExit: %s\n", err)
	return 1
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

func slurp(ch chan []byte) bytes.Buffer {
	var buf bytes.Buffer
	for data := range ch {
		buf.Write(data)
	}
	return buf
}

// do a blocking read of stdout until the channel closes
func (cmd *SshCmd) DrainStdout() bytes.Buffer {
	return slurp(cmd.Stdout)
}

// do a blocking read of stderr until the channel closes
func (cmd *SshCmd) DrainStderr() bytes.Buffer {
	return slurp(cmd.Stderr)
}

func (cmd *SshCmd) fwdStdin() {
	for {
		data, ok := <-cmd.Stdin
		if !ok {
			break // channel closed, all done
		}
		wrote, err := cmd.stdin.Write(data)
		if wrote != len(data) {
			log.Panic("truncated write to stdin! (%d out of %d written)", wrote, len(data))
		}
		if err != nil {
			log.Panic("Failed write: %s", err)
		}
	}
}

func (cmd *SshCmd) fwdStdxxx(rd io.Reader, ch chan []byte, which string) {
	buf := make([]byte, 1024)
	for {
		read, err := rd.Read(buf)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Printf("[%s] got %s on read\n", which, err)
			break
		}
		ch <- buf[0:read]
	}
	close(ch)
}

func (cmd *SshCmd) fwdStdout() {
	cmd.fwdStdxxx(cmd.stdout, cmd.Stdout, "stdout")
}

func (cmd *SshCmd) fwdStderr() {
	cmd.fwdStdxxx(cmd.stderr, cmd.Stderr, "stderr")
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
