package main

import (
	"os"
	"fmt"
	"log"
	"io/ioutil"
	"flag"
	"text/template"
)

type runRemoteOptions struct {
	localScriptPath, commandString string
	localLogPath, remoteLogPath string
	maxJobs int
	backgroundRemoteJob bool
	Env map[string] string
}

// rather than directly executing commands, push a shell script
// to the remote system so it can handle basic IO redirection without
// forcing a lot of cumbersome escaping on the source system
var ScriptTemplate string = `#!/bin/bash

export DEBIAN_FRONTEND=noninteractive
EXIT=0

{{range $k, $v := .Env}}
export {{$k}}={{$v}}
{{end}}

{{if .backgroundRemoteJob}}
nohup {{.commandString}} &
{{else}}
{{.commandString}}
{{end}}
EXIT=$?

rm -f {{.remoteScriptPath}}
exit $EXIT
`

func runRemoteScript(localScriptPath string) (err error) {
	return nil
}

func generateCommandScript(options runRemoteOptions) (localScriptPath string) {
	hostname, _ := os.Hostname()
	tmp, err := ioutil.TempFile("", fmt.Sprintf("%s-", hostname))
	if err != nil {
		log.Fatal("Could not create a tempfile for command script: %s", err)
	}
	defer tmp.Close()

	t := template.Must(template.New("script").Parse(ScriptTemplate))
	t.Execute(os.Stdout, options)

	localScriptPath = tmp.Name()
	return
}

func RunRemote() {
	var opt runRemoteOptions
	flag.Parse()

	opt.Env = map[string] string { "test": "foo", "bar": "baz" }

	script := generateCommandScript(opt)
	log.Print(script)
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
