package main

import (
	torrent_kad "torrent-kad"
)

func main() {
	//var client torrent_kad.Client
	//client.Node=common.NewNode(1000)
	//wg:=sync.WaitGroup{}
	//client.Node.Run(&wg)
	//client.Node.Create()
	//client.PutFile("src/chord/chordNode.go")
	torrent_kad.GenerateTorrentFile("src")
}
