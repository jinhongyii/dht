package kademlia

//todo:add data validator
import (
	"chord"
	"fmt"
	"math/big"
	"net"
	"net/rpc"
	"sync"
	"time"
)

//用的时候应该让val是存k-v的服务器地址
type Client struct {
	wg       *sync.WaitGroup
	listener net.Listener
	Node_    Node
	Server   *rpc.Server
}

func (this *Client) Put(key string, val string) bool {
	this.Node_.IterativeStore(key, val, true)
	return true
}
func (this *Client) Get(key string) (Set, bool) {
	return this.Node_.IterativeFindValue(key)
}

//func (this *Client)Del(key string)bool{
//	NodeTodelete:=this.Node_.IterativeFindNode(key)
//	for _,node:=range NodeTodelete{
//		this.Node_.delete(node.Ip,key)
//	}
//	return true
//}
func (this *Client) Create() {
	this.Node_.KvStorage.Init()
	this.Node_.RoutingTable.Init()
	go this.Node_.Refresh()
	go this.Node_.Republish()
	go this.Node_.Expire()
	go this.Node_.Replicate()
}
func (this *Client) Join(ip string) bool {
	this.Node_.KvStorage.Init()
	this.Node_.RoutingTable.Init()
	if !this.Node_.ping(ip) {
		return false
	}
	this.Node_.RoutingTable.update(&Contact{Ip: ip, Id: chord.HashString(ip)})
	this.Node_.IterativeFindNode(ip)
	unoccupied := true
	for i := 1; i <= maxbucket; i++ {
		if unoccupied {
			if this.Node_.RoutingTable.buckets[i].Len() > 0 {
				unoccupied = false
			}
		} else {
			tmp := big.NewInt(0)
			tmp.Exp(big.NewInt(2), big.NewInt(int64(i-1)), nil)
			tmp.Xor(tmp, this.Node_.RoutingTable.Id)
			this.Node_.IterativeFindNode_(tmp)
			this.Node_.RoutingTable.RefreshMap[i] = time.Now().Add(tRefresh)
		}
	}
	go this.Node_.Refresh()
	go this.Node_.Republish()
	go this.Node_.Expire()
	go this.Node_.Replicate()
	return true
}
func (this *Client) Run(wg *sync.WaitGroup) {
	this.wg = wg
	wg.Add(1)
	var e error
	this.listener, e = net.Listen("tcp", this.Node_.RoutingTable.Ip)
	fmt.Println("listen at ", this.Node_.RoutingTable.Ip)
	if e != nil {
		fmt.Println(e)
		return
	}
	go this.Server.Accept(this.listener)
	this.Node_.Listening = true
	this.Node_.RoutingTable.Ip = chord.GetLocalAddress() + this.Node_.RoutingTable.Ip
	this.Node_.KvStorage.ip = this.Node_.RoutingTable.Ip //debug
	this.Node_.RoutingTable.Id = chord.HashString(this.Node_.RoutingTable.Ip)
}
func (this *Client) Quit() {
	this.wg.Done()
	this.Node_.Listening = false
	this.listener.Close()
}
func (this *Client) Dump() {
	fmt.Println("ip:", this.Node_.RoutingTable.Ip)
	for i := 1; i <= maxbucket; i++ {
		if this.Node_.RoutingTable.buckets[i].Len() > 0 {
			for tmp := this.Node_.RoutingTable.buckets[i].Front(); tmp != nil; tmp = tmp.Next() {
				fmt.Print(tmp.Value.(Contact).Ip, " ")
			}
			fmt.Print(" | ")
		}
	}
	fmt.Println()
	fmt.Println(this.Node_.KvStorage.storageMap)
}
func (this *Client) Ping(ip string) bool {
	return this.Node_.ping(ip)
}
