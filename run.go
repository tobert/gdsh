package main

import (
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

rm -f $0
exit $EXIT
`

func RunRemote(gdshOpts GdshOptions) int {
	hostname, _ := os.Hostname()
	remoteFile := fmt.Sprintf("/tmp/gdsh-script-%s-%d.sh", hostname, time.Now().Unix())
	conns := NewSshConnMgr(gdshOpts.Key)
	conns.ConnectList(gdshOpts.sshAddressList(), gdshOpts.User, gdshOpts.Key)

	buf := new(bytes.Buffer)

	if gdshOpts.Command != "" {
		t := template.Must(template.New("script").Parse(ScriptTemplate))
		t.Execute(buf, gdshOpts)
	} else if gdshOpts.Script != "" {
		f, err := os.Open(gdshOpts.Script)
		if err != nil {
			log.Fatal("Could not read script file '", gdshOpts.Script, "': ", err)
		}
		io.Copy(buf, f)
		f.Close()
	}

	conns.ScpAll(buf.Bytes(), "0700", remoteFile)
	conns.RunCmdAll(fmt.Sprintf("/bin/bash %s", remoteFile), gdshOpts.Env)

	conns.StopAll()

	return 1
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
