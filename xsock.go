package xsock

import (
	"log"
	"net"
	"os"
)

type Config struct {
	// Default is 1024 byte
	ByteBufferSize uint64
	// Use this byte two times at the end of each segment two times sequentially. Default: ETX 0x03
	ETXCode uint8
	// Set true to remove socket file at start of the connection
	AutoRemoveSocket bool
}

type Server struct {
	config *Config
}

type ListenerController struct {
	listener  *net.UnixListener
	isClosing bool
}

func (c *ListenerController) Close() error {
	c.isClosing = true
	return c.listener.Close()
}

func CreateXSockServer(config *Config) *Server {
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
					// Maybe the connection is closed
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

func (s *Server) Listen(socketAddress string, channelBufferSize int) (chan []byte, *ListenerController, error) {
	if s.config.AutoRemoveSocket {
		os.Remove(socketAddress)
	}
	resultChan := make(chan []byte, channelBufferSize)
	sockAddr := &net.UnixAddr{
		Name: socketAddress,
		Net:  "unix",
	}

	l, err := net.ListenUnix("unix", sockAddr)

	if err != nil {
		return nil, nil, err
	}

	lcont := &ListenerController{listener: l}

	go func() {
		for {
			conn, err := l.AcceptUnix()

			if err != nil {
				if lcont.isClosing {
					break
				}
				log.Printf("Skipping connection request. Cause: %v", err)
				continue
			}
			go s.connectionHandler(conn, &resultChan)

		}
	}()

	return resultChan, lcont, nil
}
