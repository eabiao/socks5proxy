package main

import (
	"io"
	"log"
	"net"
	"strconv"
	"time"
)

func main() {
	listenAddr := "127.0.0.1:1080"

	ssk, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("listen:", listenAddr)

	for {
		sk, err := ssk.Accept()
		if err != nil {
			log.Println(err)
		}

		go handleConnect(sk)
	}
}

// 处理请求
func handleConnect(client net.Conn) {
	defer client.Close()
	client.(*net.TCPConn).SetKeepAlive(true)

	// 错误处理
	defer catch()

	// 握手
	handShake(client)

	// 解析请求
	req := parseRequest(client)
	log.Println(req.addr)

	client.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})

	target, err := net.DialTimeout("tcp", req.addr, 2*time.Second)
	if err != nil {
		return
	}
	defer target.Close()
	target.(*net.TCPConn).SetKeepAlive(true)

	relay(client, target)
}

const (
	CmdConnect = 1

	AtypIPv4       = 1
	AtypDomainName = 3
	AtypIPv6       = 4

	ErrCommandNotSupport = Error("command not support")
)

// 握手
func handShake(client net.Conn) {
	conn := newBuffConn(client, 256)

	// read VER, NMETHODS
	buff := conn.readBytes(2)
	nmethods := buff[1]

	// read METHODS
	conn.readBytes(int(nmethods))

	// write VER METHOD
	conn.writeBytes([]byte{5, 0})
}

type HttpRequest struct {
	addr string
}

// 解析请求
func parseRequest(client net.Conn) *HttpRequest {
	conn := newBuffConn(client, 256)

	// read VER CMD RSV
	buff := conn.readBytes(3)

	cmd := buff[1]
	if cmd != CmdConnect {
		panic(ErrCommandNotSupport)
	}

	// read ATYP
	buff = conn.readBytes(1)
	addrType := buff[0]

	var host string
	var port int

	// read host
	if addrType == AtypDomainName {
		buff = conn.readBytes(1)
		domainLen := buff[0]
		buff = conn.readBytes(int(domainLen))
		host = string(buff[:domainLen])
	} else if addrType == AtypIPv4 {
		buff = conn.readBytes(net.IPv4len)
		host = net.IP(buff[:net.IPv4len]).String()
	} else if addrType == AtypIPv6 {
		buff = conn.readBytes(net.IPv6len)
		host = net.IP(buff[:net.IPv6len]).String()
	}

	// read port
	buff = conn.readBytes(2)
	port = (int(buff[0]) << 8) | int(buff[1])

	addr := host + ":" + strconv.Itoa(port)
	return &HttpRequest{addr: addr}
}

// 数据传输
func relay(left, right net.Conn) (int64, int64) {
	ch := make(chan int64)

	go func() {
		reqN, _ := io.Copy(right, left)
		right.SetDeadline(time.Now())
		left.SetDeadline(time.Now())
		ch <- reqN
	}()

	respN, _ := io.Copy(left, right)
	right.SetDeadline(time.Now())
	left.SetDeadline(time.Now())
	reqN := <-ch

	return reqN, respN
}

// 带缓存的连接
type BuffConn struct {
	conn net.Conn
	buff []byte
}

func newBuffConn(conn net.Conn, buffSize int) *BuffConn {
	return &BuffConn{
		conn: conn,
		buff: make([]byte, buffSize),
	}
}

func (bc *BuffConn) readBytes(size int) []byte {
	_, err := io.ReadFull(bc.conn, bc.buff[:size])
	try(err)
	return bc.buff[:size]
}

func (bc *BuffConn) writeBytes(data []byte) {
	_, err := bc.conn.Write(data)
	try(err)
}

// error
type Error string

func (e Error) Error() string {
	return string(e)
}

// try
func try(err error) {
	if err != nil {
		panic(err)
	}
}

// catch
func catch() {
	if err := recover(); err != nil {
		log.Println(err)
	}
}
