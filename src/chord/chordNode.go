package chord

import (
	"bufio"
	"crypto/sha1"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/rpc"
	"os"
	"strings"
	"sync"
)

//todo:implement r-successor
type ChordKV struct {
	Key string
	Val string
}
type Counter struct {
	V   map[string]string
	mux sync.Mutex
}

const (
	m            = 160
	maxfindTimes = 32
)

type FingerType struct {
	Ip string
	Id *big.Int
}
type Node struct {
	Id           *big.Int
	Ip           string
	KvStorage    Counter
	Successors   [m + 1]FingerType
	Finger       [m + 1]FingerType
	Predecessor  *FingerType
	Listening    bool
	File         *os.File
	bufferWriter *bufio.Writer
}

func (this *Node) Merge(kvpairs *map[string]string, success *bool) error {
	this.KvStorage.mux.Lock()
	for k, v := range *kvpairs {
		this.KvStorage.V[k] = v
		length, err := this.bufferWriter.WriteString("put " + k + " " + v + "\n")
		this.bufferWriter.Flush()
		if err != nil {
			fmt.Println("actually write:", length, " ", err)
		}
	}
	this.KvStorage.mux.Unlock()
	return nil
}

func (this *Node) GetKeyValMap(a *int, b *map[string]string) error {
	this.KvStorage.mux.Lock()
	*b = this.KvStorage.V
	this.KvStorage.mux.Unlock()
	return nil
}

func (this *Node) GetPredecessor(a *int, b *FingerType) error {
	if this.Predecessor != nil {
		*b = *this.Predecessor
		return nil
	} else {
		return errors.New("no predecessor")
	}

}
func (this *Node) GetListeningStatus(a int, listening *bool) error {
	*listening = this.Listening
	return nil
}
func (this *Node) GetSuccessors(a int, successors *[m + 1]FingerType) error {
	for i := 1; i <= m; i++ {
		(*successors)[i] = this.Successors[i]
	}
	return nil
}
func (this *Node) getWorkingSuccessor() *FingerType {
	var i int
	for i = 1; i <= m; i++ {
		if this.ping(this.Successors[i].Ip) {
			break
		}
	}
	if i != 1 {
		client, err := rpc.Dial("tcp", this.Successors[i].Ip)
		if err != nil {
			log.Fatal("dialing:", err)
		}
		var suc_Successors [m + 1]FingerType
		_ = client.Call("Node.GetSuccessors", 0, &suc_Successors)
		this.Successors[1] = this.Successors[i]
		for j := 2; j <= m; j++ {
			this.Successors[j] = suc_Successors[j-1]
		}
		client.Close()
	}
	return &this.Successors[1]

}

func (this *Node) stabilize() {
	client, e := rpc.Dial("tcp", this.getWorkingSuccessor().Ip)
	if e != nil {
		log.Fatal("dialing:", e)
	}
	var p FingerType
	err := client.Call("Node.GetPredecessor", 0, &p)
	client.Close()

	if (p.Id != nil) && between(this.Id, p.Id, this.getWorkingSuccessor().Id, false) {
		*this.getWorkingSuccessor() = p
	}
	client, e = rpc.Dial("tcp", this.getWorkingSuccessor().Ip)
	if e != nil {
		log.Fatal("dialing:", e)
	}
	_ = client.Call("Node.Notify", &FingerType{this.Ip, this.Id}, nil)
	var suc_Successors [m + 1]FingerType
	err = client.Call("Node.GetSuccessors", 0, &suc_Successors)
	if err != nil {
		fmt.Println(err)
	}
	for i := 2; i <= m; i++ {
		this.Successors[i] = suc_Successors[i-1]
	}
	client.Close()

}

func (this *Node) ping(ip string) bool {
	client, e := rpc.Dial("tcp", ip)
	if e != nil {
		//fmt.Println("ping ",ip," error")
		return false
	} else {
		var listening bool
		_ = client.Call("Node.GetListeningStatus", 0, &listening)
		client.Close()
		if listening {
			return true
		} else {
			return false
		}
	}
}
func (this *Node) checkPredecessor() {
	if this.Predecessor != nil {
		if !this.ping(this.Predecessor.Ip) {
			this.Predecessor = nil
		}
	}
}

func (this *Node) fix_fingers(fingerEntry *int) {
	_ = this.FindSuccessor(&FindRequest{*jump(this.Id, *fingerEntry), 0}, &this.Finger[*fingerEntry])
	fingerFound := this.Finger[*fingerEntry]
	*fingerEntry++
	if *fingerEntry > m {
		*fingerEntry = 1
		return
	}
	for {
		if between(this.Id, jump(this.Id, *fingerEntry), fingerFound.Id, true) {
			this.Finger[*fingerEntry] = fingerFound
			*fingerEntry++
			if *fingerEntry > m {
				*fingerEntry = 1
				break
			}
		} else {
			break
		}
	}
}
func (this *Node) clearbackup() {
	this.File.Close()
	this.File, _ = os.Create(strings.ReplaceAll(this.Ip, ":", "_") + ".backup")
}
func (this *Node) recover() {
	bufferReader := bufio.NewReader(this.File)
	this.File.Seek(0, 0)
	this.KvStorage.mux.Lock()
	for {
		line, e := bufferReader.ReadString('\n')
		if e != nil {
			this.KvStorage.mux.Unlock()
			return
		}
		words := strings.Split(line, " ")
		if words[0] == "put" {
			this.KvStorage.V[words[1]] = words[2][:len(words[2])-1]
		} else {
			delete(this.KvStorage.V, words[1][:len(words[1])-1])
		}
	}

}
func (this *Node) CompleteMigrate(otherNode FingerType, lala *int) error {
	var deletion []string
	this.KvStorage.mux.Lock()
	for k, _ := range this.KvStorage.V {
		if between(this.Predecessor.Id, hashString(k), otherNode.Id, true) {
			deletion = append(deletion, k)
		}
	}
	for _, v := range deletion {
		delete(this.KvStorage.V, v)
		length, err := this.bufferWriter.WriteString("delete " + v + "\n")
		this.bufferWriter.Flush()
		if err != nil {
			fmt.Println("actually write:", length, " ", err)
		}
	}
	this.KvStorage.mux.Unlock()
	return nil
}

