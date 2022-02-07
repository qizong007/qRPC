package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"qRPC"
	"qRPC/qrpc"
	"time"
)

func startServer(addr chan string) {
	// pick a free port
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr <- l.Addr().String()
	// server 先开电话
	qRPC.Accept(l)
}

func main() {
	addr := make(chan string)
	go startServer(addr)

	// 确保 server 端口监听成功，client 再发起请求
	conn, _ := net.Dial("tcp", <-addr)
	defer func() { _ = conn.Close() }()

	time.Sleep(time.Second)

	// client 与 server 握手
	_ = json.NewEncoder(conn).Encode(qRPC.DefaultOption)
	msg := qrpc.NewGobPhone(conn)

	for i := 0; i < 5; i++ {
		h := &qrpc.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}
		_ = msg.Write(h, fmt.Sprintf("qrpc req %d", h.Seq))
		_ = msg.ReadHeader(h)
		var reply string
		_ = msg.ReadBody(&reply)
		log.Println("reply:", reply)
	}
}
