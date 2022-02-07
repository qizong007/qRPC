package qrpc

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

type GobPhone struct {
	conn io.ReadWriteCloser // 连接实例
	buf  *bufio.Writer      // 为了防止阻塞而创建的带缓冲的Writer（提升性能）
	dec  *gob.Decoder
	enc  *gob.Encoder
}

// 相当于显示 implement，表明了预期接口，防止由方法签名更改引起的破坏
// 提高可读性，且有助于代码的导航/生成
var _ Phone = (*GobPhone)(nil)

func NewGobPhone(conn io.ReadWriteCloser) Phone {
	buf := bufio.NewWriter(conn)
	return &GobPhone{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf),
	}
}

func (c *GobPhone) ReadHeader(h *Header) error {
	return c.dec.Decode(h)
}

func (c *GobPhone) ReadBody(body interface{}) error {
	return c.dec.Decode(body)
}

func (c *GobPhone) Write(h *Header, body interface{}) (err error) {
	defer func() {
		_ = c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()
	if err = c.enc.Encode(h); err != nil {
		log.Println("rpc message: gob error encoding header:", err)
		return err
	}
	if err = c.enc.Encode(body); err != nil {
		log.Println("rpc message: gob error encoding body:", err)
		return err
	}
	return nil
}

func (c *GobPhone) Close() error {
	return c.conn.Close()
}
