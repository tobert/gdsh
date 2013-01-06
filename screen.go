package main

import "fmt"

func nameScreen(screenName string) {
	fmt.Printf("\033k%s\033\\", screenName)
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
