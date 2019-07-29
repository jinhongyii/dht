package chord

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"strings"
	"sync"
	"time"
)

type Client struct {
	Node_    Node
	Server   *rpc.Server
	wg       *sync.WaitGroup
	Listener net.Listener
}

func (this *Client) Create() {
	this.Node_.KvStorage.V = make(map[string]string)
	this.Node_.additionalStorage.V = make(map[string]string)
	this.Node_.Successors[1].Ip = this.Node_.Ip
	this.Node_.Successors[1].Id = this.Node_.Id
	this.Node_.Predecessor = nil
	//path := strings.ReplaceAll(this.Node_.Ip, ":", "_") + ".backup"
	//this.Node_.File, _ = os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
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
	this.Node_.additionalStorage.V = make(map[string]string)
	this.Node_.Predecessor = nil
	//path := strings.ReplaceAll(this.Node_.Ip, ":", "_") + ".backup"
	//this.Node_.File, _ = os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	//this.Node_.recover()
	client, e := rpc.Dial("tcp", otherNode)
	if e != nil {
		return false
	}
	err := client.Call("Node.FindSuccessor", &FindRequest{*this.Node_.Id, 0}, &this.Node_.Successors[1])
	if err != nil {
		return this.Join(otherNode)
	}
	client.Close()
	client, e = rpc.Dial("tcp", this.Node_.getWorkingSuccessor().Ip)
	if e != nil {
		return this.Join(otherNode)
	}
	var receivedMap map[string]string
	var p FingerType
	err = client.Call("Node.GetKeyValMap", 0, &receivedMap)
	if err != nil {
		return this.Join(otherNode)
	}
	err = client.Call("Node.GetPredecessor", 0, &p)
	if err != nil {
		return this.Join(otherNode)
	}
	this.Node_.KvStorage.Mux.Lock()
	for k, v := range receivedMap {
		var k_hash = HashString(k)
		if between(p.Id, k_hash, this.Node_.Id, true) {
			this.Node_.KvStorage.V[k] = v
			//length, err := this.Node_.File.WriteString("put " + k + " " + v + "\n")
			//if err != nil {
			//	fmt.Println("actually write:", length, " ", err)
			//}
		}
	}
	this.Node_.KvStorage.Mux.Unlock()

	err = client.Call("Node.CompleteMigrate", &FingerType{this.Node_.Ip, this.Node_.Id}, nil)
	err = client.Call("Node.Notify", &FingerType{this.Node_.Ip, this.Node_.Id}, nil)

	if err != nil {
		fmt.Println(err, "(join)")
		return this.Join(otherNode)
	}
	client.Close()
	go this.Stabilize()
	go this.Fix_fingers()
	go this.CheckPredecessor()
	return true
}

