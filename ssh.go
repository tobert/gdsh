package main

// thanks to: http://dave.cheney.net/tag/golang

import (
	"code.google.com/p/go.crypto/ssh"
	"fmt"
	"log"
	"net"
	"os"
)

type sshConnection struct {
	command chan string
	stdin   chan string
	stdout  chan string
	stderr  chan string
}

func NewSshConnection() *sshConnection {
	ret := sshConnection{
		make(chan string),
		make(chan string),
		make(chan string),
		make(chan string),
	}
	return &ret
}

func connectNode(address string, gdshOptions GdshOptions, conn *sshConnection) {
	// TODO: allow setting port, or detect it from the list file
	sshHost := fmt.Sprintf("%s:22", address)
	agentSock := os.Getenv("SSH_AUTH_SOCK")
	auth := []ssh.ClientAuth{}
	privKey := gdshOptions.Key
	user := gdshOptions.User

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

		command := <-conn.command

		go func() {
			sess, err := remote.NewSession()
			if err != nil {
				// maybe not fatal in the future?
				log.Fatal("[", sshHost, "] session creation failed: ", err)
			}
			defer sess.Close()

			for k, v := range gdshOptions.Env {
				sess.Setenv(k, v)
			}

			if err := sess.Run(command); err != nil {
				log.Fatal("[", sshHost, "] command failed: ", err)
			} else {
				log.Printf("[%s] executed command: %v %v\n", sshHost, command, sess)
			}
		}()
	}

	msg := fmt.Sprintf("all set: %s %v %s ----------\n", config, remote, err)
	conn.stdout <- msg
}

func connectAll(gdshOptions GdshOptions) map[string]sshConnection {
	conns := map[string]sshConnection{}
	conn := NewSshConnection()

	if gdshOptions.Node != "" {
		conns[gdshOptions.Node] = *conn
		go connectNode(gdshOptions.Node, gdshOptions, conn)
	} else {
		list := loadListByName(gdshOptions.List)
		for _, node := range list {
			conns[node.Address] = *conn
			go connectNode(node.Address, gdshOptions, conn)
		}
	}
	return conns
}

/*
	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()
	}()
	if err := session.Run("/usr/bin/scp -qrt ./"); err != nil {
		panic("Failed to run: " + err.Error())
	}
}

*/

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
