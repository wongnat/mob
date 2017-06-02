package main

import (
    "os"
    "os/signal"
    "syscall"
    "bufio"
    "fmt"
    "log"
    //"bytes"
    "strings"
    "path/filepath"
    "net"
    "mob/proto"
    "mob/client/music"
    "github.com/tcolgate/mp3"
    "github.com/cenkalti/rpc2"
)

var peers []string

var conn net.Conn
var client *rpc2.Client

// Assume mp3 is no larger than 50MB
// We reuse this buffer for each song we play
// Don't need to worry when it gets GCed since we're using it the whole time
var songBuf [20 * 1024 * 1024]byte


// TODO: when all clients in peerMap make rpc to say that they are done with the song
// notify the next set of seeders to begin seeding
func main() {
    c := make(chan os.Signal, 2)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-c
        if client != nil {
            handleLeave()
        }
        os.Exit(1)
    }()

    music.Init() // initialize SDL audio
    defer music.Quit()

    fmt.Print(
`
              ___.
  _____   ____\_ |__
 /     \ /  _ \| __ \
|  Y Y  (  <_> ) \_\ \
|__|_|  /\____/|___  /
      \/           \/
`)

    fmt.Println()
    fmt.Println("internet radio version 0.0.0")
    reader := bufio.NewReader(os.Stdin)

    for {
        fmt.Print(">>> ")
        input, _ := reader.ReadString('\n')
        input = strings.TrimSuffix(input, "\n")
        input = strings.TrimSpace(input)
        input = strings.ToLower(input)

        strs := strings.Split(input, " ")

        switch strs[0] {
        case "join": // join 192.168.1.12:1234
            handleJoin(strs[1])
        case "leave": // leave the network
            handleLeave()
        case "list": // list
            handleList()
        case "play": // play blah.mp3
            handlePlay(strs[1])
        case "quit": // quit the program
            if client != nil {
                handleLeave()
            }
            return
        case "help": // help
            handleHelp()
        default:     // error message continue
            fmt.Println("Error: not a valid command")
            handleHelp()
        }
    }
}

func handleJoin(input string) {
    if client != nil {
        handleLeave()
    }

    var res proto.TrackerSlice
    var err error

    conn, err = net.Dial("tcp", input)
    if err != nil {
        log.Println(err)
    }

    client = rpc2.NewClient(conn)
    go client.Run()

    addr := strings.Split(conn.LocalAddr().String(), ":")[0]
    client.Call("join", proto.ClientInfoMsg{addr, getSongNames()}, nil)
    client.Call("peers", proto.ClientCmdMsg{""}, &res)
    peers = res.Res
    fmt.Println(peers)
}

func handleLeave() {
    if client == nil {
        fmt.Println("Error: not connected to a tracker")
        return
    }

    client.Call("leave", proto.ClientInfoMsg{conn.LocalAddr().String(), nil}, nil)
}

func handleList() {
    if client == nil {
        fmt.Println("Error: not connected to a tracker")
        return
    }

    var res proto.TrackerSlice

    client.Call("list", proto.ClientCmdMsg{""}, &res)
    fmt.Println(res.Res)
}

// need to listen for when to start playing to others
func handlePlay(input string) {
    if client == nil {
        fmt.Println("Error: not connected to a tracker")
        return
    }

    client.Call("play", proto.ClientCmdMsg{input}, nil)
}

func handleHelp() {
    fmt.Print(
`commands:
    join - connect to a tracker
    exit - disconnect from a tracker
    list - list all available songs
    play - enqueue a song to be played
    help - show commands
`)
}

// udp handshake receive on port 6121
// udp song receive on port 6122
func seedToPeers(songFile string) {
    fmt.Println("starting to seed to peers")
    // check if we have the song locally
        // if so, then open the file
        // else wait for songBuf to be populated

    // begin sending UDP packets with mp3 frames to peers

    // delay

    // start playing the song



    // Open the mp3 file
    r, err := os.Open("../songs/" + songFile)
    if err != nil {
        fmt.Println(err)
        return
    }

    d := mp3.NewDecoder(r)
    music.PlayFromMp3Dec(d, &songBuf)
}

// Returns csv of all song names in the songs folder.
func getSongNames() ([]string) {
    var songs []string
    filepath.Walk("../songs", func (p string, i os.FileInfo, err error) error {
        if err != nil {
            log.Println(err)
            return nil
        }

        s := filepath.Base(p)
        if strings.Compare(s, "songs") != 0 && strings.Contains(s, ".mp3") {
            songs = append(songs, s)
        }

        return nil
    })

    return songs
}
