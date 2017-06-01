package main

import (
    "os"
    "bufio"
    "fmt"
    "log"
    "bytes"
    "strings"
    "path/filepath"
    "encoding/gob"
    "net"
    "mob/proto"
    "mob/client/music"
    "github.com/tcolgate/mp3"
)

var peers []string

// Assume mp3 is no larger than 50MB
// We reuse this buffer for each song we play
// Don't need to worry when it gets GCed since we're using it the whole time
var songBuf [50 * 1024 * 1024]byte

func main() {
    music.Init() // initialize SDL audio
    defer music.Quit()

    fmt.Println("mob client ...")

    // Open the mp3 file
    /*
    r, err := os.Open("../songs/The-entertainer-piano.mp3")
    if err != nil {
        fmt.Println(err)
        return
    }

    d := mp3.NewDecoder(r)
    music.PlayFromSongBuf(d, &songBuf)*/

    reader := bufio.NewReader(os.Stdin)

    for {

        fmt.Println("Commands: join, list, play, exit")
        input, _ := reader.ReadString('\n')

        input = strings.Replace(input, "\n", "", -1)

        if strings.Compare("join", input) == 0 {
            handleJoin()
        } else if strings.Compare("list", input) == 0 {
            handleList()
        } else if strings.Compare("play", input) == 0 {
            handlePlay()
        } else {
            handleExit()
            break
        }
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


func handleJoin() {
    conn, err := net.Dial("tcp", os.Args[1])
    if err != nil {
        // handle error
    }

    // Interface to tracker node
    enc := gob.NewEncoder(conn) // Will write to tracker

    // Send list of songs to tracker
    enc.Encode(proto.Client_Init_Packet{getSongNames()})

    go receiveNearestNodes(conn)
}

func handleList() {

}

func handlePlay() {

}

func handleExit() {

}

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

func receiveNearestNodes(conn net.Conn) {
    dec := gob.NewDecoder(conn) // Will read from tracker

    var info *proto.Node_Info = new(proto.Node_Info)

    var err error

    for err == nil {
        err = dec.Decode(info)

        peers = info.Nodes
    }
}
