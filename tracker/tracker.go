package main

import (
    "os"
    "fmt"
    "net"
    //"bufio"
    "strings"
    "mob/proto"
    "encoding/gob"
    "github.com/tcolgate/mp3"
)

// IP -> array of songs
var peer_map map[string][]string
//var song_queue
func main() {
    peer_map = make(map[string][]string)

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

    // Add client to our map
    peer_map[conn.RemoteAddr().String()] = strings.Split(init_packet.Songs, ";")

    // TODO: Listen for client commands


    // TODO: Remove this part later
    // Stream an mp3 to client



    skipped := 0
    var counter uint64
    counter = 0
    r, err := os.Open("../songs/east-asian.mp3")
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

        var frame_packet proto.Mp3_Frame_Packet
        //frame_packet.Seqnum = counter
        //frame_packet.Mp3_frame = f

        err := enc.Encode(frame_packet)
        if err != nil {
            //fmt.Println(err)
        }
        counter += 1

        //fmt.Println(counter)
        //fmt.Println(f.Size())
        fmt.Println(f.String())
    }
}
