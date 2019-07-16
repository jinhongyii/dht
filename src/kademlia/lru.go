package kademlia

import (
	"container/list"
	"sync"
)

type LRUReplacer struct {
	size      int
	vals      list.List
	hashTable map[AddrType]*list.Element
	mux       sync.Mutex
}

func (this *LRUReplacer) Insert(value AddrType) {
	this.mux.Lock()
	pointer, ok := this.hashTable[value]
	if ok {
		this.vals.MoveToBack(pointer)
	} else {
		this.vals.PushBack(value)
		this.hashTable[value] = this.vals.Back()
	}
	this.mux.Unlock()
}
func (this *LRUReplacer) Victim(value *AddrType) bool {
	if len(this.hashTable) == 0 {
		return false
	}
	this.mux.Lock()
	*value = this.vals.Front().Value.(AddrType)
	delete(this.hashTable, *value)
	this.mux.Unlock()
	return true
}
func (this *LRUReplacer) Erase(value AddrType) bool {
	this.mux.Lock()
	pointer, ok := this.hashTable[value]
	if !ok {
		return false
	}
	this.vals.Remove(pointer)
	delete(this.hashTable, value)
	this.mux.Unlock()
	return true
}
func (this *LRUReplacer) Size() int {
	return len(this.hashTable)
}

func (this *LRUReplacer) ToArray() []AddrType {
	ret := make([]AddrType, 0)
	for i := this.vals.Front(); i != nil; i = i.Next() {
		ret = append(ret, i.Value.(AddrType))
	}
	return ret
}
func (this *LRUReplacer) front() AddrType {
	return this.vals.Front().Value.(AddrType)
}
func (this *LRUReplacer) UndoInsertion() {
	this.mux.Lock()
	tmp := this.vals.Back().Value.(AddrType)
	this.vals.Remove(this.vals.Back())
	delete(this.hashTable, tmp)
	this.mux.Unlock()
}
