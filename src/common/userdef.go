package common

import (
	"chord"
	"fmt"
	"net/rpc"
	"strconv"
)

func NewNode(port int) *chord.Client {
	newClient := new(chord.Client)
	newClient.Node_.Ip = ":" + strconv.Itoa(port)
	newClient.Node_.Predecessor = nil

	newClient.Server = rpc.NewServer()
	err := newClient.Server.Register(&newClient.Node_)
	if err != nil {
		fmt.Println(err)
	}
	var res = newClient
	return res
}
