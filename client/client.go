package main

import (
    "os"
    "os/signal"
    "syscall"
    "bufio"
    "fmt"
    "log"
    "bytes"
    "encoding/binary"
    "strings"
    "path/filepath"
    "net"
    "mob/proto"
    "mob/client/music"
    //"github.com/tcolgate/mp3"
    "github.com/cenkalti/rpc2"
)

var peers []string

var conn net.Conn
var client *rpc2.Client

var ip net.IP

var udpHandshaker *net.UDPConn
var udpMp3Framer  *net.UDPConn

// var connected bool

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


    ifaces, _ := net.Interfaces()
    // handle err
    for _, i := range ifaces {
        addrs, _ := i.Addrs()
        // handle err
        for _, addr := range addrs {

            switch v := addr.(type) {
            case *net.IPNet:
                    ip = v.IP
            case *net.IPAddr:
                    ip = v.IP
            }
            // process IP address
        }
    }

    //fmt.Println(ip.String())
    //os.Exit(0)

    handshakeAddr := net.UDPAddr{IP: ip, Port: 6121,}
    mp3FrameAddr  := net.UDPAddr{IP: ip, Port: 6122,}

    udpHandshaker, _ = net.ListenUDP("udp", &handshakeAddr)
    udpMp3Framer, _  = net.ListenUDP("udp", &mp3FrameAddr)


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
    go handlePing()

    //addr := strings.Split(conn.LocalAddr().String(), ":")[0]
    _, port, _ := net.SplitHostPort(conn.LocalAddr().String())
    client.Call("join", proto.ClientInfoMsg{net.JoinHostPort(ip.String(), port), getSongNames()}, nil)
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

    //
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

func handlePing() {
    _, port, _ := net.SplitHostPort(conn.LocalAddr().String())
    for {
        var res proto.TrackerRes
        client.Call("ping", proto.ClientInfoMsg{net.JoinHostPort(ip.String(), port), nil}, &res)
        if res.Res != "" {
            go seedToPeers(res.Res)
        }
    }
}

func listenForSeeders() {

}

// udp handshake receive on port 6121
// udp song receive on port 6122
func seedToPeers(songFile string) {
    // Handle handshakes
    req := proto.HandshakePacket{ip.String()}

    buf := &bytes.Buffer{}
    err := binary.Write(buf, binary.BigEndian, &req)
    if err != nil {
        log.Println(err)
    }

    var peerToConn map[string]net.Conn

    // loop to broadcast
    for _, peer := range peers {
        addr, _, _ := net.SplitHostPort(peer)
        udpConn, _ := net.Dial("udp", net.JoinHostPort(addr, "6121"))
        peerToConn[peer] = udpConn
        udpConn.Write(buf.Bytes())
    }

    var seedees []string
    seedeeCount := 0
    for peer, _ := range peerToConn {
        go func () {
            recvBuf := make([]byte, 2048)
            var tmpBuf *bytes.Reader
            ack := proto.HandshakePacket{}
            peerToConn[peer].Read(recvBuf)
            tmpBuf = bytes.NewReader(recvBuf)
            binary.Read(tmpBuf, binary.BigEndian, &ack)
            if (seedeeCount < 2) {
                seedees = append(seedees, peer)
                seedeeCount++
            }
        }()
    }

    req = proto.HandshakePacket{ip.String()}
    buf = &bytes.Buffer{}
    err = binary.Write(buf, binary.BigEndian, &req)
    for _, seedee := range seedees {
        peerToConn[seedee].Write(buf.Bytes())
    }


    // Handle mp3 frames

    //for {

    //}


    // check if we have the song locally
        // if so, then open the file
        // else wait for songBuf to be populated

    // begin sending UDP packets with mp3 frames to peers

    // delay

    // start playing the song



    // Open the mp3 file
    /*
    r, err := os.Open("../songs/" + songFile)
    if err != nil {
        fmt.Println(err)
        return
    }

    d := mp3.NewDecoder(r)
    music.PlayFromMp3Dec(d, &songBuf)*/
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
