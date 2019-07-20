package common

import (
	"fmt"
	"kademlia"
	"net/rpc"
	"strconv"
)

//func NewNode(port int) DhtNode {
//	newClient := new(chord.Client)
//	newClient.Node_.Ip = ":" + strconv.Itoa(port)
//	newClient.Node_.Predecessor = nil
//
//	newClient.Server = rpc.NewServer()
//	err := newClient.Server.Register(&newClient.Node_)
//	if err != nil {
//		fmt.Println(err)
//	}
//	var res = newClient
//	return res
//}
func NewNode(port int) *kademlia.Client {
	newClient := new(kademlia.Client)
	newClient.Node_.RoutingTable.Ip = ":" + strconv.Itoa(port)
	newClient.Server = rpc.NewServer()
	err := newClient.Server.Register(&newClient.Node_)
	if err != nil {
		fmt.Println(err)
	}
	return newClient
}
