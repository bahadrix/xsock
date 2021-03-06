package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
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
		RxSocketFileMode:         0700,
	}

	var etx int
	var fmod uint

	flag.StringVar(&config.ReceiverSocketAddress, "rx", "", "Receiver socket address")
	flag.StringVar(&config.TransmitterSocketAddress, "tx", "", "Transmitter socket address")
	flag.IntVar(&config.RoutePackBufferSize, "pack-buffer-size", 10000, "Route buffer size. Actual buffer used while absence of transmitters.")
	flag.Uint64Var(&receiverConfig.ByteBufferSize, "byte-buffer-size", 1024, "Receiver read buffer size in bytes")
	flag.UintVar(&fmod, "rxmode", 0700, "Rx socket's file mode. Use zero prefixed version like 0777")
	flag.IntVar(&etx, "etx", 3, "Etx code")

	flag.Parse()

	receiverConfig.ETXCode = uint8(etx)
	config.RxSocketFileMode = os.FileMode(fmod)
	route := CreateRoute(config)

	// Print banner
	fmt.Println(XPROXY_LOGO)
	fmt.Println("Diff dash: ALPHA")
	fmt.Printf("BUILD: %s\n", BUILD_HASH)

	configJson, _ := json.Marshal(config)
	var configJsonPretty bytes.Buffer
	_ = json.Indent(&configJsonPretty, configJson, "", "    ")
	fmt.Printf("Config:\n%s\n\n", configJsonPretty.Bytes())

	// Start server
	err := route.Start()

	if err != nil {
		log.Fatal(err)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	<-sc

	log.Println("Shutting down")
}
