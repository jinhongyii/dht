package kademlia

import (
	"fmt"
	"math"
	"math/big"
	"time"
)

type RoutingTable struct {
	buckets    [maxbucket + 1]LRUReplacer
	Id         *big.Int
	Ip         string
	RefreshMap [maxbucket + 1]time.Time
}

func (this *RoutingTable) update(header *Contact) {
	bucketid := distance2(header.Id, this.Id).BitLen()
	this.buckets[bucketid].mux.Lock()
	this.buckets[bucketid].Insert(Contact{Ip: header.Ip, Id: header.Id})
	if this.buckets[bucketid].Size() > K {

		if online, contact := Ping(Contact{Ip: this.Ip, Id: this.Id}, this.buckets[bucketid].Front().Value.(Contact).Ip); online {
			this.buckets[bucketid].UndoInsertion()
			this.buckets[bucketid].Insert(contact)
		} else {
			var deleted Contact
			this.buckets[bucketid].Victim(&deleted)
		}
	}
	this.buckets[bucketid].mux.Unlock()
}
func (this *RoutingTable) failNode(contact Contact) {
	bucketid := distance2(contact.Id, this.Id).BitLen()
	if this.buckets[bucketid].Exist(contact) {
		this.buckets[bucketid].mux.Lock()
		this.buckets[bucketid].Erase(contact)
		this.buckets[bucketid].mux.Unlock()
	}
}
func (this *RoutingTable) GetClosest(id *big.Int, requiredNum int) []Contact {
	ret := make(Contacts, 0)

	dis := distance2(id, this.Id)
	id_bits := fmt.Sprintf("%b", dis)
	totlen := len(id_bits)
	var filled [maxbucket + 1]bool
	for i, bit := range id_bits {
		if bit == '1' {
			if full := tryFill(&this.buckets[totlen-i], &ret, requiredNum); full {
				return ret
			}
			filled[totlen-i] = true
		}
	}
	for i := 1; i <= maxbucket; i++ {
		if !filled[i] {
			if full := tryFill(&this.buckets[i], &ret, requiredNum); full {
				return ret
			}
		}
	}
	return ret

}
func tryFill(replacer *LRUReplacer, contacts *Contacts, required int) bool {
	for i := replacer.Front(); i != nil; i = i.Next() {
		if len(*contacts) == required {
			return true
		}
		*contacts = append(*contacts, i.Value.(Contact))
	}
	return len(*contacts) == required
}
func (rt *RoutingTable) Init() {

	for i := 0; i <= maxbucket; i++ {
		rt.buckets[i].Init()
	}
}
func (this *RoutingTable) calculateExpire(id *big.Int) time.Duration {
	dis := distance2(id, this.Id)
	bucketid := dis.BitLen()
	sum := 0
	for i := 1; i < bucketid; i++ {
		sum += this.buckets[i].Len()
	}
	for i := this.buckets[bucketid].Front(); i != nil; i = i.Next() {
		if distance2(id, i.Value.(Contact).Id).Cmp(dis) < 0 {
			sum++
		}
	}
	return time.Duration(int64(float64(tExpire) * math.Exp(1-float64(sum)/float64(K)))) //这里没有按照xlattice上面的写，用的负指数相关
}
func (this *RoutingTable) getbucketid(id *big.Int) int {
	return distance2(id, this.Id).BitLen()
}
func (this *Node) refresh() {
	unoccupied := true
	for i := 1; i <= maxbucket; i++ {
		if unoccupied {
			if this.RoutingTable.buckets[i].Len() > 0 {
				unoccupied = false
			}
		} else {
			if time.Now().After(this.RoutingTable.RefreshMap[i]) {
				tmp := big.NewInt(0)
				tmp.Exp(big.NewInt(2), big.NewInt(int64(i-1)), nil)
				tmp.Xor(tmp, this.RoutingTable.Id)
				this.IterativeFindNode_(tmp)
				this.RoutingTable.RefreshMap[i].Add(tRefresh)
			}
		}
	}
}
