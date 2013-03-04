## WARNING

I only just started on rewriting some ancient perl, ruby, and shell in Go for a practice project to
pick up the basics of the language & libraries. This probably isn't terribly useful yet.

Original is here: https://github.com/tobert/perl-ssh-tools

## Overview

gdsh is a set of tools for managing clusters over ssh. It's a couple generations removed from "dsh"
and is written Go this time around. You are free to substitute whatever you want for the acronym, but
the politically correct expansion is "Go Distributed SHell".

### Configuration

Cluster node lists are simple text files with one hostname or IP per line. Each node may optionally
have a comment after a hash mark. The comment is used in places where it's sensible to show it, otherwise
it's ignored.

### Tools

gdsh builds as a single multi-call binary. It can be executed as "gdsh [subcommand] [args]" or
via links/symlinks to the binary such as "ln -s gdsh ~/bin/nssh" in which case it will automatically
select the subcommand.

#### nssh

This is a simple ssh command wrapper that parses the arguments just enough to grab the hostname
and set your screen title with escape codes. It's really handy if you use a lot of screens and
ctrl-a " to list them. Patches to support tmux or xterm titles are welcome.

#### run

Run the given command or script on all the remote servers. Commands (-c) will be run from a shell wrapper
that is pushed first. Scripts (-s) are run unmodified.

    gdsh run --list default -c "sudo systemctl restart sshd.service"

#### push

Push a file to all servers in the list.

    gdsh push --list default -L /etc/sysctl.conf -R /tmp/sysctl.conf
    gdsh run --list default -c "sudo cp /tmp/sysctl.conf /etc/sysctl.conf"

#### pull

Gets files from remote servers and stashes them locally, creating a directory per remote server.

    gdsh pull --list default -L /tmp -R /etc/resolv.conf
    ls -l /tmp/*/resolv.conf

#### in progress

serial mode

rsync (or similar), ping, local, psgrep, pkill

More advanced, probably much later:

netstat, sysstat

## TESTING

Add a few aliases to your local host in /etc/hosts then use those names in your gdsh lists.

In /etc/hosts:
    # keep a backup, you can mess things up badly if this file breaks
    127.0.0.1 localhost.localdomain	localhost h1 h2 h3
    ::1 localhost.localdomain	localhost h1 h2 h3

In ~/.gdsh/nodes.default:
    localhost
    h1
    h2
    h3

Then try it:
    go build && ./gdsh run -c uptime

## LICENSE

Copyright 2013 Albert P Tobey.  All rights reserved.
Use of this source code is governed by a BSD-style
license that can be found in the LICENSE file.
