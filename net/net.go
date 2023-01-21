package net

import (
	"fmt"
	"log"
	"syscall"
)

const BACKLOG int = 64

func Accept(fd int) (int, error) {
	nfd, _, err := syscall.Accept(fd)
	//ignore Scokaddr(client addr)
	return nfd, err
}

func Connect(host [4]byte, port int) (int, error) {
	s, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		log.Printf("init socket err: %v\n", err)
		return -1, err
	}
	var addr syscall.SockaddrInet4
	addr.Addr = host
	addr.Port = port
	err = syscall.Connect(s, &addr)
	if err != nil {
		log.Printf("connect err: %v\n", err)
		return -1, err
	}
	return s, err
}

func Read(fd int, buf []byte) (int, error) {
	return syscall.Read(fd, buf)
}

func Write(fd int, buf []byte) (int, error) {
	return syscall.Write(fd, buf)
}

func TcpServer(port int) (int, error) {
	s, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		log.Printf("init socket err: %v\n", err)
		return -1, err
	}
	syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, port)
	if err != nil {
		log.Printf("set SO_REUSEPORT err: %v\n", err)
		syscall.Close(s)
		return -1, nil
	}
	var addr syscall.SockaddrInet4
	addr.Port = port
	err = syscall.Bind(s, &addr)
	if err != nil {
		log.Printf("bind addr err: %v\n", err)
		syscall.Close(s)
		return -1, nil
	}
	err = syscall.Listen(s, BACKLOG)
	fmt.Println("")
	if err != nil {
		log.Printf("listen socket err: %v\n", err)
		syscall.Close(s)
		return -1, nil
	}
	return s, nil
}

func Close(fd int) {
	syscall.Close(fd)
}
