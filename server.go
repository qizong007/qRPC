package qRPC

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"qRPC/qrpc"
	"reflect"
	"sync"
)

// 报文格式
// | Option{MagicNumber: xxx, CodecType: xxx} | Header{ServiceMethod ...} | Body interface{} |
// | <------      固定 JSON 编码      ------>  | <-------   编码方式由 CodeType 决定   ------->|
// 在一次连接中，Option 固定在报文的最开始，Header 和 Body 可以有多个，比如：
// | Option | Header1 | Body1 | Header2 | Body2 | ...

const MagicNumber = 0xbaebae

type Option struct {
	MagicNumber int
	MsgType     qrpc.Type
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	MsgType:     qrpc.GobType,
}

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = NewServer()

func (server *Server) Accept(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		go server.ServeConn(conn)
	}
}

func Accept(listener net.Listener) {
	DefaultServer.Accept(listener)
}

func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() { _ = conn.Close() }()
	// 解析option
	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", opt.MagicNumber)
		return
	}
	f := qrpc.NewPhoneFuncMap[opt.MsgType]
	if f == nil {
		log.Printf("rpc server: invalid msg type %s", opt.MsgType)
		return
	}
	// 处理消息
	server.serveMessage(f(conn))
}

// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{}{}

func (server *Server) serveMessage(msg qrpc.Phone) {
	// 处理请求是并发的，但是回复请求的报文必须是逐个发送的
	// 并发容易导致多个回复报文交织在一起，客户端无法解析
	// 所以需要确保响应完整性
	sending := new(sync.Mutex)
	// 保证所有请求都处理完
	wg := new(sync.WaitGroup)
	for {
		// 1. 读取请求
		req, err := server.parseRequest(msg)
		if err != nil {
			if req == nil {
				break // 无法解决，直接关闭连接
			}
			req.h.Error = err.Error()
			// 3. 发送响应
			server.sendResponse(msg, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		// 2. 处理请求
		go server.handleRequest(msg, req, sending, wg)
	}
	wg.Wait()
	_ = msg.Close()
}

type request struct {
	h            *qrpc.Header
	argv, replyv reflect.Value
}

// 解析请求头
func (server *Server) parseRequestHeader(msg qrpc.Phone) (*qrpc.Header, error) {
	var h qrpc.Header
	if err := msg.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

// 1. 解析请求数据
func (server *Server) parseRequest(msg qrpc.Phone) (*request, error) {
	h, err := server.parseRequestHeader(msg)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}
	// TODO: 请求参数类型未知
	req.argv = reflect.New(reflect.TypeOf(""))
	if err = msg.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv err:", err)
	}
	return req, nil
}

// 2. 逻辑处理请求
func (server *Server) handleRequest(msg qrpc.Phone, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	// TODO: 需要调用已注册的rpc方法获取 replyv
	defer wg.Done()
	log.Println("req: ", req.h, req.argv.Elem())
	req.replyv = reflect.ValueOf(fmt.Sprintf("qrpc resp %d", req.h.Seq))
	server.sendResponse(msg, req.h, req.replyv.Interface(), sending)
}

// 3. 返回响应数据
func (server *Server) sendResponse(msg qrpc.Phone, h *qrpc.Header, body interface{}, sending *sync.Mutex) {
	// 加锁目的
	// 防止一个goroutine在写buffer的时候，另一个在尝试flush
	sending.Lock()
	defer sending.Unlock()
	if err := msg.Write(h, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}
