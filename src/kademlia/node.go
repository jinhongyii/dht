package kademlia

import (
	"chord"
	"fmt"
	"math/big"
	"net/rpc"
	"time"
)

const (
	maxbucket = 160
	k         = 20
	alpha     = 3
)

type Counter chord.Counter
type AddrType chord.FingerType
type RPCHeader AddrType
type Node struct {
	//1-base
	//todo:没加锁不知道对不对
	routingTable RoutingTable //lru里放addrtype
	listening    bool
	kvStorage    Counter
}

func (this *Node) lookup(id *big.Int) {

}
func distance(addr1 *AddrType, addr2 *AddrType) *big.Int {
	return distance2(addr1.Id, addr2.Id)
}
func distance2(id1 *big.Int, id2 *big.Int) *big.Int {
	dis := big.NewInt(0)
	dis.Xor(id1, id2)
	return dis
}

//func (this *Node)getKClosest(id *big.Int)[]AddrType{
//	ret:=make([]AddrType,0)
//	bucket:=id.BitLen()
//	if this.routingTable[bucket].Size()==k{
//		tmp:=this.routingTable[bucket].ToArray()
//		for _,i:=range tmp{
//			ret =append(ret,i)
//		}
//		return ret
//	}else {
//		tmp:=this.routingTable[bucket].ToArray()
//		for _,i:=range tmp{
//			ret =append(ret,i)
//		}
//		var pq =make(PriorityQueue,0)
//		for i:=1;i<bucket;i++{
//			tmp=this.routingTable[i].ToArray()
//			for _,j:=range tmp {
//				pq.Push(&Item{&j,distance(&j,id),0})
//			}
//		}
//		for i:=0;len(ret)<k;i++{
//			item:=pq.Pop()
//			ret =append(ret,*item.(*Item).value)
//		}
//		for
//		return ret
//	}
//}

type LookUpType struct {
	addr  AddrType
	query bool
}

func (this *Node) lookUp(id *big.Int) []AddrType {

	lookupAddr := AddrType{Ip: ip, Id: chord.HashString(ip)}
	k_closest := this.routingTable.getKClosest(lookupAddr.Id)
	pq := make(PriorityQueue, 0)
	for _, tmp := range k_closest {
		pq.Push(&Item{tmp})
	}
}
func (this *Node) findNode(ip string, recv chan []AddrType) {
	client, e := rpc.Dial("tcp", ip)
	if e == nil {

	}
}

type PingRequest struct {
	RPCHeader
}

func ping(header RPCHeader, ip string) bool {
	var success bool
	for times := 0; times < 3; times++ { //todo:maybe we need to reduce retry times
		ch := make(chan bool)
		go func() {
			client, err := rpc.Dial("tcp", ip)
			if err != nil {
				ch <- false
				return
			} else {
				var success bool
				_ = client.Call("Node.RPCPing", header, &success)
				client.Close()
				if success {
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
			fmt.Println("ping ", ip, " time out")
			continue
		}
	}
	return false
}
func (this *Node) RPCPing(header RPCHeader, success *bool) error {
	this.routingTable.update(&header)
	*success = this.listening
	return nil
}

type KVPair struct {
	key string
	val string
}
type StoreRequest struct {
	pair   KVPair
	header RPCHeader
}

func (this *Node) RPCStore(request StoreRequest, success *bool) error {
	this.routingTable.update(&request.header)
	this.kvStorage.Mux.Lock()
	this.kvStorage.V[request.pair.key] = request.pair.val
	this.kvStorage.Mux.Unlock()
	return nil
}

type FindNodeRequest struct {
	header RPCHeader
	id     *big.Int
}

func (this *Node) RPCFindNode(request FindNodeRequest, closest *[]AddrType) error {
	this.routingTable.update(&request.header)
	*closest = this.routingTable.getKClosest(request.id)
	return nil
}

type FindValueRequest struct {
	header RPCHeader
	hashId *big.Int
	key    string
}
type FindValueReturn struct {
	addr []AddrType
	val  string
}

func (this *Node) RPCFindValue(request FindValueRequest, ret *FindValueReturn) error {
	this.routingTable.update(&request.header)
	this.kvStorage.Mux.Lock()
	val, ok := this.kvStorage.V[request.key]
	if ok {
		(*ret).val = val
	} else {
		(*ret).addr = this.routingTable.getKClosest(request.hashId)
	}
	return nil
}
