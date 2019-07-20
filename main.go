package main

import (
	"chord"
	"common"
	"fmt"
	"strconv"
	"sync"
)

func main() {
	var port int
	fmt.Scanln(&port)
	node := common.NewNode(port)

	var joined = false
	var stop = false
	fmt.Println("local address: " + chord.GetLocalAddress())
	wg := sync.WaitGroup{}
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
				(*node).Run(&wg)
				(*node).Create()
				joined = true
				println("create server at port " + strconv.Itoa(port))
			} else {
				println("already joined")
			}
		case "join":
			if !joined {
				(*node).Run(&wg)
				(*node).Join(str2)
				joined = true
				fmt.Println("join successful at " + str2)
			} else {
				println("already joined")
			}
		case "quit":
			if joined {
				(*node).Quit()
			}
			stop = true
		case "put":
			if joined {
				(*node).Put(str2, str3)
				fmt.Println("put successful")
			} else {
				fmt.Println("not joined in a chord ring")
			}
		case "get":
			if joined {
				result, success := (*node).Get(str2)
				if !success {
					fmt.Println("not found")
				} else {
					fmt.Println(str2 + " => " + result)
				}
			} else {
				fmt.Println("not joined in a chord ring")
			}
			//case "delete":
			//	if joined {
			//		success := (*node).Del(str2)
			//		if !success {
			//			fmt.Println("the key is not in the ring")
			//		}
			//	} else {
			//		fmt.Println("not joined in a chord ring")
			//	}
			//case "dump":
			//	if joined {
			//		(*node).Dump()
			//	}
			//}
		}

	}
	wg.Wait()
}
