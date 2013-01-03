package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Node struct {
	hostname, comment string
	rank              int
}

func LoadMachineListByName(name string) (node_list []Node) {
	machine_lists, _ := ListMachineLists()

	for _, mlist := range machine_lists {
		fmt.Printf("mlist: %s\n", mlist)
	}

	return
}

func ListMachineLists() (lists []string, err error) {
	gdshd := path.Join(os.Getenv("HOME"), ".gdsh")
	stat, err := os.Stat(gdshd)

	// usually because it doesn't exist, just return an empty list and the error
	if err != nil {
		return
	}

	// unusual, but possible if somebody creates a file called ~/.gdsh, which is wrong
	if !stat.IsDir() {
		return
	}

	visitor := func(path string, f os.FileInfo, err error) error {
		// ModeType is a poorly named mask meaning "not a special file", so assume it's a file
		if f.Mode()&os.ModeType != 0 {
			return nil
		}
		// TODO: maybe check /^machines\./ like the old stuff? or just read everything in that directory?

		lists = append(lists, path)

		return nil
	}

	err = filepath.Walk(gdshd, visitor)
	return lists, err
}

func ReadMachineList(path string) (node_list []Node, err error) {
	fd, err := os.Open(path)
	if err != nil {
		return
	}
	buf := bufio.NewReader(fd)
	defer fd.Close()

	line, err := buf.ReadString('\n')
	line_no := 1
	for err != io.EOF {
		var node Node

		node.rank = line_no
		parts := strings.SplitN(strings.Trim(line, "\n"), "#", 2)
		node.hostname = strings.Trim(parts[0], " ")

		if len(parts) == 2 {
			node.comment = strings.Trim(parts[1], " ")
		}

		node_list = append(node_list, node)

		line, err = buf.ReadString('\n')
		line_no++
	}

	return node_list, err
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
