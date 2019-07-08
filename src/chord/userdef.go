package chord

import (
	"net/rpc"
	"strconv"
)

func NewNode(port int) *Client {
	newClient := new(Client)
	newClient.Node_.Ip = ":" + strconv.Itoa(port)
	newClient.Node_.Predecessor = nil
	newClient.Node_.Successors[1] = FingerType{newClient.Node_.Ip, newClient.Node_.Id}
	newClient.server = rpc.NewServer()
	newClient.server.Register(&newClient.Node_)
	var res = newClient
	return res
}
