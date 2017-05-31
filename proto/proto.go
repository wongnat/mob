package proto

import (
    "github.com/sparrc/go-ping"
    "time"
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
    pinger, err := ping.NewPinger(address[0:9])
    if err != nil {
        panic(err)
    }

    pinger.Count = 1
    pinger.Run()
    stats := pinger.Statistics()
    return int64(stats.MinRtt / time.Millisecond)
}
