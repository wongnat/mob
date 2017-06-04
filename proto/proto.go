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
    Res string
}

type SeedToPeersPacket struct {
    SongFile string
}

type SeedToPeersReply struct {
  // EMPTY
}

type ListenForSeedersPacket struct {
  // EMPTY
}

type ListenForSeedersReply struct {
  // EMPTY
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

// TODO: 3-way handshake protocol to set up streaming dependencies
// 1) clients broadcast udp packet to all other clients
// 2) if client doesn't have a seeder already, they will ACK received udp packet
// 3) seeder client will ACK seedee's original ACK to notify that it is being
//    seeded to. The seedee will set a boolean when they know they are being seeded to.
