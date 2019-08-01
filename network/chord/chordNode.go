package chord

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"
)

type ChordKV struct {
	Key string
	Val string
}
type Counter struct {
	V   map[string]string
	Mux sync.Mutex
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
	Id                *big.Int
	Ip                string
	KvStorage         Counter
	additionalStorage Counter
	sucMux            sync.RWMutex
	Successors        [m + 1]FingerType
	Finger            [m + 1]FingerType
	Predecessor       *FingerType
	Listening         bool
	File              *os.File
	stabilizeMux      sync.Mutex
	fixMux            sync.Mutex
}

func (this *Node) Merge(kvpairs *map[string]string, success *bool) error {
	this.KvStorage.Mux.Lock()
	for k, v := range *kvpairs {
		this.KvStorage.V[k] = v
		//length, err := this.File.WriteString("put " + k + " " + v + "\n")
		//if err != nil {
		//	fmt.Println("actually write:", length, " ", err)
		//}
	}
	this.KvStorage.Mux.Unlock()
	return nil
}

func (this *Node) GetKeyValMap(a *int, b *map[string]string) error {
	this.KvStorage.Mux.Lock()
	*b = this.KvStorage.V
	this.KvStorage.Mux.Unlock()
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
	//this.sucMux.RLock()
	*successors = this.Successors
	//this.sucMux.RUnlock()
	return nil
}
func (this *Node) getWorkingSuccessor() FingerType {
	var i int
	this.sucMux.Lock()
	for i = 1; i <= m; i++ {
		if this.ping(this.Successors[i].Ip) {
			break
		}
	}
	if i != 1 {
		if i == m+1 {
			this.sucMux.Unlock()
			return FingerType{}
		}
		//fmt.Println(this.Ip, " successor set to ", this.Successors[i].Ip, "(getworkingsuccessor)")
		client, err := rpc.Dial("tcp", this.Successors[i].Ip)
		if err == nil {
			defer client.Close()
		}
		if err != nil {
			this.sucMux.Unlock()
			return this.getWorkingSuccessor()
		}
		var suc_Successors [m + 1]FingerType
		_ = client.Call("Node.GetSuccessors", 0, &suc_Successors)
		this.Successors[1] = this.Successors[i]
		for j := 2; j <= m; j++ {
			this.Successors[j] = suc_Successors[j-1]
		}
		this.sucMux.Unlock()
	} else {
		this.sucMux.Unlock()
	}
	return this.Successors[1]

}

func (this *Node) stabilize() {
	suc := this.getWorkingSuccessor()
	if suc.Id == nil {
		return
	}
	client, e := rpc.Dial("tcp", suc.Ip)
	if e == nil {
		defer client.Close()
	}
	if e != nil {
		//log.Fatal("dialing:", e)
		return
	}
	var p FingerType
	err := client.Call("Node.GetPredecessor", 0, &p)

	if err == nil && this.ping(p.Ip) {

		this.sucMux.Lock()
		if (p.Id != nil) && between(this.Id, p.Id, this.Successors[1].Id, false) {
			this.Successors[1] = p
			//fmt.Println(this.Ip, " successor set to ", p.Ip, "(stabilize)")
		}
		client, e = rpc.Dial("tcp", this.Successors[1].Ip)
		if e == nil {
			defer client.Close()
		}
		this.sucMux.Unlock()
		if e != nil {
			//log.Fatal("dialing:", e)
			return
		}
	}

	err = client.Call("Node.Notify", &FingerType{this.Ip, this.Id}, nil)
	if err != nil {
		//fmt.Println(err, "(stabilize)")
	}
	var suc_Successors [m + 1]FingerType
	err = client.Call("Node.GetSuccessors", 0, &suc_Successors)
	if err != nil {

		return
	}
	this.sucMux.Lock()
	for i := 2; i <= m; i++ {
		this.Successors[i] = suc_Successors[i-1]
	}
	this.sucMux.Unlock()

}

