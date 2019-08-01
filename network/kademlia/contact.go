package kademlia

import "math/big"

type Contact struct {
	Id *big.Int //server's Id
	Ip string   //server's Ip
}

func (this *Contact) Cmp(other *Contact) {
	return
}

type Contacts []Contact

func (this *Contacts) Len() int {
	return len(*this)
}
func (this *Contacts) Swap(i int, j int) {
	(*this)[i].Ip, (*this)[i].Id = (*this)[j].Ip, (*this)[j].Id
}
func (this *Contacts) Less(i, j int) bool {
	return (*this)[i].Id.Cmp((*this)[j].Id) < 0
}
func (this *Contacts) Push(x interface{}) {
	*this = append(*this, x.(Contact))
}
func (this *Contacts) Pop() interface{} {
	old := *this
	n := len(old)
	item := old[n-1]
	*this = old[0 : n-1]
	return item
}
