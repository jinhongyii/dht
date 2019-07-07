package chord

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/rpc"
	"strconv"
	"sync"
	"time"
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
	Id big.Int
}
type Node struct {
	id *big.Int
	ip string
	kvStorage   Counter
	successors  [m+1]FingerType
	finger      [m+1]FingerType
	predecessor *FingerType
}
func (this *Node) Merge(kvpairs *map[string]string ,success *bool)error{
	this.kvStorage.mux.Lock()
	for k,v:=range *kvpairs {
		this.kvStorage.V[k]=v
	}
	this.kvStorage.mux.Unlock()
	return nil
}
//create a new ring
func (this *Node) Create(port int) {
	this.predecessor = nil
	this.ip=GetLocalAddress()+":"+strconv.Itoa(port)
	this.id=hashString(this.ip)
	this.kvStorage.V=make(map[string]string)
	this.successors[1].Ip =this.ip
	this.successors[1].Id =*this.id
}
func (this *Node) GetKeyValMap(a *int, b *map[string]string) error {
	this.kvStorage.mux.Lock()
	b = &(this.kvStorage.V)
	this.kvStorage.mux.Unlock()
	return nil
}
func (this* Node) GetPredecessor(a *int,b *FingerType)error{
	if this.predecessor!=nil {
		*b = *this.predecessor
	}else {
		*b=FingerType{}
	}
	return nil
}
func (this *Node)Stabilize()  {
	for ;;{
		this.stabilize()
		time.Sleep(100*time.Millisecond)
	}
}
func (this *Node) stabilize(){
	client,e:=rpc.DialHTTP("tcp",this.successors[1].Ip)
	if e!=nil{
		log.Fatal("dialing:", e)
	}
	var p FingerType
	_ = client.Call("Node.GetPredecessor", 0, &p)
	client.Close()
	var tmp big.Int
	if (p.Ip!="" && p.Id.Cmp(&tmp)!=0) && between(this.id,&p.Id,&this.successors[1].Id,false) {
		this.successors[1]=p
	}
	client,e=rpc.DialHTTP("tcp",this.successors[1].Ip)
	_ =client.Call("Node.Notify", &FingerType{this.ip,*this.id},nil)
	client.Close()

}
func (this* Node)CheckPredecessor()  {
	for ;;{
		this.checkPredecessor()
		time.Sleep(100*time.Millisecond)
	}
}
func (this *Node) checkPredecessor(){
	if(this.predecessor!=nil) {
		client, e := rpc.DialHTTP("tcp", this.predecessor.Ip)
		if (e != nil) {
			this.predecessor = nil
		} else {
			client.Close()
		}
	}

}
func (this* Node)Fix_fingers(){
	var fingerEntry =1
	for ;;{
		this.fix_fingers(&fingerEntry)
		time.Sleep(100*time.Millisecond)
	}
}
func (this* Node) fix_fingers(fingerEntry *int){
	_ = this.FindSuccessor(&FindRequest{*jump(this.id, *fingerEntry), 0}, &this.finger[*fingerEntry])
	*fingerEntry++
	if *fingerEntry> m {
		*fingerEntry=1
	}
}
func (this* Node)CompleteMigrate(otherNode *FingerType,lala *int)error{
	var deletion []string
	this.kvStorage.mux.Lock()
	if otherNode.Id.Cmp(this.id)< 0 {

		for k:=range this.kvStorage.V {
			k_hash:=hashString(k)
			if k_hash.Cmp(&otherNode.Id)<= 0 || k_hash.Cmp(&this.predecessor.Id)>0 {
				deletion=append(deletion, k)
			}
		}
	} else{
		for k:=range this.kvStorage.V {
			k_hash:=hashString(k)
			if k_hash.Cmp(&otherNode.Id)<= 0 {
				deletion=append(deletion,k)
			}
		}
	}
	for _,v:=range deletion{
		delete(this.kvStorage.V, v)
	}
	this.kvStorage.mux.Unlock()
	return nil
}
//join a ring containing OtherNode
func (this *Node) Join(otherNode string, port int) {
	this.ip=GetLocalAddress()+":"+strconv.Itoa(port)
	this.id=hashString(this.ip)
	this.kvStorage.V=make(map[string]string)
	this.predecessor = nil
	client, e := rpc.DialHTTP("tcp", otherNode)
	if e != nil {
		log.Fatal("dialing:", e)
	}
	err := client.Call("Node.FindSuccessor",& FindRequest{*this.id,0}, &this.successors[1])
	if err != nil {
		fmt.Println(err)
	}
	var receivedMap map[string]string
	var p FingerType
	client.Call("Node.GetKeyValMap", 0, &receivedMap)
	client.Call("Node.GetPredecessor",0,&p)
	this.kvStorage.mux.Lock()
	for k, v := range receivedMap {
		var k_hash=hashString(k)
		if p.Id.Cmp(&this.successors[1].Id)<0 {
			if k_hash.Cmp(this.id)<= 0 {
				this.kvStorage.V[k]=v
			}
		} else{
			if(this.id.Cmp(&this.successors[1].Id)<0) {
				if k_hash.Cmp(this.id)<= 0 || k_hash.Cmp(&p.Id)>0{
					this.kvStorage.V[k]=v
				}
			}else {
				if k_hash.Cmp(this.id)<=0 &&k_hash.Cmp(&p.Id)>0{
					this.kvStorage.V[k]=v
				}
			}
		}
	}
	this.kvStorage.mux.Unlock()
	client.Close()
	client, e = rpc.DialHTTP("tcp", this.successors[1].Ip)
	if e != nil {
		log.Fatal("dialing:", e)
	}
	err=client.Call("Node.CompleteMigrate",& FingerType{this.ip,*this.id},nil)
	err = client.Call("Node.Notify",& FingerType{this.ip,*this.id}, nil)
	if err != nil {
		fmt.Println(err)
	}
	client.Close()
}
func (this *Node) Notify(otherNode *FingerType, lalala *int) error {
	if this.predecessor == nil || between(&this.predecessor.Id, &otherNode.Id, this.id, false) {
		this.predecessor = new(FingerType)
		*this.predecessor=*otherNode
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
	if this.successors[1].Id.Cmp(this.id)==0 {
		*successor=this.successors[1]
	}else if between(this.id, &request.Id, hashString(this.successors[1].Ip), true) {
		successor.Ip = this.successors[1].Ip
		successor.Id =this.successors[1].Id
	} else {
		next_step := this.closest_preceding_node(&request.Id)
		client, e := rpc.DialHTTP("tcp", next_step.Ip);
		if e != nil {
			log.Fatal("dialing:", e)
		}
		var result FingerType
		request.Times++
		err := client.Call("Node.FindSuccessor", &request, &result)
		if err != nil {
			fmt.Println("finding:", e)
		}
		*successor = result
		client.Close()
	}
	return nil
}
func (this *Node) closest_preceding_node(id *big.Int) FingerType {
	for i := m; i > 0; i-- {
		if between(this.id, &this.finger[i].Id, id, false) {
			return this.finger[i]
		}
	}
	return this.successors[1]
}
func (this *Node) Put_(args *ChordKV,success *bool)error{
	this.kvStorage.mux.Lock()
	this.kvStorage.V[args.Key]=args.Val
	this.kvStorage.mux.Unlock()
	fmt.Println("put "+args.Key +" => "+args.Val)
	return nil
}
func (this *Node)Get_(key *string,val *string) error {
	this.kvStorage.mux.Lock()
	*val=this.kvStorage.V[*key]
	if *val=="" {
		this.kvStorage.mux.Unlock()
		return errors.New("not found key")
	}
	this.kvStorage.mux.Unlock()
	fmt.Println("get "+*key +" => "+*val)
	return nil
}
func (this *Node)Delete_(key *string,success *bool)error{
	this.kvStorage.mux.Lock()
	delete(this.kvStorage.V, *key)
	this.kvStorage.mux.Unlock()
	fmt.Println("delete "+*key)
	return nil
}
func (this *Node) Put(key string ,val string){
	k_hash:=hashString(key)
	var successor FingerType
	_ = this.FindSuccessor(&FindRequest{*k_hash,0},&successor)
	client, err := rpc.DialHTTP("tcp", successor.Ip)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	err=client.Call("Node.Put_",&ChordKV{key,val},nil)
	client.Close()

}
func (this *Node) Get(key string)(string,error){
	var val string
	k_hash:=hashString(key)
	var successor FingerType
	_ = this.FindSuccessor(&FindRequest{*k_hash,0},&successor)
	client, err := rpc.DialHTTP("tcp", successor.Ip)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	err=client.Call("Node.Get_",&key,&val)
	client.Close()
	return val,err
}
func (this* Node)Delete(key string){
	k_hash:=hashString(key)
	var successor FingerType
	_ = this.FindSuccessor(&FindRequest{*k_hash,0},&successor)
	client, err := rpc.DialHTTP("tcp", successor.Ip)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	err=client.Call("Node.Delete_",&key,nil)
	client.Close()
}
func (this *Node)Dump(){
	fmt.Println(this.kvStorage.V)
	fmt.Println("successor: "+this.successors[1].Ip)
	fmt.Println("predecessor: "+this.predecessor.Ip)

}
func (this *Node) Quit(){
	client, err := rpc.DialHTTP("tcp", this.successors[1].Ip)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	err=client.Call("Node.Merge",&this.kvStorage.V,nil)
	client.Close()
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
