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

	addr, err := handShake(client)
	if err != nil {
		return
	}

	log.Println(addr)

	target, err := net.DialTimeout("tcp", addr, 2*time.Second)
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
//func handShake(client net.Conn) (string, error) {

//type HttpRequest struct {
//	addr string
//	data []byte
//}

// 解析请求
//func parseRequest(client net.Conn) (*HttpRequest, error) {

// 解析请求
func handShake(client net.Conn) (string, error) {

	buff := make([]byte, 1+1+255+2)

	// handshake
	if _, err := io.ReadFull(client, buff[:2]); err != nil {
		return "", err
	}
	nmethods := buff[1]

	if _, err := io.ReadFull(client, buff[:nmethods]); err != nil {
		return "", err
	}

	if _, err := client.Write([]byte{5, 0}); err != nil {
		return "", err
	}

	// -----------------------
	// read addr

	// VER CMD RSV ATYP
	if _, err := io.ReadFull(client, buff[:4]); err != nil {
		return "", err
	}

	cmd := buff[1]
	if cmd != CmdConnect {
		return "", Error("command not support")
	}

	addrType := buff[3]
	var addrData []byte

	if addrType == AtypDomainName {
		if _, err := io.ReadFull(client, buff[1:2]); err != nil {
			return "", err
		}
		if _, err := io.ReadFull(client, buff[2:2+int(buff[1])+2]); err != nil {
			return "", err
		}
		addrData = buff[:1+1+int(buff[1])+2]
	} else if addrType == AtypIPv4 {
		if _, err := io.ReadFull(client, buff[1:1+net.IPv4len+2]); err != nil {
			return "", err
		}
		addrData = buff[:1+net.IPv4len+2]
	} else if addrType == AtypIPv6 {
		if _, err := io.ReadFull(client, buff[1:1+net.IPv6len+2]); err != nil {
			return "", err
		}
		addrData = buff[:1+net.IPv6len+2]
	}

	client.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})

	var host, port string

	if addrType == AtypDomainName {
		host = string(addrData[2 : 2+int(addrData[1])])
		port = strconv.Itoa((int(addrData[2+int(addrData[1])]) << 8) | int(addrData[2+int(addrData[1])+1]))
	} else if addrType == AtypIPv4 {
		host = net.IP(addrData[1 : 1+net.IPv4len]).String()
		port = strconv.Itoa((int(addrData[1+net.IPv4len]) << 8) | int(addrData[1+net.IPv4len+1]))
	} else if addrType == AtypIPv6 {
		host = net.IP(addrData[1 : 1+net.IPv6len]).String()
		port = strconv.Itoa((int(addrData[1+net.IPv6len]) << 8) | int(addrData[1+net.IPv6len+1]))
	}

	addr := host + ":" + port
	return addr, nil
}

type Error string

func (e Error) Error() string {
	return string(e)
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
