package main

import (
	"fmt"
	"io/ioutil"
	torrent_kad "torrent-kad"
)

func main() {
	//var client torrent_kad.Client
	//client.Node=common.NewNode(1000)
	//wg:=sync.WaitGroup{}
	//client.Node.Run(&wg)
	//client.Node.Create()
	//client.PutFile("src/chord/chordNode.go")
	torrent, _ := ioutil.ReadFile("src.torrent")
	decoder := torrent_kad.NewDecoder(string(torrent))
	fmt.Println(decoder.Get())
}
