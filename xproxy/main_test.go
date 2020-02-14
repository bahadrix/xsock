package main

import (
	"fmt"
	"github.com/bahadrix/xsock"
	"github.com/stretchr/testify/assert"
	"log"
	"net"
	"testing"
	"time"
)

const MSG = "Civışlı kıkıbıç"

func TestRoute(t *testing.T) {

	txSockAddr := "/tmp/tx.sock"
	rxSockAddr := "/tmp/proxy.sock"

	route := CreateRoute(&RouteConfig{
		ReceiverSocketAddress:    rxSockAddr,
		TransmitterSocketAddress: txSockAddr,
		RoutePackBufferSize:      10,
		ReceiverPackBufferSize:   2,
		ReceiverServerConfig: &xsock.Config{
			ByteBufferSize:   1024,
			ETXCode:          0x03,
			AutoRemoveSocket: true,
		},
	})

	node := xsock.CreateXSockServer(&xsock.Config{
		ByteBufferSize:   1024,
		ETXCode:          0x03,
		AutoRemoveSocket: true,
	})

	go func() {
		err := route.Start()
		if err != nil {
			t.Error(err)
		}
	}()
	time.Sleep(time.Second)

	// Publisher node
	sampleCount := route.Config.RoutePackBufferSize * 250
	err := sendSample(MSG, rxSockAddr, sampleCount, 0*time.Second)

	assert.Nil(t, err)
	assert.Equal(t, true, route.isDropping)
	assert.Equal(t, uint(sampleCount-route.Config.RoutePackBufferSize), route.dropCount)

	// Subscriber node
	nodeChan, _, err := node.Listen(txSockAddr, 10)
	log.Print("Subscriber1 started to listen")
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, route.Config.RoutePackBufferSize, len(route.PackBuffer), "Pack buffer must be full")

	for i := 1; i <= route.Config.RoutePackBufferSize; i++ {
		log.Printf("A %d/%d", i, route.Config.RoutePackBufferSize)
		pack := <-nodeChan
		assert.Equal(t, MSG, string(pack))
	}

}

func sendSample(msg string, sockAddr string, packCount int, interval time.Duration) error {
	rxAddr := &net.UnixAddr{Name: sockAddr, Net: "unix"}
	pubConn, err := net.DialUnix("unix", nil, rxAddr)

	if err != nil {
		return err
	}

	defer pubConn.Close()
	for i := 0; i < packCount; i++ {
		_, err := pubConn.Write([]byte(fmt.Sprintf("%s\x03\x03", msg)))
		if err != nil {
			return err
		}
		time.Sleep(interval)
	}

	return nil
}
