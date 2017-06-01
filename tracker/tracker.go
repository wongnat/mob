package main

import (
    "os"
    "fmt"
    "log"
    "net"
    "strings"
    //"io/ioutil"
    "mob/proto"
    "encoding/gob"
    //"github.com/tcolgate/mp3"
    //"time"
)

type peerInfo struct {
    Conn net.Conn
    Songs []string
}

// IP -> array of songs
var peerMap map[string]peerInfo

var songQueue []string

func main() {
    peerMap   = make(map[string]peerInfo)
    songQueue = make([]string, 0)

    ln, err := net.Listen("tcp", ":" + os.Args[1])
    if err != nil {
    	// handle error
    }

    fmt.Println("mob tracker listening on port " + os.Args[1] + " ...")

    for {
    	conn, err := ln.Accept()
    	if err != nil {
    		log.Println(err)
    	}

    	go handleConnection(conn)
    }
}

func handleConnection(conn net.Conn) {
    fmt.Println("Accepted new client!") // TODO: add time stamp
    clientAddr := conn.RemoteAddr().String()
    clientAddr = strings.Split(clientAddr, ":")[0]

    // TODO: Listen for client commands
    cmdConn, err := net.Dial("tcp", clientAddr + ":6123")
    if err != nil {
        log.Println(err)
    }

    //conn.SetKeepAlive(false)
    //cmdConn.SetKeepAlive(false)

    cmdEnc := gob.NewEncoder(cmdConn) // Will write to network.
    cmdDec := gob.NewDecoder(cmdConn) // Will read from network.

    var initPacket proto.ClientInitPacket

    dec := gob.NewDecoder(conn) // Will read from network.
    dec.Decode(&initPacket)

    // Add client to our map and slice

    info := peerInfo{conn, strings.Split(initPacket.Songs, ";")}
    peerMap[clientAddr] = info

    updateInformation()

    for { // handle commands
        var cmd proto.ClientCmdPacket
        err := cmdDec.Decode(&cmd)
        if err != nil {
            log.Println(err)
        }

        switch cmd.Cmd {
        case "leave":
            delete(peerMap, clientAddr)
            updateInformation()
            //cmdEnc.Encode(proto.TrackerResPacket{"Received goodbye from " + conn.LocalAddr().String()})
            cmdConn.Close()
            conn.Close()
            return
        case "list":
            cmdEnc.Encode(proto.TrackerSongsPacket{getSongList()})
        case "play":
            songQueue = append(songQueue, cmd.Arg)
            cmdEnc.Encode(proto.TrackerResPacket{cmd.Arg + " was successfully queued."})
        }
    }
}

func updateInformation() {
    keys := make([]string, 0, len(peerMap))
    for k := range peerMap {
        keys = append(keys, k)
    }

    for i := 0; i < len(keys); i++ {
        enc := gob.NewEncoder(peerMap[keys[i]].Conn)
        enc.Encode(proto.ClientInfoPacket{keys})
    }
}

func getSongList() ([]string) {
    var songs []string

    keys := make([]string, 0, len(peerMap))
    for k := range peerMap {
        keys = append(keys, k)
    }

    for i := 0; i < len(keys); i++ {
        songs = append(songs, peerMap[keys[i]].Songs...)
    }

    return songs
}
