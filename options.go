package main
// gdsh option parsing - a custom parser is implemented because the
// few packages I checked out didn't do what I want, or ended up being
// really clunky to do some of the uglier bits of what gdsh needs

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"strings"
)

type GdshOptions struct {
	List         string            // --list/-l
	Next         bool              // --next
	Reset        bool              // --reset
	Verbose      bool              // --verbose/-v
	Comment      string            // --comment
	InclRe       string            // --include
	ExclRe       string            // --exclude
	Key          string            // --key/-i
	User         string            // --user
	Node         string            // --node/-h
	Command      string            // --command/-c
	Script       string            // --script/-s
	BgJob        bool              // --background/-b
	RemoteLog    string            // --remote-log/-r
	RemoteScript string            // --remote-script-path
	Env          map[string]string // --env/-e key=val
	Args         []string          // leftover arguments for subcommands
}

var usageText = `
HELP!
`

func printUsage() {
	fmt.Printf("%s", usageText)
}

func deleteArg(index int) (args []string, deleted string) {
	args = make([]string, len(os.Args)-1)

	offset := 0
	for i := range os.Args {
		if i == index {
			deleted = os.Args[i]
			offset = 1
			continue
		}
		args[i-offset] = os.Args[i]
	}

	return
}

func parseArgs(args []string, command string) (opt GdshOptions) {
	env := map[string]string{}

	// defaults
	opt = GdshOptions{
		List:         "default",
		Next:         false,
		Reset:        false,
		Verbose:      false,
		Comment:      "",
		InclRe:       "",
		ExclRe:       "",
		Key:          "",
		Node:         "",
		Command:      "",
		Script:       "",
		BgJob:        false,
		RemoteLog:    "",
		RemoteScript: "",
		Env:          env,
	}

	user, err := user.Current()
	if err == nil {
		opt.User = user.Username
	}

	skip := true
	cont := false
	for i, arg := range args {
		if skip {
			skip = false
			continue
		}

		switch arg {
		case "--list", "-l":
			opt.List = args[i+1]
			skip = true
		case "next", "--next":
			opt.Next = true
			cont = true
		case "reset", "--reset":
			opt.Reset = true
			cont = true
		case "--comment":
			opt.Comment = args[i+1]
			skip = true
		case "--verbose", "-v":
			opt.Verbose = true
			cont = true
		case "--key", "-i":
			opt.Key = args[i+1]
			skip = true
		case "--node", "-n":
			opt.Node = args[i+1]
			skip = true
		case "--incl":
			opt.InclRe = args[i+1]
			skip = true
		case "--excl":
			opt.ExclRe = args[i+1]
			skip = true
		case "--root":
			opt.User = "root"
			cont = true
		case "--user":
			opt.User = args[i+1]
			skip = true
		case "--help":
			printUsage()
			os.Exit(0)
		case "--env", "-e":
			parts := strings.SplitN(strings.Trim(args[i+1], " \t"), "=", 2)
			env[parts[0]] = parts[1]
			skip = true
		}

		switch command {
		case "run":
			switch arg {
			case "--command", "-c":
				opt.Command = args[i+1]
				skip = true
			case "--script", "-s":
				opt.Script = args[i+1]
				skip = true
			case "--background":
				opt.BgJob = true
				cont = true
			}
		}

		if skip || cont {
			continue
		}

		opt.Args = append(opt.Args, arg)
	}

	switch command {
	case "run":
		if opt.Command != "" && opt.Script != "" {
			log.Fatal("--script/-s and --command/-c are mutually exclusive!")
		}
	}

	//opt.Env = env

	log.Printf("Options: %s\n", opt)

	return
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
