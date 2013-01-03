package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"os/exec"
)

type NsshOptions struct {
	dsh_list, hostname, comment string
	ssh_args                    []string
	nssh_next, nssh_reset       bool
}

// parse command line options, recognizing common ssh options and saving them
// while extracting nssh's options and, hopefully, the hostname that can be
// infixed, e.g. ssh foobar.com -l tobert
func ParseOptions() (opts NsshOptions) {
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

	skip_next := false
	for i, arg := range os.Args[1:] {
		if skip_next {
			continue
		} else if no_arg_re.MatchString(arg) {
			opts.ssh_args = append(opts.ssh_args, arg)
		} else if has_arg_re.MatchString(arg) {
			opts.ssh_args = append(opts.ssh_args, arg, os.Args[i+1])
			skip_next = true
		} else if arg == "--list" {
			opts.dsh_list = os.Args[i+1]
			skip_next = true
		} else if arg == "--comment" {
			opts.comment = os.Args[i+1]
			skip_next = true
		} else if arg == "--next" {
			opts.nssh_next = true
		} else if arg == "--reset" {
			opts.nssh_reset = true
		} else if user_at_re.MatchString(arg) {
			parts := strings.SplitN(arg, "@", 2)
			opts.ssh_args = append(opts.ssh_args, "-o", fmt.Sprintf("'User %s'", parts[0]))
			opts.hostname = parts[1]
		} else {
			// all that remains at this point is the hostname or some new ssh
			// option not accounted for in the above REs
			opts.hostname = arg
		}
	}

	if opts.hostname == "" && opts.nssh_next {
		if opts.dsh_list == "" {
			opts.dsh_list = "default"
		}
	}

	return opts
}

// doesn't try to detect the terminal or anything, just print the escape code
func NamedScreenSSHWrapper() {
	opts := ParseOptions()

	screen_string := "ERROR(nssh)"
	if opts.nssh_next {
		fmt.Print("Placeholder: --next not implemented yet.\n")
		screen_string = "nssh PLACEHOLDER --next"
	} else if opts.nssh_reset {
		fmt.Print("Placeholder: --reset not implemented yet.\n")
		screen_string = "nssh PLACEHOLDER --reset"
	} else if opts.comment != "" {
		screen_string = fmt.Sprintf("%s [%s]", os.Args[1], opts.comment)
	} else {
		screen_string = os.Args[len(os.Args)-1]
	}

	fmt.Printf("\033k%s\033\\", screen_string)

	ssh, err := exec.LookPath("ssh")
	if err != nil { panic("Cannot locate the ssh binary.") }

	execLocalCommand(ssh, opts.ssh_args)
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
