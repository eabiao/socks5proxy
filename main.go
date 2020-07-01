package main

import (
	"io"
	"log"
	"net"
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

	req, err := parseRequest(client)
	if err != nil {
		return
	}

	log.Println(req.addr)

	target, err := net.DialTimeout("tcp", req.addr, 2*time.Second)
	if err != nil {
		return
	}
	defer target.Close()
	target.(*net.TCPConn).SetKeepAlive(true)

	relay(client, target)
}

// http请求
type HttpRequest struct {
	isHttps bool
	addr    string
	data    []byte
}

const MaxAddrLen = 1 + 1 + 255 + 2

// 解析请求
func parseRequest(client net.Conn) (*HttpRequest, error) {

	buf := make([]byte, MaxAddrLen)

	if _, err := io.ReadFull(client, buf[:2]); err != nil {
		return nil, err
	}

	nmethods := buf[1]
	if _, err := io.ReadFull(client, buf[:nmethods]); err != nil {
		return nil, err
	}

	if _, err := client.Write([]byte{5, 0}); err != nil {
		return nil, err
	}

	if _, err := io.ReadFull(client, buf[:3]); err != nil {
		return nil, err
	}

	if _, err := io.ReadFull(client, buf[:1]); err != nil {
		return nil, err
	}

	var addrData []byte
	addrType := buf[0]
	if addrType == 3 {
		if _, err := io.ReadFull(client, buf[1:2]); err != nil {
			return nil, err
		}
		if _, err := io.ReadFull(client, buf[2:2+int(buf[1])+2]); err != nil {
			return nil, err
		}
		addrData = buf[:1+1+int(buf[1])+2]
	} else if addrType == 1 {
		if _, err := io.ReadFull(client, buf[1:1+net.IPv4len+2]); err != nil {
			return nil, err
		}
		addrData = buf[:1+net.IPv4len+2]
	} else if addrType == 4 {
		if _, err := io.ReadFull(r, b[1:1+net.IPv6len+2]); err != nil {
			return nil, err
		}
		addrData = b[:1+net.IPv6len+2]
	}

	return nil, nil
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
