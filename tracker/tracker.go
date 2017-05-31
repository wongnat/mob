package main

import (
    "os"
    "fmt"
    "log"
    "net"
    "strings"
    "io/ioutil"
    "mob/proto"
    "encoding/gob"
    "github.com/tcolgate/mp3"
    "time"
)

// IP -> array of songs
var peer_map map[string][]string

// slice of IP
var peers []string

//var song_queue
func main() {
    peer_map = make(map[string][]string)
    peers = make([]string, 0)

    ln, err := net.Listen("tcp", ":" + os.Args[1])
    if err != nil {
    	// handle error
    }

    fmt.Println("mob tracker listening on port " + os.Args[1] + " ...")

    for {
    	conn, err := ln.Accept()
    	if err != nil {
    		// handle error
    	}

    	go handleConnection(conn)
    }
}

func handleConnection(conn net.Conn) {
    fmt.Println("Accepted new client!")

    var init_packet proto.Client_Init_Packet

    enc := gob.NewEncoder(conn) // Will write to network.
    dec := gob.NewDecoder(conn) // Will read from network.

    dec.Decode(&init_packet)

    // Add client to our map and slice
    peers = append(peers, conn.RemoteAddr().String())
    peer_map[conn.RemoteAddr().String()] = strings.Split(init_packet.Songs, ";")

    go updateInformation(conn)

    // TODO: Listen for client commands


    // TODO: Remove this part later
    // Stream an mp3 to client
    skipped := 0
    var counter uint64
    counter = 0
    r, err := os.Open("../songs/The-entertainer-piano.mp3")
    if err != nil {
        fmt.Println(err)
        return
    }

    d := mp3.NewDecoder(r)
    var f mp3.Frame
    for {

        if err := d.Decode(&f, &skipped); err != nil {
            fmt.Println(err)
            break
        }

        byte_reader := f.Reader()
        frame_bytes, e := ioutil.ReadAll(byte_reader)
        if e != nil {
            log.Println(err)
        }

        var frame_packet proto.Mp3_Frame_Packet
        frame_packet.Seqnum = counter
        frame_packet.Mp3_frame = frame_bytes

        err := enc.Encode(frame_packet)
        if err != nil {
            //fmt.Println(err)
        }

        counter += 1
        //fmt.Println(counter)
    }
}

// tracker regularly sends host info to all nodes in order to inform them of the current network
func updateInformation(conn net.Conn) {
    enc := gob.NewEncoder(conn) // Will write to network.
    
    var err error

    for err == nil {
        err = enc.Encode(proto.Node_Info{peers})
        time.Sleep(1 * time.Second)
    }
}
