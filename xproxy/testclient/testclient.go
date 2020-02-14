package main

import (
	"flag"
	"fmt"
	"github.com/bahadrix/xsock"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var mode string
var socketAddr string

func main() {

	var publisherInterval int

	flag.StringVar(&mode, "mode", "", "publish|subscribe")
	flag.StringVar(&socketAddr, "sock", "", "socket address")
	flag.IntVar(&publisherInterval, "interval", 1000000, "Publisher interval in microsecs")
	flag.Parse()

	switch mode {
	case "publish":
		publish(publisherInterval)
	case "subscribe":
		subscribe()
	default:
		println("Input error. Mode muste be publish or subscribe")

	}

}

func publish(interval int) {
	rxAddr := &net.UnixAddr{Name: socketAddr, Net: "unix"}
	pubConn, err := net.DialUnix("unix", nil, rxAddr)

	if err != nil {
		log.Fatal(err)
	}

	defer pubConn.Close()

	go func() {
		i := 0
		for {
			i++
			_, err := pubConn.Write([]byte(fmt.Sprintf("[%05d] %s\x03\x03", i, "Cıvışıç işiçğ")))
			if err != nil {
				log.Printf("Error on write: %s", err.Error())
			}
			time.Sleep(time.Duration(interval) * time.Microsecond)
		}
	}()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)

	<-sc
}

func subscribe() {

	rx := xsock.CreateXSockServer(&xsock.Config{
		ByteBufferSize:   1024,
		ETXCode:          0x03,
		AutoRemoveSocket: true,
	})

	packs, _, err := rx.Listen(socketAddr, 10)

	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			p := <-packs
			log.Printf("RxPack: %s", p)
		}
	}()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)

	<-sc
}
