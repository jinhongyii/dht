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
func getBytesFromFilePart(filepath string, index int, totpartnum int) ([]byte, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, errors.New("no such file")
	}
	fileSize, _ := f.Seek(0, 2)
	var partSize int64
	if index == totpartnum-1 {
		partSize = fileSize - (int64(totpartnum)-1)*(fileSize/int64(totpartnum))
	} else {
		partSize = fileSize / int64(totpartnum)
	}
	total := make([]byte, partSize)
	f.Seek(int64(index)*(fileSize/int64(totpartnum)), 0)
	f.Read(total)
	f.Close()
	return total, nil

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
func (this *AppClient) GetFile(filehash string, recvpath string) bool {
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
				makeFile(recvpath, received)
				client.Close()
				return true
			}
			client.Close()
		}
	}
	return false
}

func makeFile(recvpath string, received []byte) {
	file, _ := os.Create(recvpath)
	file.Write(received)
	file.Close()
}

func (this *AppClient) GetFileFromMultipleServer(filehash string, recvpath string, maxThread int) bool {
	downloadAddrs, success := this.node.Get(filehash)
	if !success {
		fmt.Println("file not shared")
		return false
	}
	downloadAddrlist := strings.Split(downloadAddrs, "|")
	cnt := 0
	var availableList []string
	for _, addr := range downloadAddrlist {
		if cnt == maxThread {
			break
		}
		if this.node.Ping(addr) {
			availableList = append(availableList, addr)
			cnt++
		}
	}
	fmt.Println("available list get ")
	filebyte := make([]byte, 0)
	var tmp = make([][]byte, len(availableList))
	var wg sync.WaitGroup
	for i, addr := range availableList {
		wg.Add(1)
		go downloadPart(addr, filehash, i, len(availableList), &wg, &tmp[i])
	}
	wg.Wait()
	for _, recvbytes := range tmp {
		filebyte = append(filebyte, recvbytes...)
	}
	makeFile(recvpath, filebyte)
	return true

}
func downloadPart(serverIp string, fileHash string, index int, totpartnum int, wg *sync.WaitGroup, ret *[]byte) error {
	client, e := rpc.Dial("tcp", serverIp)
	if e != nil {
		fmt.Println(e)
		wg.Done()
		return errors.New("can't dial server,need to filter addr list again")
	}
	var received []byte
	err := client.Call("AppServer.GetFilePart", Getfilerequest{fileHash, index, totpartnum}, &received)
	if err != nil {
		fmt.Println(err)
		wg.Done()
		client.Close()
		return errors.New("getFile failed")
	}
	client.Close()
	*ret = received
	fmt.Println("get part ", index, " successful:", string(received))
	wg.Done()
	return nil
}
