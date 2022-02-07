package qrpc

import "io"

type Header struct {
	ServiceMethod string // 服务名和方法名("Service.Method")
	Seq           uint64 // 请求的ID
	Error         string
}

type Phone interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

type NewPhoneFunc func(io.ReadWriteCloser) Phone

type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json"
)

var NewPhoneFuncMap map[Type]NewPhoneFunc

func init() {
	NewPhoneFuncMap = make(map[Type]NewPhoneFunc)
	NewPhoneFuncMap[GobType] = NewGobPhone
}