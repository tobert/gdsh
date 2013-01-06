// Copyright 2013 Albert P. Tobey. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"
)

// UserIdToUsername parses /etc/passwd and return the username
// os.user has utilities to do this, but the require libc and
// I'd rather not pull in all of libc just for getpwent
func UserIdToUsername(id int) (username string) {
	fd, err := os.Open("/etc/passwd")
	if err != nil {
		return
	}
	buf := bufio.NewReader(fd)
	defer fd.Close()

	line, err := buf.ReadString('\n')
	for err != io.EOF {
		parts := strings.Split(line, ":")
		uid, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}
		if uid == id {
			return parts[0]
		}
		line, err = buf.ReadString('\n')
	}
	return ""
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
