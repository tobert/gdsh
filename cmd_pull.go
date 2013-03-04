// Copyright 2013 Albert P. Tobey. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"./src/gdssh"
	"fmt"
	"os"
	"path"
)

type pullTask struct {
	local  string
	remote string
}

func parsePullOptions(opt GdshOptions) *pullTask {
	// STUB: parse options, check for directory
	fmt.Println("WARNING: this is a stub and will write to your /tmp!")
	task := pullTask{local: "/tmp", remote: "/etc/hosts"}
	return &task
}

func (task *pullTask) Run(conn *gdssh.Conn) error {
	bname := path.Base(task.remote)
	hostpath := path.Join(task.local, conn.Host)
	os.Mkdir(hostpath, 0755)
	local := path.Join(hostpath, bname)
	conn.ScpPull(local, task.remote)
	return nil
}

func cmdPull(opt GdshOptions) int {
	pool := sshPool(opt)
	task := parsePullOptions(opt)
	pool.All(task)
	pool.Close()

	return 1
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
