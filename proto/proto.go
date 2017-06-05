package proto

import (
    "github.com/sparrc/go-ping"
    "time"
)

type ClientInfoMsg struct {
    Ip string
    List []string
}

type ClientCmdMsg struct {
    Arg string
}

type TrackerRes struct {
    Res string
}

type TrackerSlice struct {
    Res []string
}

type Mp3FramePacket struct {
    Seqnum uint64
    Mp3Frame []byte
}

type ClientInfoPacket struct {
    ClientIps []string
}

type HandshakePacket struct {
    Type string // "request", "accept", "reject", "confirm"
}

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
