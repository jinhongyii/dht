package kademlia

import (
	"fmt"
	"sync"
	"time"
)

type MemStore struct {
	storageMap   map[string]string //todo:change to multimap
	Mux          sync.Mutex
	expireMap    map[string]time.Time
	republishMap map[string]time.Time
	replicateMap map[string]time.Time
}

//固定重置replicatemap
func (this *MemStore) put(key string, val string, republish bool, expire time.Duration, replicate bool) {
	this.Mux.Lock()
	this.storageMap[key] = val
	this.Mux.Unlock()
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
	this.Mux.Lock()
	val, ok := this.storageMap[key]
	this.Mux.Unlock()
	if ok {
		return val, true
	} else {
		return "", false
	}
}
func (this *Node) expire() {
	this.KvStorage.Mux.Lock()
	for key, t := range this.KvStorage.expireMap {
		if t.After(time.Now()) {
			delete(this.KvStorage.expireMap, key)
			delete(this.KvStorage.replicateMap, key)
			delete(this.KvStorage.storageMap, key)
		}
	}
	this.KvStorage.Mux.Unlock()
}
func (this *Node) replicate() {
	keysToReplicate := make([]KVPair, 0)
	this.KvStorage.Mux.Lock()
	for key, t := range this.KvStorage.replicateMap {
		if t.After(time.Now()) {
			val, _ := this.KvStorage.get(key)
			keysToReplicate = append(keysToReplicate, KVPair{Key: key, Val: val})
		}
	}
	this.KvStorage.Mux.Unlock()
	for _, pair := range keysToReplicate {
		this.IterativeStore(pair.Key, pair.Val, false)
	}
}
func (this *Node) republish() {
	KeysToRepublish := make([]KVPair, 0)
	this.KvStorage.Mux.Lock()
	for key, t := range this.KvStorage.republishMap {
		if t.After(time.Now()) {
			val, _ := this.KvStorage.get(key)
			KeysToRepublish = append(KeysToRepublish, KVPair{Key: key, Val: val})
		}
	}
	this.KvStorage.Mux.Unlock()
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

		time.Sleep(120 * time.Second)
		fmt.Println(this.RoutingTable.Ip, "start to expire")
		this.expire()
	}
}
func (this *Node) Replicate() {
	for this.Listening {

		time.Sleep(120 * time.Second)
		fmt.Println(this.RoutingTable.Ip, "start to replicate")
		this.replicate()
	}
}
func (this *Node) Republish() {
	for this.Listening {
		time.Sleep(120 * time.Second)
		fmt.Println("start to republish")
		this.republish()
	}
}
func (this *Node) Refresh() {
	for this.Listening {
		time.Sleep(120 * time.Second)
		fmt.Println("start to refresh")
		this.refresh()
	}
}
