package main

import (
	"fmt"
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

	err := handShake(client)
	if err != nil {
		return
	}

	req, err := parseRequest(client)
	if err != nil {
		return
	}
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
)

// 握手
func handShake(client net.Conn) error {
	buff := make([]byte, 256)

	// read VER, NMETHODS
	if _, err := io.ReadFull(client, buff[:2]); err != nil {
		return err
	}
	nmethods := buff[1]

	// read METHODS
	if _, err := io.ReadFull(client, buff[:nmethods]); err != nil {
		return err
	}

	// write VER METHOD
	if _, err := client.Write([]byte{5, 0}); err != nil {
		return err
	}
	return nil
}

type HttpRequest struct {
	addr string
}

// 解析请求
func parseRequest(client net.Conn) (*HttpRequest, error) {
	buff := make([]byte, 256)

	// read VER CMD RSV
	if _, err := io.ReadFull(client, buff[:3]); err != nil {
		return nil, err
	}

	cmd := buff[1]
	if cmd != CmdConnect {
		return nil, Error(fmt.Sprint("command not support", cmd))
	}

	// read ATYP
	if _, err := io.ReadFull(client, buff[:1]); err != nil {
		return nil, err
	}
	addrType := buff[0]

	var host string
	var port int

	// read host
	if addrType == AtypDomainName {
		if _, err := io.ReadFull(client, buff[:1]); err != nil {
			return nil, err
		}
		domainLen := buff[0]
		if _, err := io.ReadFull(client, buff[:domainLen]); err != nil {
			return nil, err
		}
		host = string(buff[:domainLen])
	} else if addrType == AtypIPv4 {
		if _, err := io.ReadFull(client, buff[:net.IPv4len]); err != nil {
			return nil, err
		}
		host = net.IP(buff[:net.IPv4len]).String()
	} else if addrType == AtypIPv6 {
		if _, err := io.ReadFull(client, buff[:net.IPv6len]); err != nil {
			return nil, err
		}
		host = net.IP(buff[:net.IPv6len]).String()
	}

	// read port
	if _, err := io.ReadFull(client, buff[:2]); err != nil {
		return nil, err
	}
	port = (int(buff[0]) << 8) | int(buff[1])

	addr := host + ":" + strconv.Itoa(port)
	request := &HttpRequest{
		addr: addr,
	}
	return request, nil
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

type Error string

func (e Error) Error() string {
	return string(e)
}
