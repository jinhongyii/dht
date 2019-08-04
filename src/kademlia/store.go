package kademlia

import (
	"fmt"
	"sync"
	"time"
)

type Set map[string]struct{}

func (this *Set) Put(k string) {
	(*this)[k] = struct{}{}
}
func (this Set) p() *Set {
	return &this
}
func (this *Set) Exist(k string) bool {
	_, ok := (*this)[k]
	return ok
}
func (this *Set) Delete(k string) {
	delete(*this, k)
}
func (this *Set) Len() int {
	return len(*this)
}

type MemStore struct {
	ip           string //debug
	storageMap   map[string]Set
	StoreMux     sync.Mutex
	expireMap    map[KVPair]time.Time
	ReplicateMux sync.Mutex
	RepublishMux sync.Mutex
	republishMap map[KVPair]time.Time
	replicateMap map[KVPair]time.Time
}

//固定重置replicatemap
func (this *MemStore) put(key string, val string, republish bool, expire time.Time) {
	//fmt.Println(this.ip," :put lock ")
	this.StoreMux.Lock()
	s, ok := this.storageMap[key]
	if ok {
		s.Put(val)
	} else {
		this.storageMap[key] = make(Set)
		this.storageMap[key].p().Put(val)
	}
	//fmt.Println(this.ip," :put unlock")
	this.StoreMux.Unlock()
	pair := KVPair{Key: key, Val: val}
	//把节点变成源
	if republish {
		this.republishMap[pair] = time.Now().Add(tRepublish)
		delete(this.expireMap, pair)
	} else if _, ok := this.republishMap[pair]; !ok { //本来不是个源

		this.replicateMap[pair] = time.Now().Add(tReplicate)

	}

}
func (this *MemStore) get(key string) (Set, bool) {
	//fmt.Println(this.ip," :get lock")
	this.StoreMux.Lock()
	val, ok := this.storageMap[key]
	//fmt.Println(this.ip," :get unlock")
	this.StoreMux.Unlock()
	if ok {
		return val, true
	} else {
		return nil, false
	}
}
func (this *Node) expire() {
	this.KvStorage.ReplicateMux.Lock()
	//fmt.Println(this.RoutingTable.Ip," :expire replicateMux got")
	this.KvStorage.RepublishMux.Lock()
	//fmt.Println(this.RoutingTable.Ip," :expire republishMux got")
	//fmt.Println(this.RoutingTable.Ip," :expire lock")
	this.KvStorage.StoreMux.Lock()
	//fmt.Println(this.RoutingTable.Ip," :expire lock got")
	deleteList := make([]KVPair, 0)
	for pair, t := range this.KvStorage.expireMap {
		if time.Now().After(t) {
			//delete(this.KvStorage.expireMap, key)
			deleteList = append(deleteList, pair)
			delete(this.KvStorage.replicateMap, pair)
			this.KvStorage.storageMap[pair.Key].p().Delete(pair.Val)
		}
	}
	for _, key := range deleteList {
		delete(this.KvStorage.replicateMap, key)
		fmt.Println(this.RoutingTable.Ip, " expire ", key)
	}
	this.KvStorage.RepublishMux.Unlock()
	this.KvStorage.ReplicateMux.Unlock()
	//fmt.Println(this.RoutingTable.Ip," :expire unlock")
	this.KvStorage.StoreMux.Unlock()
}
func (this *Node) replicate() {
	keysToReplicate := make([]KVPair, 0)
	this.KvStorage.ReplicateMux.Lock()
	//fmt.Println(this.RoutingTable.Ip," :replicate replicateMux got")
	for pair, t := range this.KvStorage.replicateMap {
		if time.Now().After(t) {
			keysToReplicate = append(keysToReplicate, KVPair{Key: pair.Key, Val: pair.Val})
		}
	}
	this.KvStorage.ReplicateMux.Unlock()
	//fmt.Println(this.RoutingTable.Ip," :replicate replicateMux unlock")

	for _, pair := range keysToReplicate {
		this.IterativeStore(pair.Key, pair.Val, false, this.KvStorage.expireMap[pair])
	}
}
func (this *Node) republish() {
	KeysToRepublish := make([]KVPair, 0)
	this.KvStorage.RepublishMux.Lock()
	//fmt.Println(this.RoutingTable.Ip," :republish republishMux got")
	for pair, t := range this.KvStorage.republishMap {
		if time.Now().After(t) {
			KeysToRepublish = append(KeysToRepublish, KVPair{Key: pair.Key, Val: pair.Val})
		}
	}
	this.KvStorage.RepublishMux.Unlock()
	//fmt.Println(this.RoutingTable.Ip," :republish republishMux unlock")
	for _, pair := range KeysToRepublish {
		this.IterativeStore(pair.Key, pair.Val, true, this.KvStorage.expireMap[pair])
	}
}
func (this *MemStore) Init() {
	this.storageMap = make(map[string]Set)
	this.expireMap = make(map[KVPair]time.Time)
	this.republishMap = make(map[KVPair]time.Time)
	this.replicateMap = make(map[KVPair]time.Time)
}
func (this *Node) Expire() {
	for this.Listening {

		time.Sleep(60 * time.Second)
		fmt.Println(this.RoutingTable.Ip, "start to expire")
		this.expire()
	}
}
func (this *Node) Replicate() {
	for this.Listening {

		time.Sleep(60 * time.Second)
		fmt.Println(this.RoutingTable.Ip, "start to replicate")
		this.replicate()
	}
}
func (this *Node) Republish() {
	for this.Listening {
		time.Sleep(60 * time.Second)
		fmt.Println(this.RoutingTable.Ip, "start to republish")
		this.republish()
	}
}
func (this *Node) Refresh() {
	for this.Listening {
		time.Sleep(60 * time.Second)
		fmt.Println(this.RoutingTable.Ip, "start to refresh")
		this.refresh()
	}
}
