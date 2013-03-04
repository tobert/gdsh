// Copyright 2013 Albert P. Tobey. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"./src/gdssh"
)

func sshPool(opt GdshOptions) *gdssh.Pool {
	list := hostPortMap(opt.List)
	pool := gdssh.NewPool()
	pool.Configure(list, opt.User, opt.Key)
	pool.Start()
	return pool
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
