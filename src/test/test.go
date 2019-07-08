package main

import (
	"chord"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"strconv"
	"time"
)

func init1() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

type keyval struct {
	key string
	val string
}

func main() {
	init1()
	var servers [100]*rpc.Server
	var nodes [100]chord.Node
	var joined [100]bool
	servers[0] = rpc.NewServer()
	_ = servers[0].Register(&nodes[0])
	localAddress := chord.GetLocalAddress()
	fmt.Println("local address: " + localAddress)
	port := 1000
	l, e := net.Listen("tcp", ":"+strconv.Itoa(port))
	if e != nil {
		log.Fatal("listen error", e)
	}
	go servers[0].Accept(l)
	nodes[0].Create(port)
	go nodes[0].Stabilize()
	go nodes[0].Fix_fingers()
	go nodes[0].CheckPredecessor()
	joined[0] = true
	kvMap := make(map[string]string)

	var nodecnt = 1
	for i := 0; i < 5; i++ {
		//join 15 nodes
		for j := 0; j < 15; j++ {
			var index = i*15 + j + 1
			servers[index] = rpc.NewServer()
			_ = servers[index].Register(&nodes[index])
			port++
			listener, e := net.Listen("tcp", ":"+strconv.Itoa(port))
			if e != nil {
				log.Fatal("listen error", e)
			}
			go servers[index].Accept(listener)
			nodes[index].Join(localAddress+":"+strconv.Itoa(1000+5*i), port)
			go nodes[index].Stabilize()
			go nodes[index].Fix_fingers()
			go nodes[index].CheckPredecessor()
			joined[index] = true
			time.Sleep(3 * time.Second)
			fmt.Println("port ", port, " joined at 1000")
		}
		nodecnt += 15
		time.Sleep(30 * time.Second)
		//put 300 kv
		for j := 0; j < 300; j++ {
			k := RandStringRunes(30)
			v := RandStringRunes(30)
			kvMap[k] = v
			nodes[rand.Intn(nodecnt+1)+i*5].Put(k, v)
		}
		//get 200 kv and check correctness
		var keyList [200]string
		cnt := 0
		for k, v := range kvMap {
			if cnt == 200 {
				break
			}
			fetchedVal, _ := nodes[rand.Intn(nodecnt+1)+i*5].Get(k)
			if fetchedVal != v {
				log.Fatal("actual: ", fetchedVal, " expected: ", v)
			}
			keyList[cnt] = k
			cnt++
		}
		//delete 150 kv
		for j := 0; j < 150; j++ {
			delete(kvMap, keyList[j])
			nodes[rand.Intn(nodecnt+1)+i*5].Delete(keyList[j])
		}
		//quit 5 nodes
		for j := 0; j < 5; j++ {
			nodes[j+i*5].Quit()
			time.Sleep(3 * time.Second)
		}
		nodecnt -= 5
		time.Sleep(30 * time.Second)
		//put 300 kv
		for j := 0; j < 300; j++ {
			k := RandStringRunes(30)
			v := RandStringRunes(30)
			kvMap[k] = v
			nodes[rand.Intn(nodecnt+1)+i*5+5].Put(k, v)
		}
		//get 200 kv and check correctness

		cnt = 0
		for k, v := range kvMap {
			if cnt == 200 {
				break
			}
			fetchedVal, _ := nodes[rand.Intn(nodecnt+1)+i*5+5].Get(k)
			if fetchedVal != v {
				log.Fatal("actual: ", fetchedVal, " expected: ", v)
			}
			keyList[cnt] = k
			cnt++
		}
		//delete 150 kv
		for j := 0; j < 150; j++ {
			delete(kvMap, keyList[j])
			nodes[rand.Intn(nodecnt+1)+i*5+5].Delete(keyList[j])
		}
	}
}
