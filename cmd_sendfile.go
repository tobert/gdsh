package main

import (
	"./src/gdssh"
	"fmt"
	"log"
)

type sfTask struct {
	local string
	remote string
}

func parseSfOptions(opt GdshOptions) *sfTask {
	task := sfTask{}

	if len(opt.Args) == 2 {
		task.local = opt.Args[0]
		task.remote = opt.Args[1]
	} else if len(opt.Args) == 4 {
		skip := false
		for i, arg := range opt.Args {
			if skip {
				skip = false
				continue
			} else if arg == "--local" {
				task.local = opt.Args[i+1]
				skip = true
			} else if arg == "--remote" {
				task.remote = opt.Args[i+1]
				skip = true
			}
		}
	} else {
		log.Fatal("Wrong number of arguments.")
	}

	return &task
}

func (task *sfTask) Run(conn *gdssh.Conn) error {
// stat localFile, check for errors
// copy localfile
//conn.Scp(task.script.Bytes(), "0555", task.filename)
	fmt.Printf("scp %s %s:%s\n", task.local, conn.Host, task.remote)
	return nil
}

func cmdSendfile(opt GdshOptions) int {
	//stat, err := os.Stat(opt.)
	pool := sshPool(opt)

	/*	f, err := os.Open(opt.Script)
		if err != nil {
			log.Fatal("Could not read script file '", opt.Script, "': ", err)
		}
		io.Copy(run.script, f)
		f.Close()
	} */

	task := parseSfOptions(opt)
	pool.All(task)
	pool.Close()

	return 1
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
