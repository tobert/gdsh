package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

type nsshOptions struct {
	dsh_list, hostname, comment string
	ssh_args                    []string
	nssh_next, nssh_reset       bool
}

// public, will be serialized as json
type NsshPlaceholder struct {
	Dsh_list, Node string
}
type NsshPlaceholderList []NsshPlaceholder

func nsshPlaceholderFile() string {
	return path.Join(os.Getenv("HOME"), ".gdsh", ".nssh-next.json")
}

func loadNsshPlaceholders() (ph NsshPlaceholderList, err error) {
	jsonBytes, err := ioutil.ReadFile(nsshPlaceholderFile())
	// TODO: check for missing file and return an initialized, empty
	// data structure instead
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsonBytes, &ph)
	return
}

func saveNsshPlaceholders(ph NsshPlaceholderList) {
	jsonBytes, err := json.Marshal(&ph)
	if err == nil {
		ioutil.WriteFile(nsshPlaceholderFile(), jsonBytes, 0644)
	}
	log.Fatal("BUG: could not persist placeholder data: ", err)
}

func nextNsshNode(dshListName string) (node Node) {
	dshList := LoadNodeListByName(dshListName)

	placeholders, err := loadNsshPlaceholders()
	if err != nil {
		log.Fatal("getting next node from list failed: %s", err)
	}

	for _, ph := range placeholders {
		if ph.Dsh_list == dshListName {
			for i, n := range dshList {
				if n.hostname == ph.Node {
					// already at last node in list
					if i == len(dshList)-1 {
						log.Fatal("Already at last node in list '%s'\n", dshListName)
					} else {
						ph.Node = dshList[i+1].hostname
						saveNsshPlaceholders(placeholders)
						return dshList[i+1]
					}
				}
			}
		}
	}

	// no placeholder found, create one
	var ph = NsshPlaceholder{dshListName, dshList[0].hostname}
	saveNsshPlaceholders(append(placeholders, ph))

	return dshList[0]
}

func resetNsshNode(dshListName string) {
}

// parse command line options, recognizing common ssh options and saving them
// while extracting nssh's options and, hopefully, the hostname that can be
// infixed, e.g. ssh foobar.com -l tobert
func ParseOptions() (opts nsshOptions) {
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
	for i, arg := range os.Args {
		if skip_next {
			skip_next = false
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
			opts.ssh_args = append(opts.ssh_args, "-o", fmt.Sprintf("User %s", parts[0]))
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

	if opts.nssh_next {
		next := nextNsshNode(opts.dsh_list)
		opts.hostname = next.hostname
		opts.comment = next.comment
		log.Printf("previous: %s", next.hostname)
	} else if opts.nssh_reset {
		resetNsshNode(opts.dsh_list)
		return
	}

	if opts.comment != "" {
		fmt.Printf("\033k%s [%s]\033\\", opts.hostname, opts.comment)
	} else {
		fmt.Printf("\033k%s\033\\", opts.hostname)
	}

	if len(opts.ssh_args) == 0 && opts.hostname == "" {
		log.Fatal("not enough arguments for ssh\n")
	}

	ssh, err := exec.LookPath("ssh")
	if err != nil {
		panic("Cannot locate the ssh binary.")
	}

	execLocalCommand(ssh, append(opts.ssh_args, opts.hostname))
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
