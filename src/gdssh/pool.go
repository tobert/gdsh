// Copyright 2013 Albert P. Tobey. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gdssh

import (
	"fmt"
	"sync"
	"time"
)

type Task interface {
	Run(conn *Conn) error
}

type Pool struct {
	conns         []*Conn
	lock          sync.Mutex
	messages      chan string // messages related to connection management
	errors        chan error  // connection errors
	done          bool
	MaxRetries    int // maximum retries per connection
	RetryInterval int // seconds
	Retries       int // running total for the pool
}

func NewPool() *Pool {
	return &Pool{
		messages:      make(chan string),
		errors:        make(chan error),
		done:          false,
		MaxRetries:    100,
		RetryInterval: 2,
		Retries:       0,
	}
}

// configure a pool all at once using a map of node:port + user/key (shared)
func (pool *Pool) Configure(list map[string]int, user string, key string) {
	for node, port := range list {
		conn := NewConn(node, port, user, key)
		pool.Add(conn)
	}
}

func (pool *Pool) Add(conn *Conn) {
	pool.lock.Lock()
	pool.conns = append(pool.conns, conn)
	pool.lock.Unlock()
}

func (pool *Pool) msg(format string, a ...interface{}) {
	// TODO: remove this print
	fmt.Printf(format, a)
	fmt.Print("\n")
	pool.messages <- fmt.Sprintf(format, a)
}

func (pool *Pool) err(e error) {
	// TODO: remove this print
	fmt.Printf("ERROR: %s\n", e)
	pool.errors <- e
}

// expected to be run as a goroutine per connection
func monitor(pool *Pool, conn *Conn) {
	secs := pool.RetryInterval // int seconds
	ivl := time.Duration(secs) * time.Second

	time.Sleep(ivl) // wait at least one interval before looping

	ticker := time.NewTicker(ivl)

	// TODO: limit retries & fail
	for {
		select {
		case t, _ := <-ticker.C:
			if !conn.Alive() {
				pool.lock.Lock()
				if next := conn.Started.Add(ivl); next.After(t) {
					err := conn.Reconnect()
					if err != nil {
						pool.err(err)
						pool.msg("Reconnect failed, next retry in %d seconds: %s", secs, err)
					}
				} else {
					pool.msg("Retrying connection to %s in ~%d seconds.", conn.address, secs)
				}
				pool.lock.Unlock()
			}
		case <-conn.done:
			break
		}
	}
}

func (pool *Pool) Start() {
	wg := sync.WaitGroup{}
	for _, conn := range pool.conns {
		var cp = conn // local pointer copy for the goroutine to close over
		wg.Add(1)
		go func() {
			err := cp.Connect()

			if err != nil {
				pool.err(err)
				pool.msg("Connection failed: %s", err)
			}

			// regardless of success/failure
			wg.Done()

			// monitor keeps running
			monitor(pool, cp)
		}()
	}
	wg.Wait()
	fmt.Printf("Connection pool is up!\n")
}

func (pool *Pool) Close() {
	for _, conn := range pool.conns {
		conn.Close()
	}
}

func (pool *Pool) All(task Task) {
	wg := sync.WaitGroup{}
	for _, conn := range pool.conns {
		wg.Add(1)
		go func(c *Conn) {
			task.Run(c)
			wg.Done()
		}(conn)
	}
	fmt.Printf("Waiting for all tasks to run ...\n")
	wg.Wait()
	return
}

func (pool *Pool) AllSerial(task Task) {
	for _, conn := range pool.conns {
		task.Run(conn)
	}
	return
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
