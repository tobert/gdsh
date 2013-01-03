package main

import (
	"os"
	"syscall"
)

func execLocalCommand(command string, argv []string) (err error) {
	return syscall.Exec(command, argv, os.Environ())
}

func runRemoteCommand(command []string) (err error) {
	return nil
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
