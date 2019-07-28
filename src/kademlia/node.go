package kademlia

import (
	"chord"
	"fmt"
	"math/big"
	"net/rpc"
	"sort"
	"time"
)

const (
	maxbucket  = 160
	K          = 15 //debug mode
	alpha      = 3
	tExpire    = 874 * time.Second
	tRefresh   = 36 * time.Second
	tReplicate = 36 * time.Second
	tRepublish = 864 * time.Second
)

type Counter chord.Counter

type Node struct {
	//1-base
	//todo:没加锁不知道对不对
	RoutingTable RoutingTable //lru里放addrtype
	Listening    bool
	KvStorage    MemStore
}

func distance(addr1 *Contact, addr2 *Contact) *big.Int {
	return distance2(addr1.Id, addr2.Id)
}
func distance2(id1 *big.Int, id2 *big.Int) *big.Int {
	dis := big.NewInt(0)
	dis.Xor(id1, id2)
	return dis
}

func (this *Node) IterativeFindNode_(k_hash *big.Int) []Contact {
	tmp := this.RoutingTable.GetClosest(k_hash, K)
	bucketid := this.RoutingTable.getbucketid(k_hash)
	this.RoutingTable.RefreshMap[bucketid] = time.Now().Add(tRefresh)
	shortList := make(Contacts, 0)
	seen := make(map[string]struct{})
	for i := 0; i < alpha && i < len(tmp); i++ {
		seen[tmp[i].Ip] = struct{}{}
	}
	seen[this.RoutingTable.Ip] = struct{}{}
	probenum := 0
	ch := make(chan FindNodeReturn, 20)
	for i := 0; probenum < alpha && probenum < len(tmp) && i < len(tmp); i++ {
		contact := tmp[i]
		if this.ping(contact.Ip) {
			go this.findNode(contact.Ip, ch, k_hash)
			probenum++
		}
	}
	for probenum > 0 {
		recvClosest := <-ch
		probenum--
		if recvClosest.Header.Id == nil {
			continue
		}
		shortList = append(shortList, recvClosest.Header)
		this.RoutingTable.update(&recvClosest.Header)
		for _, contact := range recvClosest.Closest { //todo:loose parallelism should limit connects?
			if _, ok := seen[contact.Ip]; !ok {
				seen[contact.Ip] = struct{}{}
				go this.findNode(contact.Ip, ch, k_hash)
				probenum++
			}
		}
	}
	sort.Slice(shortList, func(i, j int) bool {
		disi := distance2(shortList[i].Id, k_hash)
		disj := distance2(shortList[j].Id, k_hash)
		return disi.Cmp(disj) < 0
	})
	if len(shortList) > K {
		shortList = shortList[:K]
	}
	return shortList
}
func (this *Node) IterativeFindNode(key string) []Contact {
	k_hash := chord.HashString(key)
	return this.IterativeFindNode_(k_hash)
}
func (this *Node) findNode(ip string, recv chan FindNodeReturn, target *big.Int) {
	//fmt.Println("send findNode to ",ip)
	if !this.ping(ip) {
		this.RoutingTable.failNode(Contact{chord.HashString(ip), ip})
		recv <- FindNodeReturn{Contact{}, nil}
		return
	}
	client, e := rpc.Dial("tcp", ip)
	if e != nil {
		fmt.Println(e)

		recv <- FindNodeReturn{Contact{}, nil}
		return
	}
	defer client.Close()
	var ret FindNodeReturn
	err := client.Call("Node.RPCFindNode",
		FindNodeRequest{Contact{this.RoutingTable.Id, this.RoutingTable.Ip}, target},
		&ret)

	if err != nil {
		fmt.Println(err)
		recv <- FindNodeReturn{Contact{}, nil}
		return
	}
	recv <- ret
}
func (this *Node) IterativeFindValue(key string) (Set, bool) {
	val, success := this.KvStorage.get(key)
	if success {
		return val, true
	}
	k_hash := chord.HashString(key)
	tmp := this.RoutingTable.GetClosest(k_hash, K)
	bucketid := this.RoutingTable.getbucketid(k_hash)
	this.RoutingTable.RefreshMap[bucketid] = time.Now().Add(tRefresh)
	shortList := make(Contacts, 0)
	seen := make(map[string]struct{})
	for i := 0; i < alpha && i < len(tmp); i++ {
		seen[tmp[i].Ip] = struct{}{}
	}
	seen[this.RoutingTable.Ip] = struct{}{}
	probenum := 0
	ch := make(chan FindValueReturn, 20)

	for i := 0; probenum < alpha && probenum < len(tmp) && i < len(tmp); i++ {
		contact := tmp[i]
		if this.ping(contact.Ip) {
			go this.findVal(contact.Ip, ch, key, k_hash)
			probenum++
		}

	}
	for probenum > 0 {
		recvValue := <-ch
		probenum--
		if recvValue.Header.Id == nil {
			continue
		}
		this.RoutingTable.update(&recvValue.Header)
		if recvValue.Closest == nil {
			sort.Slice(shortList, func(i, j int) bool {
				disi := distance2(shortList[i].Id, k_hash)
				disj := distance2(shortList[j].Id, k_hash)
				return disi.Cmp(disj) < 0
			})
			if shortList.Len() != 0 {
				for val, _ := range recvValue.Val {
					this.store(shortList[0].Ip, key, val, time.Now().Add(this.RoutingTable.calculateExpire(shortList[0].Id)), false)
				}

			}
			//fmt.Println("get ", recvValue.Val, " at ", recvValue.Header.Ip, " from ", this.RoutingTable.Ip)
			return recvValue.Val, true
		}
		shortList = append(shortList, recvValue.Header)
		//fmt.Println("found ",recvValue.Closest," from ",recvValue.Header.Ip)
		for _, contact := range recvValue.Closest { //todo:loose parallelism should limit connects?
			if _, ok := seen[contact.Ip]; !ok {
				seen[contact.Ip] = struct{}{}
				go this.findVal(contact.Ip, ch, key, k_hash)
				probenum++
			}
		}
	}
	return nil, false
}
func (this *Node) findVal(ip string, recv chan FindValueReturn, key string, key_hash *big.Int) {
	//fmt.Println("send findval to ",ip)
	if !this.ping(ip) {
		this.RoutingTable.failNode(Contact{chord.HashString(ip), ip})
		recv <- FindValueReturn{Contact{}, nil, nil}
		return
	}
	client, e := rpc.Dial("tcp", ip)
	if e != nil {
		recv <- FindValueReturn{Contact{}, nil, nil}
		return
	}
	defer client.Close()
	var ret FindValueReturn
	err := client.Call("Node.RPCFindValue",
		FindValueRequest{Contact{this.RoutingTable.Id, this.RoutingTable.Ip},
			key_hash, key}, &ret)
	if err != nil {
		recv <- FindValueReturn{Contact{}, nil, nil}
		return
	}
	recv <- ret
}

