package main

import (
	"os"
	"syscall"
)

func execLocalCommand(command string, args []string) (err error) {
	// the exec syscall expects the command name to be in argv
	argv := make([]string, len(args)+1)
	argv[0] = command
	for i := range args {
		argv[i+1] = args[i]
	}

	return syscall.Exec(command, argv, os.Environ())
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
