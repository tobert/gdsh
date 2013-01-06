package main

import (
	"log"
	"os"
	"path"
)

var supportedCommands = map[string]func(GdshOptions) int{
	"nssh": NamedScreenSSHWrapper,
	"run":  RunRemote,
}

func main() {
	if fun, ok := supportedCommands[path.Base(os.Args[0])]; ok {
		// multi-call binary
		opt := parseArgs(os.Args, path.Base(os.Args[0]))
		os.Exit(fun(opt))
	} else if len(os.Args) > 1 {
		// gdsh run --args style
		if fun, ok := supportedCommands[os.Args[1]]; ok {
			args, command := deleteArg(1)
			opt := parseArgs(args, command)
			os.Exit(fun(opt))
		}
	}

	log.Fatal("unsupported call to gdsh: %s\n", os.Args)
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
