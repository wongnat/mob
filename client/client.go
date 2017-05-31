package main

import (
    "os"
    "fmt"
    "log"
    "bytes"
    "strings"
    "path/filepath"
    //"encoding/gob"
    //"net"
    //"mob/proto"
    "mob/client/music"
    "github.com/tcolgate/mp3"
)

// Assume mp3 is no larger than 50MB
// We reuse this buffer for each song we play
// Don't need to worry when it gets GCed since we're using it the whole time
var songBuf [50 * 1024 * 1024]byte

func main() {
    music.Init() // initialize SDL audio
    defer music.Quit()

    fmt.Println("mob client ...")

    // Open the mp3 file
    r, err := os.Open("../songs/The-entertainer-piano.mp3")
    if err != nil {
        fmt.Println(err)
        return
    }

    d := mp3.NewDecoder(r)
    music.PlayFromSongBuf(d, &songBuf)
    /*
    conn, err := net.Dial("tcp", os.Args[1])
    if err != nil {
        // handle error
    }

    // Interface to tracker node
    enc := gob.NewEncoder(conn) // Will write to tracker
    dec := gob.NewDecoder(conn) // Will read from tracker

    // Send list of songs to tracker
    enc.Encode(proto.Client_Init_Packet{getSongNames()})

    for {
        var frame_packet *proto.Mp3_Frame_Packet = new(proto.Mp3_Frame_Packet)

        dec.Decode(frame_packet)


        fmt.Println(frame_packet.Mp3_frame)
        // music.PlayBuffer(&frame_packet.Mp3_frame)
    }*/


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

func seedToPeers(ip_addrs []string) {
    // check if we have the song locally
        // if so, then open the file
        // else wait for songBuf to be populated
        
    // begin sending UDP packets with mp3 frames to peers

    // delay

    // start playing the song

}

// Returns csv of all song names in the songs folder.
func getSongNames() string {
    var buffer bytes.Buffer
    filepath.Walk("../songs", func (p string, i os.FileInfo, err error) error {
        if err != nil {
            log.Println(err)
            return nil
        }

        s := filepath.Base(p)
        if strings.Compare(s, "songs") != 0 && strings.Contains(s, ".mp3") {
            buffer.WriteString(s + ";")
        }

        return nil
    })

    return buffer.String()
}
