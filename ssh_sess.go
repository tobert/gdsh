// Copyright 2013 Albert P. Tobey. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// references: http://dave.cheney.net/tag/golang

package main

type SshSess struct {
	command string
	env     map[string]string
	stdin   chan []byte
	stdout  chan []byte
	stderr  chan []byte
	stop    chan bool
	done    chan bool
}

// NewSshSession creates a new session object
func NewSshSess(command string, env map[string]string) *SshSess {
	return &SshSess{
		command,
		env,
		make(chan []byte), // stdin
		make(chan []byte), // stdout
		make(chan []byte), // stderr
		make(chan bool),   // stop
		make(chan bool),   // done
	}
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
