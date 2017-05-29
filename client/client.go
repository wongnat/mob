package main

import (
    "os"
    "fmt"
    "log"
    "bytes"
    "strings"
    "path/filepath"
    //"path"
    "encoding/gob"
    "net"
    "mob/proto"
    "github.com/mikioh/tcp"
    "github.com/mikioh/tcpinfo"
    "time"
    //"os/signal"
    //"syscall"
    //"mob/client/music"
    //"github.com/tcolgate/mp3"
)

func main() {
    fmt.Println("Starting client ...")
    conn, err := net.Dial("tcp", os.Args[1])
    if err != nil {
        // handle error
    }

    enc := gob.NewEncoder(conn) // Will write to network.
    dec := gob.NewDecoder(conn) // Will read from network.

    // Encode (send) some values.
    enc.Encode(proto.Client_Init_Packet{getSongNames()})

    for {
        var frame_packet proto.Mp3_Frame_Packet

        dec.Decode(&frame_packet)

        fmt.Println(frame_packet.Seqnum)
        //fmt.Println(frame_packet.Mp3_frame.Size())
    }


    // Shell commands:
    // list
    // play
    // exit
    // join

    //for {
        // get stdin input
        // switch on string
    //    break
    //}
}

/*
func handleJoin() {

}

func handleList() {

}

func handlePlay() {

}

func handleExit() {

}*/

// Returns csv of all song names in the songs folder.
func getSongNames() string {
    var buffer bytes.Buffer
    filepath.Walk("../songs", func (p string, i os.FileInfo, err error) error {
        if err != nil {
            log.Println(err)
            return nil
        }

        s := filepath.Base(p)
        if strings.Compare(s, "songs") != 0 {
            buffer.WriteString(s + ";")
        }

        return nil
    })

    return buffer.String()
}

// get RTT in terms of milliseconds between current node and specified IP
func getRTTBetweenNodes(address string) float32 {
    c, err := net.Dial("tcp", address)

    if err != nil {
	       // error handling
    }
    defer c.Close()

    tc, err := tcp.NewConn(c)
    if err != nil {
    	// error handling
    }
    var o tcpinfo.Info
    var b [256]byte
    i, err := tc.Option(o.Level(), o.Name(), b[:])
    if err != nil {
    	// error handling
    }
    txt, err := json.Marshal(i)
    if err != nil {
    	// error handling
    }

    return (time.Seconds(txt.rtt) / 1000)
}
