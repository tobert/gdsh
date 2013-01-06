package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"text/template"
)

// rather than directly executing commands, push a shell script
// to the remote system so it can handle basic IO redirection without
// forcing a lot of cumbersome escaping on the source system
var ScriptTemplate string = `#!/bin/bash

cd /
export DEBIAN_FRONTEND=noninteractive
EXIT=0

{{range $k, $v := .Env}}
export {{$k}}={{$v}}
{{end}}

{{if .BgJob}}
	{{if .RemoteLog}}
nohup {{.Command}} 2>&1 >{{.RemoteLog}} &
	{{else}}
nohup {{.Command}} 2>&1 >/dev/null &
	{{end}}
{{else}}
	{{if .RemoteLog}}
{{.Command}} 2>&1 >{{.RemoteLog}}
	{{else}}
{{.Command}}
	{{end}}
{{end}}
EXIT=$?

rm -f {{.RemoteScript}}
exit $EXIT
`

func runRemoteScript(localScriptPath string) (err error) {
	return nil
}

func generateCommandScript(gdshOpts GdshOptions) (localScriptPath string) {
	hostname, _ := os.Hostname()
	tmp, err := ioutil.TempFile("", fmt.Sprintf("%s-", hostname))
	if err != nil {
		log.Fatal("Could not create a tempfile for command script: %s", err)
	}
	defer tmp.Close()

	if gdshOpts.RemoteScript == "" {
		gdshOpts.RemoteScript = tmp.Name()
	}

	log.Printf("LSP: '%s', C: '%s'", gdshOpts.Script, gdshOpts.Command)

	if gdshOpts.Script == "" && gdshOpts.Command != "" {
		t := template.Must(template.New("script").Parse(ScriptTemplate))
		t.Execute(tmp, gdshOpts)
		tmp.Sync()
	}

	return
}

func RunRemote(gdshOpts GdshOptions) int {
	conns := NewSshConnMgr(gdshOpts.Key)
	conns.ConnectList(gdshOpts.sshAddressList(), gdshOpts.User, gdshOpts.Key)
	log.Printf("connected\n")

	generateCommandScript(gdshOpts)

	//conns["localhost"].command <- gdshOpts.Command
	//test := <-conns["localhost"].stdout
	log.Printf("OK!: %s", conns)
	if gdshOpts.Command != "" {
		conns.RunCmdAll(SshCmd{gdshOpts.Command, gdshOpts.Env})
	}

	conns.StopAll()
	return 1
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
