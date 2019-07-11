package chord

import (
	"fmt"
	"net/rpc"
	"strconv"
)

func NewNode(port int) *Client {
	newClient := new(Client)
	newClient.Node_.Ip = ":" + strconv.Itoa(port)
	newClient.Node_.Predecessor = nil

	newClient.server = rpc.NewServer()
	err := newClient.server.Register(&newClient.Node_)
	if err != nil {
		fmt.Println(err)
	}
	var res = newClient
	return res
}