func ping(header Contact, ip string) (bool, Contact) {
	var success bool
	for times := 0; times < 1; times++ { //todo:maybe we need to reduce retry times
		ch := make(chan bool, 1)
		ch2 := make(chan Contact, 1)
		go func() {
			client, err := rpc.Dial("tcp", ip)
			if err != nil {
				ch <- false
				return
			} else {
				defer client.Close()
				var ret PingReturn
				_ = client.Call("Node.RPCPing", header, &ret)
				if ret.Success {
					ch <- true
					ch2 <- ret.Header
				} else {
					ch <- false
				}
				return
			}
		}()
		select {
		case success = <-ch:
			if success {
				return true, <-ch2
			} else {
				fmt.Println("ping ", ip, " failed")
				return false, Contact{}
			}
		case <-time.After(666 * time.Millisecond):
			fmt.Println("ping ", ip, " time out")
			continue
		}
	}
	return false, Contact{}
}
func (this *Node) store(ip string, key string, val string, expire time.Time, replicate bool) bool {
	client, e := rpc.Dial("tcp", ip)
	if e != nil {
		fmt.Println(e)
		return false
	}
	var ret StoreReturn
	defer client.Close()
	err := client.Call("Node.RPCStore", StoreRequest{
		Pair:      KVPair{key, val},
		Header:    Contact{Ip: this.RoutingTable.Ip, Id: this.RoutingTable.Id},
		Expire:    expire,
		Replicate: replicate,
	}, &ret)
	if err != nil {
		fmt.Println(err)
		return false
	}
	this.RoutingTable.update(&ret.Header)
	return ret.Success
}
func (this *Node) ping(ip string) bool {
	success, ret := ping(Contact{this.RoutingTable.Id, this.RoutingTable.Ip}, ip)
	if success {
		this.RoutingTable.update(&ret)
		return true
	} else {
		return false
	}
}

