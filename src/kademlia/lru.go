package kademlia

import (
	"container/list"
	"sync"
)

type LRUReplacer struct {
	size      int
	vals      list.List
	hashTable map[string]*list.Element
	mux       sync.Mutex
}

func (this *LRUReplacer) Insert(value Contact) {
	//this.mux.Lock()
	pointer, ok := this.hashTable[value.Ip]
	if ok {
		this.vals.MoveToBack(pointer)
	} else {
		this.vals.PushBack(value)

	}
	this.hashTable[value.Ip] = this.vals.Back()
	//this.mux.Unlock()
}
func (this *LRUReplacer) Victim(value *Contact) bool {
	if len(this.hashTable) == 0 {
		return false
	}
	//this.mux.Lock()
	*value = this.vals.Front().Value.(Contact)
	delete(this.hashTable, value.Ip)
	this.vals.Remove(this.vals.Front())
	//this.mux.Unlock()
	return true
}
func (this *LRUReplacer) Erase(value Contact) bool {
	//this.mux.Lock()
	pointer, ok := this.hashTable[value.Ip]
	if !ok {
		return false
	}
	this.vals.Remove(pointer)
	delete(this.hashTable, value.Ip)
	//this.mux.Unlock()
	return true
}
func (this *LRUReplacer) Size() int {
	return this.vals.Len()
}

func (this *LRUReplacer) ToArray() []Contact {
	ret := make([]Contact, 0)
	for i := this.vals.Front(); i != nil; i = i.Next() {
		ret = append(ret, i.Value.(Contact))
	}
	return ret
}

func (this *LRUReplacer) UndoInsertion() {
	//this.mux.Lock()
	//fmt.Println("undo insertion: ",this.ToArray())
	tmp := this.vals.Back().Value.(Contact)
	this.vals.Remove(this.vals.Back())
	delete(this.hashTable, tmp.Ip)
	//this.mux.Unlock()
}
func (this *LRUReplacer) Init() {
	this.vals.Init()
	this.hashTable = make(map[string]*list.Element)
}
func (this *LRUReplacer) Front() *list.Element {
	return this.vals.Front()
}
func (this *LRUReplacer) Len() int {
	return this.vals.Len()
}
func (this *LRUReplacer) Exist(contact Contact) bool {
	_, ok := this.hashTable[contact.Ip]
	return ok
}
