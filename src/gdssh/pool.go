// Copyright 2013 Albert P. Tobey. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gdssh

import (
	"sync"
)

type SshFun func(conn *SshConn)

type SshPool struct {
	conns []*SshConn
}

func (pool *SshPool) Add(conn *SshConn) {
	pool.conns = append(pool.conns, conn)
}

func (pool *SshPool) All(fun SshFun) {
	wg := sync.WaitGroup{}
	for _, conn := range pool.conns {
		wg.Add(1)
		go func() {
			fun(conn)
			wg.Done()
		}()
	}
	wg.Wait()
}

func (pool *SshPool) AllSerial(fun SshFun) {
	for _, conn := range pool.conns {
		fun(conn)
	}
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
