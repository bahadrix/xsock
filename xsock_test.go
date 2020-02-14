package xsock

import (
	"fmt"
	"log"
	"net"
	"testing"
	"time"
)

var sockAddr = &net.UnixAddr{Name: "xsock.sock", Net: "unix"}

func pusher(name string, msgCount int) {

	cli, err := net.DialUnix("unix", nil, sockAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	for i := 0; i <= msgCount; i++ {
		cli.Write([]byte(fmt.Sprintf(" [%d] ", i+1)))
		cli.Write([]byte(""))
		cli.Write([]byte(fmt.Sprintf("I am the %s of the ", name)))
		cli.Write([]byte("hell fire tom"))
		cli.Write([]byte("\x03\x03")) // Here is our delimiter at the end of the sequence
	}

}

func TestServer(t *testing.T) {

	socketServer := CreateXSockServer(&Config{
		ByteBufferSize:   1024,
		ETXCode:          0x03,
		AutoRemoveSocket: true,
	})

	segmentChannel, _, err := socketServer.Listen(sockAddr.Name, 10)

	if err != nil {
		log.Fatal(err)
	}

	// Start receiver
	go func() {
		for {
			segment := <-segmentChannel
			fmt.Printf("Segment: %s\n", segment)
		}
	}()

	// Send 100 message for each pusher
	msgCount := 100

	// Start pushers
	for _, name := range []string{"GATH", "KO.PATH", "SPITZ", "TACKLE.OW", "LAST.OFFENDER", "FHMEYLO", "TOTORO", "LIBIDIK"} {
		go pusher(name, msgCount)
	}

	// Wait 5 seconds to completion
	time.Sleep(time.Second * 5)

}

func TestPullServer(t *testing.T) {
	socketServer := CreateXSockServer(&Config{
		ByteBufferSize:   1024,
		ETXCode:          0x03,
		AutoRemoveSocket: true,
	})

	segmentChannel, _, err := socketServer.Listen(sockAddr.Name, 10)

	if err != nil {
		log.Fatal(err)
	}

	for {
		log.Printf("%s", <-segmentChannel)
	}
}
