package main

import (
	"./src/gdssh"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"text/template"
	"time"
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

#rm -f $0
exit $EXIT
`

// implements gdssh.Task for use with gdssh.Pool.All()
type runTask struct {
	filename string
	script   *bytes.Buffer
	env      map[string]string
}

func (task *runTask) Run(conn *gdssh.Conn) error {
	conn.Scp(task.script.Bytes(), "0555", task.filename)
	cmd := conn.Command(task.filename, task.env)
	rc, err := cmd.Run()
	log.Printf("Run() of %s on host %s returned %d and error '%s'\n", task.filename, conn.Host, rc, err)
	return nil
}

func RunRemote(opt GdshOptions) int {
	pool := sshPool(opt)

	hostname, _ := os.Hostname()
	run := runTask{
		filename: fmt.Sprintf("/tmp/gdsh-script-%s-%d.sh", hostname, time.Now().Unix()),
		script:   new(bytes.Buffer),
		env:      opt.Env,
	}

	if opt.Command != "" {
		t := template.Must(template.New("script").Parse(ScriptTemplate))
		t.Execute(run.script, opt)
	} else if opt.Script != "" {
		f, err := os.Open(opt.Script)
		if err != nil {
			log.Fatal("Could not read script file '", opt.Script, "': ", err)
		}
		io.Copy(run.script, f)
		f.Close()
	}

	fmt.Printf("Gonna run commands ...\n")
	pool.All(&run)
	fmt.Printf("Ran commands ...\n")

	pool.Close()
	fmt.Printf("Closed ssh ...\n")

	return 1
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
