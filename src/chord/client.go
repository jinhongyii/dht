package chord

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"strings"
	"sync"
	"time"
)

type Client struct {
	Node_    Node
	server   *rpc.Server
	wg       *sync.WaitGroup
	listener net.Listener
}

func (this *Client) Create() {
	this.Node_.KvStorage.V = make(map[string]string)
	this.Node_.Successors[1].Ip = this.Node_.Ip
	this.Node_.Successors[1].Id = this.Node_.Id
	this.Node_.Predecessor = new(FingerType)
	this.Node_.Predecessor.Ip = this.Node_.Ip
	this.Node_.Predecessor.Id = this.Node_.Id
	path := strings.ReplaceAll(this.Node_.Ip, ":", "_") + ".backup"
	this.Node_.File, _ = os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	this.Node_.bufferWriter = bufio.NewWriter(this.Node_.File)
	//this.Node_.recover()
	go this.Stabilize()
	go this.Fix_fingers()
	go this.CheckPredecessor()
}

func (this *Client) Stabilize() {
	for this.Node_.Listening {
		this.Node_.stabilize()
		time.Sleep(333 * time.Millisecond)
	}
}

func (this *Client) CheckPredecessor() {
	for this.Node_.Listening {
		this.Node_.checkPredecessor()
		time.Sleep(333 * time.Millisecond)
	}
}

func (this *Client) Fix_fingers() {
	var fingerEntry = 1
	for this.Node_.Listening {
		this.Node_.fix_fingers(&fingerEntry)
		time.Sleep(333 * time.Millisecond)
	}
}
func Exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
func (this *Client) Join(otherNode string) bool {
	this.Node_.KvStorage.V = make(map[string]string)
	this.Node_.Predecessor = nil
	path := strings.ReplaceAll(this.Node_.Ip, ":", "_") + ".backup"
	this.Node_.File, _ = os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	this.Node_.bufferWriter = bufio.NewWriter(this.Node_.File)
	//this.Node_.recover()
	client, e := rpc.Dial("tcp", otherNode)
	if e != nil {
		return false
	}
	err := client.Call("Node.FindSuccessor", &FindRequest{*this.Node_.Id, 0}, &this.Node_.Successors[1])
	if err != nil {
		fmt.Println(err)
	}
	client.Close()
	client, e = rpc.Dial("tcp", this.Node_.getWorkingSuccessor().Ip)
	if e != nil {
		log.Fatal("dialing:", e)
	}
	var receivedMap map[string]string
	var p FingerType
	err = client.Call("Node.GetKeyValMap", 0, &receivedMap)
	if err != nil {
		fmt.Println(err)
	}
	err = client.Call("Node.GetPredecessor", 0, &p)
	if err != nil {
		fmt.Println(err)
	}
	this.Node_.KvStorage.mux.Lock()
	for k, v := range receivedMap {
		var k_hash = hashString(k)
		if between(p.Id, k_hash, this.Node_.Id, true) {
			this.Node_.KvStorage.V[k] = v
			length, err := this.Node_.bufferWriter.WriteString("put " + k + " " + v + "\n")
			this.Node_.bufferWriter.Flush()
			if err != nil {
				fmt.Println("actually write:", length, " ", err)
			}
		}
	}
	this.Node_.KvStorage.mux.Unlock()

	err = client.Call("Node.CompleteMigrate", &FingerType{this.Node_.Ip, this.Node_.Id}, nil)
	err = client.Call("Node.Notify", &FingerType{this.Node_.Ip, this.Node_.Id}, nil)
	if err != nil {
		fmt.Println(err)
	}
	client.Close()
	go this.Stabilize()
	go this.Fix_fingers()
	go this.CheckPredecessor()
	return err == nil
}
func (this *Client) Put(key string, val string) bool {
	k_hash := hashString(key)
	var successor FingerType
	_ = this.Node_.FindSuccessor(&FindRequest{*k_hash, 0}, &successor)
	client, err := rpc.Dial("tcp", successor.Ip)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	err = client.Call("Node.Put_", &ChordKV{key, val}, nil)
	client.Close()
	return err == nil
}
func (this *Client) Get(key string) (string, bool) {
	var val string
	var maxrequest = 3
	var success = false
	k_hash := hashString(key)
	var successor FingerType
	for i := 0; i < maxrequest && !success; i++ {
		_ = this.Node_.FindSuccessor(&FindRequest{*k_hash, 0}, &successor)
		client, err := rpc.Dial("tcp", successor.Ip)
		if err != nil {
			log.Fatal("dialing:", err)
		}
		err = client.Call("Node.Get_", &key, &val)
		client.Close()
		success = (err == nil)
		if !success {
			time.Sleep(2 * time.Second)
		}
	}

	return val, success
}
func (this *Client) Del(key string) bool {
	k_hash := hashString(key)
	var successor FingerType
	_ = this.Node_.FindSuccessor(&FindRequest{*k_hash, 0}, &successor)
	client, err := rpc.Dial("tcp", successor.Ip)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	var success bool
	err = client.Call("Node.Delete_", &key, &success)
	client.Close()
	return success
}
func (this *Client) Dump() {
	fmt.Println("ip:", this.Node_.Ip)
	fmt.Println(this.Node_.KvStorage.V)
	fmt.Print("Finger: ")
	for i := 1; i <= m; i++ {
		fmt.Print(this.Node_.Finger[i].Ip, " ")
	}
	fmt.Println()
	fmt.Println("successor: ", this.Node_.Successors)
	if this.Node_.Predecessor != nil {
		fmt.Println("Predecessor: " + this.Node_.Predecessor.Ip)
	} else {
		fmt.Println("Predecessor : <nil>")
	}

}
func (this *Client) Quit() {
	client, err := rpc.Dial("tcp", this.Node_.getWorkingSuccessor().Ip)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	err = client.Call("Node.Merge", &this.Node_.KvStorage.V, nil)
	client.Close()
	this.wg.Done()
	this.Node_.Listening = false
	this.Node_.clearbackup()
	_ = this.Node_.File.Close()
	//time.Sleep(2 * time.Second)
	err = this.listener.Close()
	if err != nil {
		println(err)
	}
}
func (this *Client) Ping(addr string) bool {
	return this.Node_.ping(addr)
}
func (this *Client) Run(wg *sync.WaitGroup) {
	this.wg = wg
	wg.Add(1)
	var e error
	this.listener, e = net.Listen("tcp", this.Node_.Ip)
	if e != nil {
		fmt.Println(e)
	}
	go this.server.Accept(this.listener)
	this.Node_.Listening = true
	this.Node_.Ip = GetLocalAddress() + this.Node_.Ip
	this.Node_.Id = hashString(this.Node_.Ip)
}
