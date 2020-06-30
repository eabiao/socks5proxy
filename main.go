package main

import (
	"io"
	"log"
	"net"
	"strconv"
	"strings"
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
func handleConnect(conn net.Conn) {
	defer conn.Close()
	conn.(*net.TCPConn).SetKeepAlive(true)

	addr, err := Handshake(conn)
	if err != nil {
		return
	}

	addrStr := addr.String()
	log.Println(addrStr)

	target, err := net.DialTimeout("tcp", addrStr, 2*time.Second)
	if err != nil {
		return
	}
	defer target.Close()
	target.(*net.TCPConn).SetKeepAlive(true)

	relay(conn, target)

	//req, err := parseRequest(conn)
	//if err != nil {
	//	return
	//}
	//log.Println(req.host)
	//
	//if req.isHttps {
	//	fmt.Fprint(conn, "HTTP/1.1 200 Connection Established\r\n\r\n")
	//}
	//
	//target, err := net.DialTimeout("tcp", req.addr, 2*time.Second)
	//if err != nil {
	//	return
	//}
	//defer target.Close()
	//
	//if !req.isHttps {
	//	_, err = target.Write(req.data)
	//	if err != nil {
	//		return
	//	}
	//}
	//
	//relay(req.conn, target)
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

// http请求
type HttpRequest struct {
	conn    net.Conn
	addr    string
	isHttps bool
	data    []byte
	host    string
	port    int
}

// 解析请求
func parseRequest(client net.Conn) (*HttpRequest, error) {

	var buff = make([]byte, 1024)

	readN, err := client.Read(buff[:])
	if err != nil {
		return nil, err
	}
	data := buff[:readN]

	var isHttps bool
	var addr string

	for _, line := range strings.Split(string(data), "\r\n") {
		if strings.HasPrefix(line, "CONNECT") {
			isHttps = true
			continue
		}
		if strings.HasPrefix(line, "Host:") {
			addr = strings.Fields(line)[1]
			break
		}
	}

	if !strings.Contains(addr, ":") {
		if isHttps {
			addr = addr + ":443"
		} else {
			addr = addr + ":80"
		}
	}

	addrParts := strings.SplitN(addr, ":", 2)
	host := addrParts[0]
	port, _ := strconv.Atoi(addrParts[1])

	request := &HttpRequest{
		conn:    client,
		addr:    addr,
		isHttps: isHttps,
		data:    data,
		host:    host,
		port:    port,
	}
	return request, nil
}
