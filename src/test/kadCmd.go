package main

import (
	"chord"
	"common"
	"fmt"
	"kademlia"
	"math/big"
	"strconv"
)

func main() {
	var port int
	fmt.Scanln(&port)
	var node *kademlia.Client

	var joined = false
	var stop = false
	ip, err := chord.GetLocalPublicIpUseDnspod()
	if err != nil {
		fmt.Println(err)
		ip = chord.GetLocalAddress()
	}

	fmt.Println("address: " + ip)
	fmt.Println("local address:", chord.GetLocalAddress())
	for !stop {
		var str1, str2, str3 string

		fmt.Scanln(&str1, &str2, &str3)

		switch str1 {
		case "port":
			if !joined {
				if str2 != "" {
					port, _ = strconv.Atoi(str2)
				} else {
					port = 3410
				}
				println("port set to " + strconv.Itoa(port))
			} else {
				println("already joined")
			}
		case "create":
			if !joined {
				node = common.NewNode(port)
				node.Run()
				node.Create()
				joined = true
				println("create server at port " + strconv.Itoa(port))
			} else {
				println("already joined")
			}
		case "join":
			if !joined {
				node = common.NewNode(port)
				node.Run()
				if node.Join(str2) {
					joined = true
					fmt.Println("join successful at " + str2)
				} else {
					node.Quit()
					fmt.Println("join failed")
				}
			} else {
				println("already joined")
			}
		case "quit":
			if joined {
				node.Quit()
			}
			stop = true
		case "put":
			if joined {
				node.Put(str2, str3)
				fmt.Println("put successful")
			} else {
				fmt.Println("not joined")
			}
		case "get":
			if joined {
				result, success := node.Get(str2)
				if !success {
					fmt.Println("not found")
				} else {
					fmt.Println(str2, " => ", result)
				}
			} else {
				fmt.Println("not joined ")
			}
		case "ping":
			fmt.Println(kademlia.Ping(kademlia.Contact{big.NewInt(0), ""}, str2))
			//case "delete":
			//	if joined {
			//		success := node.Del(str2)
			//		if !success {
			//			fmt.Println("the key is not in the ring")
			//		}
			//	} else {
			//		fmt.Println("not joined in a chord ring")
			//	}
			//case "dump":
			//	if joined {
			//		node.Dump()
			//	}
			//}
		}

	}

}