func (this *Node) ping(ip string) bool {
	var success bool
	for times := 0; times < 3; times++ {
		ch := make(chan bool)
		go func() {
			client, err := rpc.Dial("tcp", ip)
			if err == nil {
				defer client.Close()
			}
			if err != nil {
				ch <- false
				return
			} else {
				var listening bool
				_ = client.Call("Node.GetListeningStatus", 0, &listening)
				if listening {
					ch <- true
				} else {
					ch <- false
				}
				return
			}
		}()
		select {
		case success = <-ch:
			if success {
				return true
			} else {
				continue
			}
		case <-time.After(666 * time.Millisecond):
			//fmt.Println("ping ", ip, " time out")
			continue
		}
	}
	return false
}
func (this *Node) checkPredecessor() {
	if this.Predecessor != nil {
		if !this.ping(this.Predecessor.Ip) {
			//var tmp = this.Predecessor.Ip
			this.Predecessor = nil
			this.additionalStorage.Mux.Lock()
			this.KvStorage.Mux.Lock()
			for k, v := range this.additionalStorage.V {
				this.KvStorage.V[k] = v
			}
			client, err := rpc.Dial("tcp", this.getWorkingSuccessor().Ip)
			if err == nil {
				defer client.Close()
			}
			if err != nil {
				fmt.Println(err)
				return
			}
			var success bool
			this.KvStorage.Mux.Unlock()

			client.Call("Node.AdditionalPutMap", this.additionalStorage.V, &success)
			this.additionalStorage.V = make(map[string]string)
			this.additionalStorage.Mux.Unlock()

			//fmt.Println(this.Ip, " predecessor set to nil  prev_predecessor:", tmp)
		}
	}
}

