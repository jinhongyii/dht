package common

import "sync"

type DhtNode interface {
	Get(k string) (string, bool)
	Put(k string, v string) bool
	Del(k string) bool
	Run(wg *sync.WaitGroup)
	Create()
	Join(addr string) bool
	Quit()
	Ping(addr string) bool
}
type dhtAdditive interface {
	DhtNode
	AppendTo()
	RemoveFrom()
}
