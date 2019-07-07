package main

import (
	"chord"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
)

func main(){
	var node chord.Node
	var port=3410
	var joined =false
	_ = rpc.Register(&node)
	rpc.HandleHTTP()



	var stop=false
	fmt.Println("local address: "+chord.GetLocalAddress())
	for ;!stop;{
		var str1,str2,str3 string

		fmt.Scanln(&str1,&str2,&str3)

		switch str1 {
		case "port":
			if(!joined) {
				if(str2!=""){
					port,_=strconv.Atoi(str2)
				} else{
					port=3410
				}
				println("port set to "+strconv.Itoa(port))
			}else {
				println("already joined")
			}
		case "create":
			if(!joined) {
				l, e := net.Listen("tcp", ":"+strconv.Itoa(port))
				if e != nil {
					log.Fatal("listen error", e)
				}
				go http.Serve(l, nil)
				node.Create(port)
				go node.Stabilize()
				go node.Fix_fingers()
				go node.CheckPredecessor()
				joined = true
				println("create server at port "+strconv.Itoa(port))
			}else {
				println("already joined")
			}
		case "join":
			if(!joined) {
				l, e := net.Listen("tcp", ":"+strconv.Itoa(port))
				if e != nil {
					log.Fatal("listen error", e)
				}
				go http.Serve(l, nil)
				node.Join(str2, port)
				joined = true
				go node.Stabilize()
				go node.Fix_fingers()
				go node.CheckPredecessor()
				fmt.Println("join "+str2)
			}else {
				println("already joined")
			}
		case "quit":
			node.Quit()
			stop=true
		case "put":
			node.Put(str2,str3)
		case "get":
			result,err:=node.Get(str2)
			if(err!=nil){
				fmt.Println(err)
			}else {
				fmt.Println(str2 + " => " + result)
			}
		case "delete":
			node.Delete(str2)
		case "dump":
			node.Dump()
		}



	}
}