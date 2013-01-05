package main

import (
	"github.com/ogier/pflag"
	"os"
)

type GdshOptions struct {
	List     string   // --list
	Next     bool     // --next
	Reset    bool     // --reset
	Verbose  bool     // --verbose
	RootUser bool     // --root
	Screen   bool     // --screen/-s
	Comment  string   // --comment <string>
	InclRe   string   // --include <regular expression>
	ExclRe   string   // --exclude <regular expression>
	Key      string   // --key <path>
	User     string   // --user <username>
	Node     string   // --node <hostname>
	Args     []string // leftover arguments for subcommands
}

func deleteArg(index int) (args []string) {
	args = make([]string, len(os.Args)-1)

	offset := 0
	for i := range os.Args {
		if i == index {
			offset = 1
			continue
		}
		args[i-offset] = os.Args[i]
	}

	return
}

func parseCommonArgs(args []string) (opt GdshOptions) {
	fs := pflag.NewFlagSet("main", pflag.ExitOnError)

	fs.StringVarP(&opt.List, "list", "l", "default", "node list")
	fs.StringVar(&opt.InclRe, "incl", "", "include only nodes that match the RE")
	fs.StringVar(&opt.ExclRe, "excl", "", "exclude any nodes that match the RE")
	fs.StringVarP(&opt.Key, "key", "i", "", "path to an ssh private key to use")
	fs.StringVar(&opt.User, "user", "", "remote username to log in as")
	fs.StringVarP(&opt.Comment, "comment", "c", "", "comment field")
	fs.StringVarP(&opt.Node, "node", "h", "", "select a node")
	fs.BoolVar(&opt.Next, "next", false, "advance to the next node in the node list")
	fs.BoolVar(&opt.Reset, "reset", false, "reset to the top of the node list")
	fs.BoolVarP(&opt.Verbose, "verbose", "v", false, "print verbose output")
	fs.BoolVar(&opt.RootUser, "root", false, "shorthand for '--user root'")
	fs.BoolVarP(&opt.Screen, "screen", "s", false, "set the screen title")

	fs.Parse(args)

	opt.Args = fs.Args()

	return
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
