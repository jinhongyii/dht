package main

import (
	"os"
	"strconv"
	torrent_kad "torrent-kad"
)

func main() {
	var init_client = new(torrent_kad.Client)
	init_port := 2000
	init_client.Init(init_port)
	init_client.Node.Create()
	clients := make([]torrent_kad.Client, 20)
	for i := 1; i <= 2; i++ {
		clients[i].Init(init_port + i)
	}
	var magnetLink string
	for i := 1; i <= 1; i++ {
		magnetLink, _ = clients[i].PutFile("c:/Users/jinho/Pictures")
	}
	for i := 2; i <= 2; i++ {
		os.MkdirAll("../../test_cache/"+strconv.Itoa(i), 0666)
		clients[i].GetFile(magnetLink, "../../test_cache/"+strconv.Itoa(i))
	}
}