//each string 3 copies
func (this *Client) SafePut(key string, val string) bool {
	ok1 := this.Put(key+"1", val)
	ok2 := this.Put(key+"2", val)
	ok3 := this.Put(key+"3", val)
	return ok1 && ok2 && ok3
}
func (this *Client) Put(key string, val string) bool {
	k_hash := HashString(key)
	var successor FingerType
	_ = this.Node_.FindSuccessor(&FindRequest{*k_hash, 0}, &successor)
	client, err := rpc.Dial("tcp", successor.Ip)
	if err != nil {
		return false
	}
	err = client.Call("Node.Put_", &ChordKV{key, val}, nil)
	client.Close()
	return err == nil
}
func (this *Client) SafeGet(key string) (string, bool) {
	success1, val1 := this.Get(key + "1")
	success2, val2 := this.Get(key + "2")
	success3, val3 := this.Get(key + "3")
	var val string
	if success1 {
		val = val1
	}
	if success2 {
		val = val2
	}
	if success3 {
		val = val3
	}
	if success1 || success2 || success3 {
		if !success1 {
			this.Put(key+"1", val)
		}
		if !success2 {
			this.Put(key+"2", val)
		}
		if !success3 {
			this.Put(key+"3", val)
		}
		return val, true
	}
	return "", false
}
func (this *Client) Get(key string) (bool, string) {
	var val string
	var maxrequest = 3
	var success = false
	k_hash := HashString(key)
	var successor FingerType
	for i := 0; i < maxrequest && !success; i++ {
		_ = this.Node_.FindSuccessor(&FindRequest{*k_hash, 0}, &successor)
		client, err := rpc.Dial("tcp", successor.Ip)
		if err == nil {
			err = client.Call("Node.Get_", &key, &val)
			client.Close()
		}
		success = err == nil
		if !success {
			time.Sleep(2 * time.Second)
		}
	}

	return success, val
}
func (this *Client) SafeDel(key string) bool {
	ok1 := this.Del(key + "1")
	ok2 := this.Del(key + "2")
	ok3 := this.Del(key + "3")
	return ok1 || ok2 || ok3
}
func (this *Client) Del(key string) bool {
	k_hash := HashString(key)
	var successor FingerType
	_ = this.Node_.FindSuccessor(&FindRequest{*k_hash, 0}, &successor)
	client, err := rpc.Dial("tcp", successor.Ip)
	if err != nil {
		return false
	}
	var success bool
	err = client.Call("Node.Delete_", &key, &success)
	if err != nil {
		return false
	}
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
	fmt.Print("Successors: ")
	for i := 1; i <= m; i++ {
		fmt.Print(this.Node_.Successors[i].Ip, " ")
	}
	fmt.Println()
	if this.Node_.Predecessor != nil {
		fmt.Println("Predecessor: " + this.Node_.Predecessor.Ip)
	} else {
		fmt.Println("Predecessor : <nil>")
	}

}
func (this *Client) Quit() {
	//client, err := rpc.Dial("tcp", this.Node_.getWorkingSuccessor().Ip)
	//if err != nil {
	//	log.Fatal("dialing:", err)
	//}
	//err = client.Call("Node.Merge", &this.Node_.KvStorage.V, nil) //todo:
	//_ = client.Close()
	////this.wg.Done()
	//this.Node_.Listening = false
	////this.Node_.clearbackup()
	////_ = this.Node_.File.Close()
	////time.Sleep(2 * time.Second)
	//err = this.Listener.Close()
	//if err != nil {
	//	println(err)
	//}
	this.ForceQuit()
}
func (this *Client) ForceQuit() {
	//this.wg.Done()
	this.Node_.Listening = false
	_ = this.Node_.File.Close()
	_ = this.Listener.Close()

}
func (this *Client) Rejoin(ip string) bool {
	this.wg.Add(1)
	var e error
	this.Listener.Accept()
	this.Listener, e = net.Listen("tcp", this.Node_.Ip)
	if e != nil {
		fmt.Println(e)
	}
	go this.Server.Accept(this.Listener)
	this.Node_.Listening = true
	this.Node_.Ip = GetLocalAddress() + this.Node_.Ip
	this.Node_.Id = HashString(this.Node_.Ip)
	this.Node_.KvStorage.V = make(map[string]string)
	this.Node_.Predecessor = nil
	path := strings.ReplaceAll(this.Node_.Ip, ":", "_") + ".backup"
	this.Node_.File, _ = os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	this.Node_.recover()
	client, e := rpc.Dial("tcp", ip)
	if e != nil {
		return false
	}
	err := client.Call("Node.FindSuccessor", &FindRequest{*this.Node_.Id, 0}, &this.Node_.Successors[1])
	if err != nil {
		return false
	}
	client.Close()
	client, e = rpc.Dial("tcp", this.Node_.getWorkingSuccessor().Ip)
	if e != nil {
		return false
	}
	var receivedMap map[string]string
	var p FingerType
	err = client.Call("Node.GetKeyValMap", 0, &receivedMap)
	if err != nil {
		return false
	}
	err = client.Call("Node.GetPredecessor", 0, &p)
	if err != nil {
		return false
	}
	this.Node_.KvStorage.Mux.Lock()
	for k, v := range receivedMap {
		var k_hash = HashString(k)
		if between(p.Id, k_hash, this.Node_.Id, true) {
			this.Node_.KvStorage.V[k] = v
			length, err := this.Node_.File.WriteString("put " + k + " " + v + "\n")
			if err != nil {
				fmt.Println("actually write:", length, " ", err)
			}
		}
	}
	this.Node_.KvStorage.Mux.Unlock()

	err = client.Call("Node.CompleteMigrate", &FingerType{this.Node_.Ip, this.Node_.Id}, nil)
	err = client.Call("Node.Notify", &FingerType{this.Node_.Ip, this.Node_.Id}, nil)
	if err != nil {
		return false
	}
	client.Close()
	go this.Stabilize()
	go this.Fix_fingers()
	go this.CheckPredecessor()
	return true
}
func (this *Client) Ping(addr string) bool {
	return this.Node_.ping(addr)
}
func (this *Client) Run() {
}
func (this *Client) SafeAppend(key string, appendPart string) bool {
	ok1 := this.AppendTo(key+"1", appendPart)
	ok2 := this.AppendTo(key+"2", appendPart)
	ok3 := this.AppendTo(key+"3", appendPart)
	return ok1 && ok2 && ok3
}
func (this *Client) AppendTo(key string, appendPart string) bool {
	k_hash := HashString(key)
	var successor FingerType
	_ = this.Node_.FindSuccessor(&FindRequest{*k_hash, 0}, &successor)
	client, err := rpc.Dial("tcp", successor.Ip)
	if err != nil {
		return false
	}
	err = client.Call("Node.Append", &ChordKV{key, appendPart}, nil)
	client.Close()
	return err == nil
}
func (this *Client) SafeRemove(key string, removepart string) bool {
	_, success := this.SafeGet(key)
	if !success {
		return false
	} else {
		ok1 := this.RemoveFrom(key+"1", removepart)
		ok2 := this.RemoveFrom(key+"2", removepart)
		ok3 := this.RemoveFrom(key+"3", removepart)
		return ok1 || ok2 || ok3
	}
}
func (this *Client) RemoveFrom(key string, removePart string) bool {
	k_hash := HashString(key)
	var successor FingerType
	_ = this.Node_.FindSuccessor(&FindRequest{*k_hash, 0}, &successor)
	client, err := rpc.Dial("tcp", successor.Ip)
	if err != nil {
		return false
	}
	var success = false
	err = client.Call("Node.Remove", &ChordKV{key, removePart}, &success)
	client.Close()
	return err == nil && success
}
