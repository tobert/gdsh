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

#### in progress

run, sendfile, gatherfile, rsync, ping, local, psgrep, pkill

More advanced, probably much later:

netstat, sysstat

