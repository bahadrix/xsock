package main

import (
	"github.com/bahadrix/xsock"
	"log"
	"net"
	"time"
)

type RouteConfig struct {
	ReceiverSocketAddress    string
	TransmitterSocketAddress string
	RoutePackBufferSize      int
	ReceiverPackBufferSize   int
	ReceiverServerConfig     *xsock.Config
}

type Route struct {
	Rx         *xsock.Server
	Config     *RouteConfig
	PackBuffer chan []byte
	isDropping bool
	dropCount  uint
}

func CreateRoute(routeConfig *RouteConfig) *Route {

	return &Route{
		Rx:         xsock.CreateXSockServer(routeConfig.ReceiverServerConfig),
		Config:     routeConfig,
		PackBuffer: make(chan []byte, routeConfig.RoutePackBufferSize),
	}

}

func (r *Route) statsCycle() {
	for {
		if !r.isDropping {
			if len(r.PackBuffer) > 10 {
				log.Printf("Pack buffer size is %d / %d", len(r.PackBuffer), cap(r.PackBuffer))
			}
		}
		time.Sleep(5 * time.Second)
	}
}

func (r *Route) Start() error {

	rxChan, _, err := r.Rx.Listen(r.Config.ReceiverSocketAddress, r.Config.ReceiverPackBufferSize)

	if err != nil {
		return err
	}

	go r.txCycle()
	log.Println("TX cycle started")

	go r.statsCycle()
	log.Println("Stats cycle started")

	log.Println("RX cycle starting")
	r.rxCycle(&rxChan)

	return nil
}

func (r *Route) rxCycle(rxChan *chan []byte) {
	for {
		pack := <-*rxChan
		select {
		case r.PackBuffer <- pack:
			if r.isDropping {
				r.isDropping = false
				log.Printf("State recovered. Packages started to send. %d packages dropped.", r.dropCount)
				r.dropCount = 0
			}

		default:
			r.dropCount++
			if !r.isDropping {
				r.isDropping = true
				log.Printf("Route buffer is full. Packages started to drop.")
			}
		}
	}
}

func (r *Route) txCycle() {

	conn := r.obtainTxConnection()

	defer conn.Close()

	for {
		pack := <-r.PackBuffer
		pack = append(pack, r.Config.ReceiverServerConfig.ETXCode, r.Config.ReceiverServerConfig.ETXCode)
		for {
			_, err := conn.Write(pack)
			if err == nil {
				break
			}
			// Got write error retry connection
			conn = r.obtainTxConnection()
		}

	}

}

func (r *Route) obtainTxConnection() (conn *net.UnixConn) {
	txAddr := &net.UnixAddr{Name: r.Config.TransmitterSocketAddress, Net: "unix"}
	var err error
	clean := true
	for {
		conn, err = net.DialUnix("unix", nil, txAddr)
		if err == nil {
			log.Printf("Connection succeeded to: %s", r.Config.TransmitterSocketAddress)
			break
		}
		if clean {
			log.Printf("TX is down (%s) Buffer: %d/%d", err.Error(), len(r.PackBuffer), cap(r.PackBuffer))
			log.Printf("I'm retrying to connect continuously for each second. Sock: %s", r.Config.TransmitterSocketAddress)
			clean = false
		}

		time.Sleep(time.Second)
	}
	return conn
}
