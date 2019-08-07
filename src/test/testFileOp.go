package main

import (
	"common"
	"fmt"
	"io/ioutil"
	torrent_kad "torrent-kad"
)

func main() {
	var client torrent_kad.Client
	client.Node = common.NewNode(1000)

	client.Node.Run()

	client.Node.Create()
	client.Joined = true
	client.PutFile("test_cache/IdeaProjects")

	torrent, _ := ioutil.ReadFile("IdeaProjects.torrent")
	decoder := torrent_kad.NewDecoder(string(torrent))
	fmt.Println(decoder.Get())
}
