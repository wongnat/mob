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

var counter int64

type peerInfo struct {
    Conn net.Conn
    Songs []string
}

// IP -> array of songs
var peerMap map[string]peerInfo

var songQueue []string

var playing bool

func main() {
    peerMap   = make(map[string]peerInfo)
    songQueue = make([]string, 0)
    counter = 0
    playing = false

    ln, err := net.Listen("tcp", ":" + os.Args[1])
    if err != nil {
    	// handle error
    }

    fmt.Println("mob tracker listening on port " + os.Args[1] + " ...")

    go startPlaying()

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
        fmt.Println("Error in listening for client commands")
    }

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
            if counter < 5 {
                log.Println(err)
                counter++
            }
        }

        switch cmd.Cmd {
        case "leave":
            // remove songQueue song provided from host that is leaving
            for i := 1; i < len(songQueue); i++ {
                for _, song := range peerMap[clientAddr].Songs {
                    if songQueue[i] == strings.ToLower(song) {
                        songQueue = append(songQueue[:i], songQueue[i+1:]...)
                    }
                }
            }

            delete(peerMap, clientAddr)
            updateInformation()
            cmdEnc.Encode(proto.TrackerResPacket{"Received goodbye from " + conn.LocalAddr().String()})
            cmdConn.Close()
            conn.Close()
            return
        case "list":
            cmdEnc.Encode(proto.TrackerSongsPacket{getSongList()})
        case "play":
            found := false
            for _, song := range getSongList() {
                if cmd.Arg == strings.ToLower(song) {
                    songQueue = append(songQueue, cmd.Arg)
                    cmdEnc.Encode(proto.TrackerResPacket{cmd.Arg + " was successfully queued."})
                    found = true
                    break
                }
            }

            if !found {
                cmdEnc.Encode(proto.TrackerResPacket{cmd.Arg + " is not a valid song."})
            }
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

func startPlaying() {
    for {
        if len(songQueue) > 0 && !playing {
            song := songQueue[0]
            var clientAddr string
            songQueue = append(songQueue[:0], songQueue[1:]...)

            Loop:
                for k, v := range peerMap {
                    for currSong := range v.Songs {
                        if strings.ToLower(currSong) == song {
                            ip = k
                            break Loop
                        }
                    }
                }

            songCon, err := net.Dial("tcp", clientAddr + ":6124")
            if err != nil {
                fmt.Println("Error in iniating song cycle")
            }

            songEnc := gob.NewEncoder(songCon) // Will write to network.
            songDec := gob.NewDecoder(songCon) // Will read from network.

            var res proto.TrackerResPacket
            cmdEnc.Encode(proto.TrackerResPacket{"start"})

            cmdDec.Decode(&res)
            playing = false
        }
    }
}
