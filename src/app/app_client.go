package main

import (
	"chord"
	"crypto/sha1"
	"errors"
	"fmt"
	"math/big"
	"net/rpc"
	"os"
	"strings"
	"sync"
)

type AppClient struct {
	node   *chord.Client
	wg     sync.WaitGroup
	server *AppServer
}

func (this *AppClient) Create() {
	this.node.Create()
}
func (this *AppClient) Join(ip string) bool {
	return this.node.Join(ip)

}
func NewClient(port int) *AppClient {
	c := new(AppClient)
	c.server = NewServer()
	c.node = chord.NewNode(port)
	c.node.Server.Register(c.server)
	c.node.Run(&c.wg)
	return c
}

//todo: if the file is too big,we need to split it
func getBytesFromFile(filepath string) ([]byte, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, errors.New("no such file")
	}
	buffer := make([]byte, 4096)
	total := make([]byte, 0)
	for {
		length, e := f.Read(buffer)
		total = append(total, buffer[:length]...)
		if e != nil {
			break
		}
	}
	f.Close()
	return total, nil
}
func hashfile(filePath string) (*big.Int, error) {
	bytes, err := getBytesFromFile(filePath)
	if err != nil {
		return nil, err
	}
	hasher := sha1.New()
	hasher.Write(bytes)
	return new(big.Int).SetBytes(hasher.Sum(nil)), nil
}

func (this *AppClient) Share(filePath string) bool {
	hash, err := hashfile(filePath)
	fmt.Println("hash finish")
	if err != nil {
		fmt.Println(err)
		return false
	}
	//get loc list from remote server
	//check whether the file has been shared before
	shareLocs, success := this.node.Get(hash.Text(16))
	fmt.Println("get loc list finish")
	if success {
		locList := strings.Split(shareLocs, "|")
		for _, loc := range locList {
			if loc == this.node.Node_.Ip {
				fmt.Println("file previously shared")
				return false
			}
		}
	}
	//append local ip to dht(so other clients can download from local server)
	this.node.AppendTo(hash.Text(16), "|"+this.node.Node_.Ip)
	fmt.Println("append finish")
	this.server.FileNameMap[hash.Text(16)] = filePath
	fmt.Println("use the link below to download file")
	fmt.Println(hash.Text(16))
	return true
}
func (this *AppClient) StopShare(fileHash string) bool {
	//just remove the local ip from remote server
	success := this.node.RemoveFrom(fileHash, "|"+this.node.Node_.Ip)
	if !success {
		fmt.Println("stop sharing failed.Maybe you haven't shared it or there're some problems with other servers")
		fmt.Println("you can shut down the application to stop sharing")
		return false
	}
	delete(this.server.FileNameMap, fileHash)
	return true
}

//randomly choose a server from the available ones
func (this *AppClient) getFile(filehash string, recvpath string) bool {
	downloadAddrs, success := this.node.Get(filehash)
	if !success {
		fmt.Println("file not shared")
	}
	downloadAddrlist := strings.Split(downloadAddrs, "|")

	var received []byte
	for _, addr := range downloadAddrlist {
		client, e := rpc.Dial("tcp", addr)
		if e == nil {
			err := client.Call("AppServer.GetFile", filehash, &received)
			if err == nil {
				file, _ := os.Create(recvpath)
				file.Write(received)
				file.Close()
				client.Close()
				return true
			}
			client.Close()
		}
	}
	return false
}
