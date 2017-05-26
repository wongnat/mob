package main

import (
    "os"
    "fmt"
    "net"
    //"bufio"
    "strings"
    "mob/proto"
    "encoding/gob"
)

// global map of peers and their songs
var peer_map map[string][]string

// IP -> array of songs

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

    //enc := gob.NewEncoder(conn) // Will write to network.
    dec := gob.NewDecoder(conn) // Will read from network.

    dec.Decode(&init_packet)



    // Add client to our map
    peer_map[init_packet.IP_Addr] = strings.Split(init_packet.Songs, ";")

    fmt.Println(peer_map[init_packet.IP_Addr])
}
