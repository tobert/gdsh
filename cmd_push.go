package main

import (
	"./src/gdssh"
	"fmt"
	"log"
	"os"
	"strings"
)

type pushTask struct {
	local  string
	remote string
}

func parsePushOptions(opt GdshOptions) *pushTask {
	task := pushTask{}

	if len(opt.Args) == 2 {
		// bare argument style, e.g. gdsh push /etc/hosts /etc/hosts
		if strings.HasPrefix(opt.Args[0], "-") {
			log.Fatal("Malformed command? Leading dashes are not allowed with bare arguments.")
		}

		if strings.HasPrefix(opt.Args[1], "-") {
			log.Fatal("Malformed command? Leading dashes are not allowed with bare arguments.")
		}

		_, err := os.Stat(opt.Args[0])
		if err != nil {
			log.Fatal(fmt.Sprintf("Not enough arguments. '%s' is not a file: %s", opt.Args[0], err))
		}

		task.local = opt.Args[0]
		task.remote = opt.Args[1]

	} else if len(opt.Args) == 4 {
		// fully-specified style, e.g. gdsh push -L /etc/hosts -R /etc
		skip := false
		for i, arg := range opt.Args {
			if skip {
				skip = false
				continue
			} else if arg == "--local" || arg == "-L" {
				_, err := os.Stat(opt.Args[i+1])
				if err != nil {
					log.Fatal(fmt.Sprintf("Invalid local file '%s': %s", opt.Args[i+1], err))
				}
				task.local = opt.Args[i+1]
				skip = true
			} else if arg == "--remote" || arg == "-R" {
				task.remote = opt.Args[i+1]
				skip = true
			}
		}
	} else {
		log.Fatal("Wrong number of arguments.")
	}

	return &task
}

// the file will be opened for each remote host, but that's fine since
// the reads will end up shared on modern operating systems
func (task *pushTask) Run(conn *gdssh.Conn) error {
	conn.ScpPush(task.local, task.remote)
	return nil
}

func cmdPush(opt GdshOptions) int {
	pool := sshPool(opt)
	task := parsePushOptions(opt)
	pool.All(task)
	pool.Close()

	return 1
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
