package main

import (
	"fmt"
	"strconv"
)

func main() {
	var quit = false
	var port = 1000
	var app *AppClient
	connected := false
	for !quit {
		var cmd1, cmd2, cmd3, cmd4, cmd5 string
		fmt.Scanln(&cmd1, &cmd2, &cmd3, &cmd4, &cmd5)
		switch cmd1 {
		case "port":
			if !connected {
				port, _ = strconv.Atoi(cmd2)
				fmt.Println("port set to ", port)
			} else {
				fmt.Println("already joined in the network")
			}
		case "quit":
			if connected {
				app.node.Quit()
				quit = true
			} else {
				quit = true
			}
		case "create":
			if !connected {
				app = NewClient(port)
				app.node.Create()
				connected = true
				fmt.Println("create network successful")
			} else {
				fmt.Println("already joined in the network")
			}
		case "join":
			if !connected {
				app = NewClient(port)
				if app.node.Join(cmd2) {
					connected = true
					fmt.Println("join ", cmd2, " successful")
				} else {
					fmt.Println("join failed")
					app.node.ForceQuit()
				}
			} else {
				fmt.Println("already joined in the network")
			}
		case "share":
			if connected {
				if app.Share(cmd2) {
					fmt.Println("share successful")
				}
			} else {
				fmt.Println("please create or join the network first")
			}
		case "stopshare":
			if connected {
				if app.StopShare(cmd2) {
					fmt.Println("stop sharing successful")
				}
			} else {
				fmt.Println("please create or join the network first")
			}
		case "getFile":
			if connected {
				if cmd4 == "-s" {
					tmp, _ := strconv.Atoi(cmd5)
					if app.GetFileFromMultipleServer(cmd2, cmd3, tmp) {
						fmt.Println("get file successful")
					}
				} else if app.GetFile(cmd2, cmd3) {
					fmt.Println("get file successful")
				}
			} else {
				fmt.Println("please create or join the network first")
			}
		case "ls":
		case "help":
		}
	}
}
