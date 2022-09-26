package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"qRPC/qrpc"
	"time"
)

func startServer(addr chan string) {
	// 选择一个空闲的端口
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr <- l.Addr().String()
	qrpc.Accept(l)
}

func main() {
	addr := make(chan string)
	go startServer(addr)

	// 确保 server 端口监听成功，client 再发起请求
	conn, _ := net.Dial("tcp", <-addr)
	defer func() { _ = conn.Close() }()

	time.Sleep(time.Second)

	// client 与 server 握手
	// | Option | Header1 | Body1 | Header2 | Body2 | ...
	_ = json.NewEncoder(conn).Encode(qrpc.DefaultOption)
	msg := qrpc.NewGobPhone(conn)

	for i := 0; i < 5; i++ {
		h := &qrpc.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}
		_ = msg.Write(h, fmt.Sprintf("qrpc req %d", h.Seq))
		_ = msg.ReadHeader(h)
		var resp string
		_ = msg.ReadBody(&resp)
		log.Println("resp:", resp)
	}
}
