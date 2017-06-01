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
    //"github.com/tcolgate/mp3"
)

var peers []string

var conn net.Conn
var cmdConn net.Conn

// Assume mp3 is no larger than 50MB
// We reuse this buffer for each song we play
// Don't need to worry when it gets GCed since we're using it the whole time
var songBuf [20 * 1024 * 1024]byte

func main() {
    music.Init() // initialize SDL audio
    defer music.Quit()

    fmt.Print(`
              ___.
  _____   ____\_ |__
 /     \ /  _ \| __ \
|  Y Y  (  <_> ) \_\ \
|__|_|  /\____/|___  /
      \/           \/
`)

    fmt.Println("internet radio version 0.0.0")
    reader := bufio.NewReader(os.Stdin)

    loop:
    for {
        fmt.Print(">>> ")
        input, _ := reader.ReadString('\n')
        input = strings.TrimSuffix(input, "\n")
        input = strings.TrimSpace(input)
        input = strings.ToLower(input)

        strs := strings.Split(input, " ")

        switch strs[0] {
        case "join": // join 192.168.1.12:1234
            go handleJoin(strs[1])
        case "leave": // leave the network
            handleLeave()
        case "list": // list
            handleList()
        case "play": // play blah.mp3
            handlePlay(strs[1])
        case "quit": // quit the program
            if cmdConn != nil {
                handleLeave()
            }
            break loop
        case "help": // help
            handleHelp()
        default:     // error message continue
            fmt.Println("Error: not a valid command")
            handleHelp()
        }
    }
}


func handleJoin(input string) {
    ln, err := net.Listen("tcp", ":6123")
    if err != nil {
    	log.Println(err)
    }

    conn, err = net.Dial("tcp", input)
    if err != nil {
        log.Println(err)
    }

    cmdConn, err = ln.Accept()
    if err != nil {
        log.Println(err)
    }

    enc := gob.NewEncoder(conn)
    enc.Encode(proto.ClientInitPacket{getSongNames()})

    //receiveNearestNodes()
}

func handleLeave() {
    if cmdConn == nil {
        fmt.Println("Error: not connected to a tracker")
    }

    enc := gob.NewEncoder(cmdConn)
    //dec := gob.NewDecoder(cmdConn)

    //var res proto.TrackerResPacket
    enc.Encode(proto.ClientCmdPacket{"leave", ""})
    //dec.Decode(&res)

    //fmt.Println(res.Res)

    conn.Close()
    cmdConn.Close()
}

func handleList() {
    if cmdConn == nil {
        fmt.Println("Error: not connected to a tracker")
    }

    enc := gob.NewEncoder(cmdConn)
    dec := gob.NewDecoder(cmdConn)

    var res proto.TrackerSongsPacket
    enc.Encode(proto.ClientCmdPacket{"list", ""})
    dec.Decode(&res)

    fmt.Println(res.Res)
}

func handlePlay(input string) {
    if cmdConn == nil {
        fmt.Println("Error: not connected to a tracker")
    }

    enc := gob.NewEncoder(cmdConn)
    dec := gob.NewDecoder(cmdConn)

    var res proto.TrackerResPacket
    enc.Encode(proto.ClientCmdPacket{"play", input})
    dec.Decode(&res)

    fmt.Println(res.Res)
}

func handleHelp() {
    fmt.Print(`commands:
    join - connect to a tracker
    exit - disconnect from a tracker
    list - list all available songs
    play - enqueue a song to be played
    help - show commands
`)
}

// udp handshake receive on port 6121
// udp song receive on port 6122
func seedToPeers(ipAddrs []string) {
    // check if we have the song locally
        // if so, then open the file
        // else wait for songBuf to be populated

    // begin sending UDP packets with mp3 frames to peers

    // delay

    // start playing the song



    // Open the mp3 file
    /*
    r, err := os.Open("../songs/The-entertainer-piano.mp3")
    if err != nil {
        fmt.Println(err)
        return
    }

    d := mp3.NewDecoder(r)
    music.PlayFromSongBuf(d, &songBuf)*/
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

// Receive IP addresses of peers
/*
func receiveNearestNodes() {
    var info proto.ClientInfoPacket
    var err error

    dec := gob.NewDecoder(conn)
    for {
        err = dec.Decode(&info) // hope this blocks
        if err != nil {
            log.Println(err)
        }
        peers = info.ClientIps
    }
}*/
