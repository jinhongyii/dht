package kademlia

import (
	"fmt"
	"sync"
	"time"
)

type MemStore struct {
	ip           string            //debug
	storageMap   map[string]string //todo:change to multimap
	StoreMux     sync.Mutex
	expireMap    map[string]time.Time
	ReplicateMux sync.Mutex
	RepublishMux sync.Mutex
	republishMap map[string]time.Time
	replicateMap map[string]time.Time
}

//固定重置replicatemap
func (this *MemStore) put(key string, val string, republish bool, expire time.Duration, replicate bool) {
	//fmt.Println(this.ip," :put lock ")
	this.StoreMux.Lock()
	this.storageMap[key] = val
	//fmt.Println(this.ip," :put unlock")
	this.StoreMux.Unlock()
	//把节点变成源
	if republish {
		this.republishMap[key] = time.Now().Add(tRepublish)
		delete(this.expireMap, key)
		delete(this.expireMap, key)
	} else if _, ok := this.republishMap[key]; !ok { //本来不是个源
		if expire != 0 {
			this.expireMap[key] = time.Now().Add(expire)
		}
		if replicate {
			this.replicateMap[key] = time.Now().Add(tReplicate)
		}
	}

}
func (this *MemStore) get(key string) (string, bool) {
	//fmt.Println(this.ip," :get lock")
	this.StoreMux.Lock()
	val, ok := this.storageMap[key]
	//fmt.Println(this.ip," :get unlock")
	this.StoreMux.Unlock()
	if ok {
		return val, true
	} else {
		return "", false
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
	deleteList := make([]string, 0)
	for key, t := range this.KvStorage.expireMap {
		if time.Now().After(t) {
			//delete(this.KvStorage.expireMap, key)
			deleteList = append(deleteList, key)
			delete(this.KvStorage.replicateMap, key)
			delete(this.KvStorage.storageMap, key)
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
	for key, t := range this.KvStorage.replicateMap {
		if time.Now().After(t) {
			val, _ := this.KvStorage.get(key)
			keysToReplicate = append(keysToReplicate, KVPair{Key: key, Val: val})
		}
	}
	this.KvStorage.ReplicateMux.Unlock()
	//fmt.Println(this.RoutingTable.Ip," :replicate replicateMux unlock")

	for _, pair := range keysToReplicate {
		this.IterativeStore(pair.Key, pair.Val, false)
	}
}
func (this *Node) republish() {
	KeysToRepublish := make([]KVPair, 0)
	this.KvStorage.RepublishMux.Lock()
	//fmt.Println(this.RoutingTable.Ip," :republish republishMux got")
	for key, t := range this.KvStorage.republishMap {
		if time.Now().After(t) {
			val, _ := this.KvStorage.get(key)
			KeysToRepublish = append(KeysToRepublish, KVPair{Key: key, Val: val})
		}
	}
	this.KvStorage.RepublishMux.Unlock()
	//fmt.Println(this.RoutingTable.Ip," :republish republishMux unlock")
	for _, pair := range KeysToRepublish {
		this.IterativeStore(pair.Key, pair.Val, true)
	}
}
func (this *MemStore) Init() {
	this.storageMap = make(map[string]string)
	this.expireMap = make(map[string]time.Time)
	this.republishMap = make(map[string]time.Time)
	this.replicateMap = make(map[string]time.Time)
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
