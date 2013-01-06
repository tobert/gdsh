package main

// thanks to: http://dave.cheney.net/tag/golang

import (
	"code.google.com/p/go.crypto/ssh"
	"fmt"
	"log"
	"net"
	"os"
)

type sshCommand struct {
	command string
	args    []string
}

type sshConnection struct {
	command chan sshCommand
	stdout  chan string
	stderr  chan string
}

func connectNode(address string, user string, privKey string, conn sshConnection) {
	// TODO: allow setting port, or detect it from the list file
	sshHost := fmt.Sprintf("%s:22", address)
	agentSock := os.Getenv("SSH_AUTH_SOCK")
	auth := []ssh.ClientAuth{}

	// only load a private key if requested ~/.ssh/id_rsa is _not_ loaded automatically
	// ssh-agent should be the usual path
	if privKey != "" {
		kr := new(keyring)
		if err := kr.loadPEM(privKey); err != nil {
			log.Fatal("Couldn't load specified private key '", privKey, "': ", err)
		}
		auth = append(auth, ssh.ClientAuthKeyring(kr))
	}

	// ssh-agent support
	if agentSock != "" {
		sock, err := net.Dial("unix", agentSock)
		if err != nil {
			log.Fatal("Could not connect to SSH_AUTH_SOCK. Is ssh-agent running?")
		}

		agent := ssh.NewAgentClient(sock)
		auth = append(auth, ssh.ClientAuthAgent(agent))
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: auth,
	}

	remote, err := ssh.Dial("tcp", sshHost, config)
	if err != nil {
		log.Fatal("ssh connection to ", sshHost, " failed: ", err)
	}

	for {
		log.Printf("going to wait [%s]\n", sshHost)

		request := <-conn.command

		go func() {
			sess, err := remote.NewSession()
			if err != nil {
				// maybe not fatal in the future?
				log.Fatal("[", sshHost, "] session creation failed: ", err)
			}
			defer sess.Close()

			if err := sess.Run(request.command); err != nil {
				log.Fatal("[", sshHost, "] command failed: ", err)
			} else {
				log.Printf("[%s] executed command: %v %v\n", sshHost, request, sess)
			}
		}()
	}

	msg := fmt.Sprintf("all set: %s %v %s ----------\n", config, remote, err)
	conn.stdout <- msg
}

func connectAll(gdshOptions GdshOptions) map[string]sshConnection {
	conns := map[string]sshConnection{}
	conn := sshConnection{
		make(chan sshCommand),
		make(chan string),
		make(chan string),
	}

	if gdshOptions.Node != "" {
		conns[gdshOptions.Node] = conn
		go connectNode(gdshOptions.Node, gdshOptions.User, gdshOptions.Key, conn)
	} else {
		list := loadListByName(gdshOptions.List)
		for _, node := range list {
			conns[node.Address] = conn
			go connectNode(node.Address, gdshOptions.User, gdshOptions.Key, conn)
		}
	}
	return conns
}

/*
func scp(gdshOptions GdshOptions) {

	client, err := ssh.Dial("tcp", "127.0.0.1:22", clientConfig)
	if err != nil {
		panic("Failed to dial: " + err.Error())
	}
	session, err := client.NewSession()
	if err != nil {
		panic("Failed to create session: " + err.Error())
	}
	defer session.Close()
	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()
		content := "123456789\n"
		fmt.Fprintln(w, "C0644", len(content), "testfile")
		fmt.Fprint(w, content)
		fmt.Fprint(w, "\x00") // ä¼ è¾ä»¥\x00ç»æ
	}()
	if err := session.Run("/usr/bin/scp -qrt ./"); err != nil {
		panic("Failed to run: " + err.Error())
	}
}

*/

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