type PingReturn struct {
	Success bool
	Header  Contact
}

func (this *Node) RPCPing(header Contact, ret *PingReturn) error {
	go this.RoutingTable.update(&header)
	ret.Success = this.Listening
	ret.Header = Contact{this.RoutingTable.Id, this.RoutingTable.Ip}
	return nil
}

type KVPair struct {
	Key string
	Val string
}
type StoreRequest struct {
	Pair      KVPair
	Header    Contact
	Expire    time.Time
	Replicate bool
}
type StoreReturn struct {
	Success bool
	Header  Contact
}

//used for publish replicate and republish
func (this *Node) IterativeStore(key string, val string, origin bool, expire time.Time) {
	k_closest := this.IterativeFindNode(key)
	fmt.Println("store: get k closest list: ", k_closest)
	if origin {
		this.KvStorage.put(key, val, true, time.Now(), false)
		for _, contact := range k_closest {
			this.store(contact.Ip, key, val, time.Now().Add(tExpire), true)
		}
	} else {
		for _, contact := range k_closest {
			this.store(contact.Ip, key, val, expire, true)
		}
	}
}
func (this *Node) RPCStore(request StoreRequest, ret *StoreReturn) error {
	fmt.Println("put ", request.Pair, " at ", this.RoutingTable.Ip, " from ", request.Header.Ip)
	this.RoutingTable.update(&request.Header)
	this.KvStorage.put(request.Pair.Key, request.Pair.Val, false, request.Expire, request.Replicate)
	ret.Success = true
	ret.Header = Contact{this.RoutingTable.Id, this.RoutingTable.Ip}
	return nil
}

type FindNodeRequest struct {
	Header Contact
	Id     *big.Int
}
type FindNodeReturn struct {
	Header  Contact
	Closest Contacts
}

func (this *Node) RPCFindNode(request FindNodeRequest, ret *FindNodeReturn) error {
	this.RoutingTable.update(&request.Header)
	ret.Closest = this.RoutingTable.GetClosest(request.Id, K)
	ret.Header = Contact{this.RoutingTable.Id, this.RoutingTable.Ip}

	return nil
}

type FindValueRequest struct {
	Header Contact
	HashId *big.Int
	Key    string
}
type FindValueReturn struct {
	Header  Contact
	Closest []Contact
	Val     Set
}

func (this *Node) RPCFindValue(request FindValueRequest, ret *FindValueReturn) error {
	fmt.Println(this.RoutingTable.Ip, " receive find value")
	this.RoutingTable.update(&request.Header)
	val, ok := this.KvStorage.get(request.Key)
	if ok {
		fmt.Println("get ", request.Key, " => ", val, " at ", this.RoutingTable.Ip, " from ", request.Header.Ip)
		(*ret).Val = val
		ret.Closest = nil
	} else {
		(*ret).Closest = this.RoutingTable.GetClosest(request.HashId, K)
		ret.Val = nil
	}
	ret.Header = Contact{this.RoutingTable.Id, this.RoutingTable.Ip}
	return nil
}

//type DeleteRequest struct {
//	Header Contact
//	key    string
//}

//func (this *Node)RPCDelete(request DeleteRequest,success *bool)error{
//	this.RoutingTable.update(&request.Header)
//	this.KvStorage.StoreMux.Lock()
//	delete(this.KvStorage.V, request.key)
//	this.KvStorage.StoreMux.Unlock()
//	*success=true
//	return nil
//}
//func (this*Node)delete(Ip string,key string)bool{
//	client,e:=rpc.Dial("tcp",Ip)
//	if e!=nil{
//		fmt.Println(e)
//		return false
//	}
//	var success bool
//	err:=client.Call("Node.RPCDelete",DeleteRequest{
//		Header:Contact{this.RoutingTable.Ip,this.RoutingTable.Id},
//		key:key,
//	},&success)
//	client.Close()
//	if err!=nil{
//		fmt.Println(err)
//		return false
//	}
//	return true
//}
