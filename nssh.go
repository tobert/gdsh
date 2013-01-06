package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// parse command line options, recognizing common ssh options and saving them
// while extracting nssh's options and, hopefully, the hostname that can be
// infixed, e.g. ssh foobar.com -l tobert
func parseNsshOptions(gdshOpts GdshOptions) (ssh_args []string) {
	// we don't actually care about what the ssh options mean, they only
	// need to be recognized and eliminated so the hostname can be found
	no_arg_re, err := regexp.Compile("^-[1246AaCfgKkMNnqsTtVvXxY]$")
	if err != nil {
		panic("BUG: regular expression compilation failed!")
	}
	has_arg_re, err := regexp.Compile("^-[bcDeFiLlmOopRSw]$")
	if err != nil {
		panic("BUG: regular expression compilation failed!")
	}

	// user@hostname is a definite match for hostname, split & replace with -o options
	user_at_re, err := regexp.Compile("^[-a-z0-9]+@[-.a-zA-Z0-9]+$")
	if err != nil {
		panic("BUG: regular expression compilation failed!")
	}

	skip_next := true
	for i, arg := range gdshOpts.Args {
		if skip_next {
			skip_next = false
			continue
		} else if no_arg_re.MatchString(arg) {
			ssh_args = append(ssh_args, arg)
		} else if has_arg_re.MatchString(arg) {
			ssh_args = append(ssh_args, arg, os.Args[i+1])
			skip_next = true
		} else if user_at_re.MatchString(arg) {
			parts := strings.SplitN(arg, "@", 2)
			ssh_args = append(ssh_args, "-o", fmt.Sprintf("User %s", parts[0]))
			gdshOpts.Node = parts[1]
		} else {
			// all that remains at this point is the hostname or some new ssh
			// option not accounted for in the above REs
			gdshOpts.Node = arg
		}
	}

	return ssh_args
}

// doesn't try to detect the terminal or anything, just print the escape code
func NamedScreenSSHWrapper(gdshOpts GdshOptions) int {
	ssh_args := parseNsshOptions(gdshOpts)

	if gdshOpts.Node != "" {
		updatePlaceholder(gdshOpts.List, gdshOpts.Node)
	}
	if gdshOpts.Next {
		next := nextNode(gdshOpts.List)
		gdshOpts.Node = next.Address
		gdshOpts.Comment = next.Comment
	} else if gdshOpts.Reset {
		resetNode(gdshOpts.List)
		return 0
	}

	if gdshOpts.Comment != "" {
		nameScreen(fmt.Sprintf("%s [%s]",gdshOpts.Node, gdshOpts.Comment))
	} else {
		nameScreen(gdshOpts.Node)
	}

	if len(ssh_args) == 0 && gdshOpts.Node == "" {
		log.Fatal("not enough arguments for ssh\n")
	}

	ssh, err := exec.LookPath("ssh")
	if err != nil {
		panic("Cannot locate the ssh binary.")
	}

	execLocalCommand(ssh, append(ssh_args, gdshOpts.Node))
	return 1 // should never happen
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
