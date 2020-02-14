package main

import (
	"flag"
	"github.com/bahadrix/xsock"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	receiverConfig := &xsock.Config{
		ByteBufferSize:   0,
		ETXCode:          0,
		AutoRemoveSocket: true,
	}

	config := &RouteConfig{
		ReceiverSocketAddress:    "",
		TransmitterSocketAddress: "",
		RoutePackBufferSize:      0,
		ReceiverPackBufferSize:   10,
		ReceiverServerConfig:     receiverConfig,
	}

	var etx int

	flag.StringVar(&config.ReceiverSocketAddress, "rx", "", "Receiver socket address")
	flag.StringVar(&config.TransmitterSocketAddress, "tx", "", "Transmitter socket address")
	flag.IntVar(&config.RoutePackBufferSize, "pack-buffer-size", 10000, "Route buffer size. Actual buffer used while absence of transmitters.")
	flag.Uint64Var(&receiverConfig.ByteBufferSize, "byte-buffer-size", 1024, "Receiver read buffer size in bytes")
	flag.IntVar(&etx, "etx", 3, "Etx code")

	flag.Parse()
	receiverConfig.ETXCode = uint8(etx)

	route := CreateRoute(config)

	go func() {
		err := route.Start()

		if err != nil {
			log.Fatal(err)
		}
	}()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)

	<-sc

	log.Println("Shutting down")
}