func (this *Node) fix_fingers(fingerEntry *int) {
	var tmp FingerType = this.Finger[*fingerEntry]
	//if this.Ip == "10.166.172.2:1010" {
	//	fmt.Println("1010 fix finger")
	//}
	ch := make(chan error)
	go func() {
		err := this.FindSuccessor(&FindRequest{*jump(this.Id, *fingerEntry), 0}, &this.Finger[*fingerEntry])
		ch <- err
	}()
	select {
	case err := <-ch:
		if err != nil {
			return
		}
	case <-time.After(2 * time.Second):
		*fingerEntry = 1
		return
	}
	if *fingerEntry == 1 && tmp != this.Finger[*fingerEntry] {
		//fmt.Println(this.Ip, " finger 1 set to ", this.Finger[*fingerEntry].Ip)
	}
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

//func (this *Node) clearbackup() {
//	this.File.Close()
//	this.File, _ = os.Create(strings.ReplaceAll(this.Ip, ":", "_") + ".backup")
//}
//func (this *Node) recover() {
//	bufferReader := bufio.NewReader(this.File)
//	this.File.Seek(0, 0)
//	this.KvStorage.Mux.Lock()
//	for {
//		line, e := bufferReader.ReadString('\n')
//		if e != nil {
//			this.KvStorage.Mux.Unlock()
//			return
//		}
//		words := strings.Split(line, " ")
//		if words[0] == "put" {
//			this.KvStorage.V[words[1]] = words[2][:len(words[2])-1]
//		} else if words[0] == "delete" {
//			delete(this.KvStorage.V, words[1][:len(words[1])-1])
//		} else if words[0] == "append" {
//			this.KvStorage.V[words[1]] += words[2][:len(words[2])-1]
//		} else {
//			tmp := this.KvStorage.V[words[1]]
//			tmp = strings.ReplaceAll(tmp, words[2][:len(words[2])-1], "")
//			this.KvStorage.V[words[1]] = tmp
//		}
//	}
//
//}
func (this *Node) CompleteMigrate(otherNode FingerType, lala *int) error {
	var deletion []string
	this.KvStorage.Mux.Lock()
	for k, _ := range this.KvStorage.V {
		if between(this.Predecessor.Id, HashString(k), otherNode.Id, true) {
			deletion = append(deletion, k)
		}
	}
	for _, v := range deletion {
		delete(this.KvStorage.V, v)
		//length, err := this.File.WriteString("delete " + v + "\n")
		//if err != nil {
		//	fmt.Println("actually write:", length, " ", err)
		//}
	}
	this.KvStorage.Mux.Unlock()
	return nil
}

func (this *Node) Notify(otherNode FingerType, lalala *int) error {

	if this.Predecessor == nil || between(this.Predecessor.Id, otherNode.Id, this.Id, false) {
		this.Predecessor = new(FingerType)
		*this.Predecessor = otherNode
		client, err := rpc.Dial("tcp", otherNode.Ip)
		if err == nil {
			defer client.Close()
		}
		if err != nil {
			fmt.Println(err)
		} else {
			useless := 0
			otherMap := make(map[string]string)
			client.Call("Node.GetKeyValMap", &useless, &otherMap)
			this.additionalStorage.V = otherMap
		}

		//fmt.Println(this.Ip, " predecessor set to ", otherNode.Ip)
		return nil
	} else if this.Predecessor.Ip == otherNode.Ip {
		return nil
	}
	return errors.New(this.Ip + " notify failed ,'father':" + otherNode.Ip + " real pre:" + this.Predecessor.Ip)

}

type FindRequest struct {
	Id    big.Int
	Times int
}

func (this *Node) FindSuccessor(request *FindRequest, successor *FingerType) error {
	if request.Times > maxfindTimes {
		return errors.New("can't find ")
	}
	var suc = new(FingerType)
	*suc = this.getWorkingSuccessor()
	if suc.Id == nil {
		*successor = *suc
		return errors.New("network fail")
	}
	if suc.Id.Cmp(this.Id) == 0 || request.Id.Cmp(this.Id) == 0 {
		*successor = *suc
	} else if between(this.Id, &request.Id, suc.Id, true) {
		successor.Ip = suc.Ip
		successor.Id = suc.Id
	} else {
		next_step := this.closest_preceding_node(&request.Id)
		//next_step:=this.getWorkingSuccessor()
		client, e := rpc.Dial("tcp", next_step.Ip)
		if e == nil {
			defer client.Close()
		}
		if e != nil {
			fmt.Println("findSuccessor wait")
			time.Sleep(1 * time.Second)
			return this.FindSuccessor(request, successor)
		}
		var result FingerType
		request.Times++
		err := client.Call("Node.FindSuccessor", &request, &result)
		if err != nil {
			//time.Sleep(1 * time.Second)
			//defer client.Close()
			//fmt.Println(this.Ip, " findSuccessor wait ,request:", request.Id)
			//return this.FindSuccessor(request, successor)
			return err
		} else {
			*successor = result
		}
	}
	return nil
}
func (this *Node) closest_preceding_node(id *big.Int) FingerType {
	for i := m; i > 0; i-- {
		if this.Finger[i].Id != nil && between(this.Id, this.Finger[i].Id, id, true) && this.ping(this.Finger[i].Ip) {
			return this.Finger[i]
		}
	}
	return this.Successors[1]
}
func (this *Node) Put_(args *ChordKV, success *bool) error {
	//time.Sleep(40*time.Millisecond)
	client, err := rpc.Dial("tcp", this.getWorkingSuccessor().Ip)
	if err == nil {
		defer client.Close()
	}
	if err != nil {
		return err
	} else {
		client.Call("Node.AdditionalPut", args, success)
	}
	this.KvStorage.Mux.Lock()
	this.KvStorage.V[args.Key] = args.Val
	this.KvStorage.Mux.Unlock()
	//length, err := this.File.WriteString("put " + args.Key + " " + args.Val + "\n")
	//
	//if err != nil {
	//	fmt.Println("actually write:", length, " ", err)
	//}
	//fmt.Println(this.Ip + " put " + args.Key + " => " + args.Val)
	return nil
}
func (this *Node) Get_(key *string, val *string) error {
	//time.Sleep(30*time.Millisecond)
	this.KvStorage.Mux.Lock()
	*val = this.KvStorage.V[*key]
	if *val == "" {
		this.KvStorage.Mux.Unlock()

		client, e := rpc.Dial("tcp", this.getWorkingSuccessor().Ip)
		if e == nil {
			defer client.Close()
		}
		if e != nil {
			return e
		}
		e = client.Call("Node.RPCFindAdditional", key, val)

		if e != nil {
			return e
		}
		return nil
	}
	this.KvStorage.Mux.Unlock()
	//fmt.Println(this.Ip + " get " + *key + " => " + *val)
	return nil
}
func (this *Node) Delete_(key *string, success *bool) error {
	client, err := rpc.Dial("tcp", this.getWorkingSuccessor().Ip)
	if err == nil {
		defer client.Close()
	}
	if err != nil {
		return err
	} else {
		client.Go("Node.AdditionalDel", key, success, nil)
	}
	this.KvStorage.Mux.Lock()
	_, ok := this.KvStorage.V[*key]
	delete(this.KvStorage.V, *key)
	this.KvStorage.Mux.Unlock()
	//length, err := this.File.WriteString("delete " + *key + "\n")
	//if err != nil {
	//	fmt.Println("actually write:", length, " ", err)
	//}
	//fmt.Println(this.Ip + " delete " + *key)
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

func HashString(elt string) *big.Int {
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
func (this *Node) Append(kv ChordKV, success *bool) error {
	this.KvStorage.Mux.Lock()
	tmp := this.KvStorage.V[kv.Key]
	this.KvStorage.V[kv.Key] = tmp + kv.Val
	//length, err := this.File.WriteString("append " + kv.Key + " " + kv.Val + "\n")
	//if err != nil {
	//	fmt.Println("actually write:", length, " ", err)
	//}
	this.KvStorage.Mux.Unlock()
	*success = true
	return nil
}

//func (this *Node) Remove(kv ChordKV, success *bool) error {
//	this.KvStorage.Mux.Lock()
//	tmp, ok := this.KvStorage.V[kv.Key]
//	if !ok {
//		this.KvStorage.Mux.Unlock()
//		*success = false
//		return nil
//	} else {
//		removed := strings.ReplaceAll(tmp, kv.Val, "")
//		//length, err := this.File.WriteString("remove " + kv.Key + " " + kv.Val + "\n")
//		//if err != nil {
//		//	fmt.Println("actually write:", length, " ", err)
//		//}
//		this.KvStorage.V[kv.Key] = removed
//		this.KvStorage.Mux.Unlock()
//		*success = true
//		return nil
//	}
//
//}

func (this *Node) AdditionalPut(key *ChordKV, success *bool) error {
	this.additionalStorage.Mux.Lock()
	this.additionalStorage.V[key.Key] = key.Val
	this.additionalStorage.Mux.Unlock()
	return nil
}
func (this *Node) AdditionalDel(key string, success *bool) error {
	this.additionalStorage.Mux.Lock()
	delete(this.additionalStorage.V, key)
	this.additionalStorage.Mux.Unlock()
	return nil
}
func (this *Node) AdditionalPutMap(kvMap map[string]string, success *bool) error {
	this.additionalStorage.Mux.Lock()
	for k, v := range kvMap {
		this.additionalStorage.V[k] = v
	}
	this.additionalStorage.Mux.Unlock()
	return nil
}
func (this *Node) QuickStabilize(a int, b *int) error {
	this.stabilizeMux.Lock()
	this.stabilize()
	this.stabilizeMux.Unlock()
	this.fixMux.Lock()
	tmp := 1
	this.fix_fingers(&tmp)
	this.fixMux.Unlock()
	return nil
}
func (this *Node) RPCFindAdditional(key string, val *string) error {
	this.additionalStorage.Mux.Lock()
	var ok bool
	*val, ok = this.additionalStorage.V[key]
	this.additionalStorage.Mux.Unlock()
	if !ok {
		return errors.New("not found key")
	}
	return nil
}
