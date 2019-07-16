package kademlia

import "math/big"

type RoutingTable struct {
	buckets [maxbucket]LRUReplacer
	id      *big.Int
	ip      string
}

func (this *RoutingTable) update(header *RPCHeader) {
	bucketid := distance2(header.Id, this.id).BitLen()
	this.buckets[bucketid].Insert(AddrType{Ip: header.Ip, Id: header.Id})
	if this.buckets[bucketid].Size() > k {
		if !ping(RPCHeader{Ip: this.ip, Id: this.id}, this.buckets[bucketid].front().Ip) {
			var deleted AddrType
			this.buckets[bucketid].Victim(&deleted)
		} else {
			this.buckets[bucketid].UndoInsertion()
		}
	}
}
func (this *RoutingTable) getKClosest(id *big.Int) []AddrType {
	ret := make([]AddrType, 0)
	bucketid := id.BitLen()
	if this.buckets[bucketid].Size() == k {
		tmp := this.buckets[bucketid].ToArray()
		for _, i := range tmp {
			ret = append(ret, i)
		}
		return ret
	} else {
		tmp := this.buckets[bucketid].ToArray()
		for _, i := range tmp {
			ret = append(ret, i)
		}
		var pq = make(PriorityQueue, 0)
		for i := 1; i < k && i != bucketid; i++ {
			tmp = this.buckets[i].ToArray()
			for _, j := range tmp {
				pq.Push(&Item{j, distance2(j.Id, id), 0})
			}
		}
		for i := 0; len(ret) < k && pq.Len() > 0; i++ {
			item := pq.Pop()
			ret = append(ret, item.(*Item).value)
		}
		return ret
	}
}
