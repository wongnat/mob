package proto

import (
    "fmt"
	"net"
	"time"
	"github.com/tatsushid/go-fastping"
)

type Client_Init_Packet struct {
    Songs string
}

type Mp3_Frame_Packet struct {
    Seqnum uint64
    Mp3_frame []byte
}

// More packet types:


// Protocol functions:

// get RTT in terms of milliseconds between current node and specified IP
func GetRTTBetweenNodes(address string) int64 {
    pinger := fastping.NewPinger()

	_, err := pinger.Network("udp")
	// We shouldn't ever get an error but we're checking anyway
	if err != nil {
		panic("Error setting network type: " + err.Error())
	}

	addr, err := net.ResolveIPAddr("ip", address)
	if err != nil {
		panic("Error resolving IP Address: " + err.Error())
	}

	pinger.AddIPAddr(addr)
	pinger.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
		fmt.Printf("%s time=%v seconds\n", addr, rtt.Seconds())
	}

	if err = pinger.Run(); err != nil {
		panic(err)
	}

    return 0
}
