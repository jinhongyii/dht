package main

import (
	"fmt"
	"strconv"
	torrent_kad "torrent-kad"
)

func main() {
	var quit = false
	var port = 2000
	var app = new(torrent_kad.Client)
	connected := false
	for !quit {
		var cmd1, cmd2, cmd3, cmd4, cmd5 string
		fmt.Scanln(&cmd1, &cmd2, &cmd3, &cmd4, &cmd5)
		switch cmd1 {
		case "port":
			if !connected {
				port, _ = strconv.Atoi(cmd2)
				fmt.Println("port set to ", port)
				app.Init(port)
			} else {
				fmt.Println("already joined in the network")
			}
		case "quit":
			if connected {
				app.Node.Quit()
				quit = true
			} else {
				quit = true
			}
		case "create":
			if !connected {
				app.Node.Create()
				app.Joined = true
				connected = true
				fmt.Println("create network successful")
			} else {
				fmt.Println("already joined in the network")
			}
		case "share":
			_, success := app.PutFile(cmd2)
			if success {
				fmt.Println("share successful")
			} else {
				fmt.Println("share fail")
			}

		case "getFile":

			if app.GetFile(cmd2, cmd3) {
				fmt.Println("get file successful")
			} else {
				fmt.Println("get file fail")
			}

		case "ls":
		case "help":
		}
	}
}
