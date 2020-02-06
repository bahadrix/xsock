package main

import (
	"log"
	"net"
	"os"
)

type Config struct {
	// Default is 1024 byte
	ByteBufferSize uint64
	// Use this byte two times at the end of each segment two times sequentially. Default: ETX 0x03
	ETXCode          uint8
	// Set true to remove socket file at start and end of the connection
	AutoRemoveSocket bool
}

type Server struct {
	config *Config
}

func CreateServer(config *Config) *Server {
	if config.ByteBufferSize == 0 {
		config.ByteBufferSize = 1024
	}
	if config.ETXCode == 0 {
		// ETX see: https://en.wikipedia.org/wiki/End-of-Text_character we use it double for 16 char table bit support
		config.ETXCode = 0x03
	}

	return &Server{config: config}

}

func (s *Server) connectionHandler(conn *net.UnixConn, resultChan *chan []byte) {
	var lastPackByte byte
	packEmpty := true

	pack := make([]byte, 0, s.config.ByteBufferSize)
	buff := make([]byte, s.config.ByteBufferSize)

	for {

		n, _, err := conn.ReadFromUnix(buff)
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok {
				if opErr.Err.Error() == "EOF" {
					// May be the connection is closed
					break
				}
			}
			log.Printf("Message read error: %v", err)
			break
		}

		for _, b := range buff[:n] {
			if !packEmpty && lastPackByte == s.config.ETXCode && b == s.config.ETXCode {
				*resultChan <- pack[:len(pack)-1]
				pack = make([]byte, 0, s.config.ByteBufferSize)
				packEmpty = true
			} else {
				pack = append(pack, b)
				packEmpty = false
			}
			lastPackByte = b
		}

	}

}

func (s *Server) Listen(socketAddress string, channelBufferSize int) (chan []byte, error) {
	resultChan := make(chan []byte, channelBufferSize)
	sockAddr := &net.UnixAddr{
		Name: socketAddress,
		Net:  "unix",
	}
	if s.config.AutoRemoveSocket {
		os.Remove(socketAddress)
		defer os.Remove(socketAddress)
	}

	l, err := net.ListenUnix("unix", sockAddr)

	if err != nil {
		return nil, err
	}

	go func() {
		for {
			conn, err := l.AcceptUnix()
			if err != nil {
				log.Printf("Error on accepting connection: %v", err)
			}
			go s.connectionHandler(conn, &resultChan)
		}
	}()

	return resultChan, nil
}
