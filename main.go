package main

import (
	"fmt"
	"log"
	"os"
	"path"
)

func supportedCommand(command string) bool {
	switch command {
	case "nssh":
		return true
	case "run":
		return true
	}
	return false
}

func main() {
	var command string

	if cmd := path.Base(os.Args[0]); supportedCommand(cmd) {
		command = cmd
	} else if len(os.Args) > 1 && supportedCommand(os.Args[1]) {
		command = os.Args[1]
	} else {
		log.Fatal(fmt.Sprintf("unsupported call to gdsh: %s\n", os.Args))
	}

	switch command {
	case "nssh":
		NamedScreenSSHWrapper()
	case "run":
		RunRemote()
	default:
		log.Fatal(fmt.Sprintf("unsupported call to gdsh: %s\n", os.Args))
	}
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