func (this *Node) Notify(otherNode FingerType, lalala *int) error {
	if this.Predecessor == nil || between(this.Predecessor.Id, otherNode.Id, this.Id, false) {
		this.Predecessor = new(FingerType)
		*this.Predecessor = otherNode
		return nil
	}
	return errors.New("you're not my father")
}

type FindRequest struct {
	Id    big.Int
	Times int
}

func (this *Node) FindSuccessor(request *FindRequest, successor *FingerType) error {
	if request.Times > maxfindTimes {
		return errors.New("can't find ")
	}
	if this.getWorkingSuccessor().Id.Cmp(this.Id) == 0 || request.Id.Cmp(this.Id) == 0 {
		*successor = *this.getWorkingSuccessor()
	} else if between(this.Id, &request.Id, this.getWorkingSuccessor().Id, true) {
		successor.Ip = this.getWorkingSuccessor().Ip
		successor.Id = this.getWorkingSuccessor().Id
	} else {

		next_step := this.closest_preceding_node(&request.Id)
		//next_step:=this.getWorkingSuccessor()
		client, e := rpc.Dial("tcp", next_step.Ip)
		if e != nil {
			log.Fatal("dialing:", e)
		}
		var result FingerType
		request.Times++
		err := client.Call("Node.FindSuccessor", &request, &result)
		if err != nil {
			fmt.Println("finding:", e)
			this.stabilize()
		} else {
			*successor = result
		}
		client.Close()
	}
	return nil
}
func (this *Node) closest_preceding_node(id *big.Int) FingerType {
	for i := m; i > 0; i-- {
		if this.Finger[i].Id != nil && between(this.Id, this.Finger[i].Id, id, true) && this.ping(this.Finger[i].Ip) {
			return this.Finger[i]
		}
	}
	return *this.getWorkingSuccessor()
}
func (this *Node) Put_(args *ChordKV, success *bool) error {
	this.KvStorage.mux.Lock()
	this.KvStorage.V[args.Key] = args.Val
	this.KvStorage.mux.Unlock()
	length, err := this.bufferWriter.WriteString("put " + args.Key + " " + args.Val + "\n")
	this.bufferWriter.Flush()
	if err != nil {
		fmt.Println("actually write:", length, " ", err)
	}
	fmt.Println(this.Ip + " put " + args.Key + " => " + args.Val)
	return nil
}
func (this *Node) Get_(key *string, val *string) error {
	this.KvStorage.mux.Lock()
	*val = this.KvStorage.V[*key]
	if *val == "" {
		this.KvStorage.mux.Unlock()
		return errors.New("not found key")
	}
	this.KvStorage.mux.Unlock()
	fmt.Println(this.Ip + " get " + *key + " => " + *val)
	return nil
}
func (this *Node) Delete_(key *string, success *bool) error {
	this.KvStorage.mux.Lock()
	_, ok := this.KvStorage.V[*key]
	delete(this.KvStorage.V, *key)
	this.KvStorage.mux.Unlock()
	length, err := this.bufferWriter.WriteString("delete " + *key + "\n")
	this.bufferWriter.Flush()
	if err != nil {
		fmt.Println("actually write:", length, " ", err)
	}
	fmt.Println(this.Ip + " delete " + *key)
	*success = ok
	return nil
}

func between(start, elt, end *big.Int, inclusive bool) bool {
	if end.Cmp(start) > 0 {
		return (start.Cmp(elt) < 0 && elt.Cmp(end) < 0) || (inclusive && elt.Cmp(end) == 0)
	} else {
		return start.Cmp(elt) < 0 || elt.Cmp(end) < 0 || (inclusive && elt.Cmp(end) == 0)
	}
}
func GetLocalAddress() string {
	var localaddress string

	ifaces, err := net.Interfaces()
	if err != nil {
		panic("init: failed to find network interfaces")
	}

	// find the first non-loopback interface with an IP address
	for _, elt := range ifaces {
		if elt.Flags&net.FlagLoopback == 0 && elt.Flags&net.FlagUp != 0 {
			addrs, err := elt.Addrs()
			if err != nil {
				panic("init: failed to get addresses for network interface")
			}

			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok {
					if ip4 := ipnet.IP.To4(); len(ip4) == net.IPv4len {
						localaddress = ip4.String()
						break
					}
				}
			}
		}
	}
	if localaddress == "" {
		panic("init: failed to find non-loopback interface with valid address on this node")
	}

	return localaddress
}

const keySize = sha1.Size * 8

var two = big.NewInt(2)
var hashMod = new(big.Int).Exp(big.NewInt(2), big.NewInt(keySize), nil)

func hashString(elt string) *big.Int {
	hasher := sha1.New()
	hasher.Write([]byte(elt))
	return new(big.Int).SetBytes(hasher.Sum(nil))
}
func jump(id *big.Int, fingerentry int) *big.Int {

	fingerentryminus1 := big.NewInt(int64(fingerentry) - 1)
	jump := new(big.Int).Exp(two, fingerentryminus1, nil)
	sum := new(big.Int).Add(id, jump)

	return new(big.Int).Mod(sum, hashMod)
}
