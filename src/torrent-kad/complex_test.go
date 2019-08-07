package torrent_kad

import (
	_ "net/http/pprof"
	"os"
	"strconv"
	"testing"
)

func BenchmarkComprehensive(b *testing.B) {
	var init_client = new(Client)
	init_port := 2000
	init_client.Init(init_port)
	init_client.Node.Create()
	clients := make([]Client, 21)
	for i := 1; i <= 20; i++ {
		clients[i].Init(init_port + i)
	}
	var magnetLink string
	for i := 1; i <= 5; i++ {
		magnetLink, _ = clients[i].PutFile("../../test_cache/src")
	}
	for i := 6; i <= 10; i++ {
		os.MkdirAll("../../test_cache/"+strconv.Itoa(i), 0666)
		clients[i].GetFile(magnetLink, "../../test_cache/"+strconv.Itoa(i))
	}
	for i := 1; i <= 5; i++ {
		clients[i].Node.Quit()
		os.MkdirAll("../../test_cache/"+strconv.Itoa(i+10), 0666)
		clients[i+10].GetFile(magnetLink, "../../test_cache/"+strconv.Itoa(i+10))
		clients[i].Joined = false
	}
	for i := 1; i <= 5; i++ {
		clients[i].Init(init_port + i)
		os.MkdirAll("../../test_cache/"+strconv.Itoa(i), 0666)
		clients[i].GetFile(magnetLink, "../../test_cache/"+strconv.Itoa(i))
	}
}
