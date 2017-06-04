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
    "time"
    "sync"
)

// TCP and RPC handlers for the tracker
var conn net.Conn
var client *rpc2.Client

// This client's LAN IP address
var publicIp string


var connectedToTracker bool
var readyToPlayMusic bool
var isSeeder bool // tells us if we have access to the mp3 or not
var alreadySeeding bool

var peerToConn map[string]net.Conn

var mu sync.Mutex

// TODO: currSong string

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

    // Initialize SDL audio
    music.Init()
    defer music.Quit()

    // Get our local network IP address
    publicIp = os.Args[1]

    // Start the shell
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

        strs := strings.Split(input, " ")
        switch strs[0] {
        case "join": // join 192.168.1.12:1234
            handleJoin(strs[1])
        case "leave": // leave the network
            handleLeave()
        case "list-songs": // list
            handleListSongs()
        case "list-peers":
            handleListPeers()
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

    conn, err := net.Dial("tcp", input)
    if err != nil {
        log.Println(err)
    }

    client = rpc2.NewClient(conn)

    // Register the rpc handlers for seedToPeers() so that tracker can notify
    // client when to start seeding
    client.Handle("seedToPeers", func(client *rpc2.Client, args *proto.HandshakePacket, reply *proto.HandshakePacket) error {
        if (alreadySeeding) {
          return nil
        }

        // Start seeding
        alreadySeeding = true
        // TODO: set alreadyseeding to false once a done rpc is issued to the tracker
        go seedToPeers(args.SongFile)
        return nil
    })

    srv.Handle("start-playing", func(client *rpc2.Client, args *proto.ClientInfoMsg, reply *proto.TrackerRes) error {
        // TODO: rpc for client to say song is done playing
        // currSong = ""
        return nil
    })

    go client.Run()

    connectedToTracker = true

    go listenForPeers()
    go handlePing()

    _, port, _ := net.SplitHostPort(conn.LocalAddr().String())
    client.Call("join", proto.ClientInfoMsg{net.JoinHostPort(publicIp, port), getSongNames()}, nil)
}

func handleLeave() {
    if client == nil {
        fmt.Println("Error: not connected to a tracker")
        return
    }

    connectedToTracker = false
    client.Call("leave", proto.ClientInfoMsg{conn.LocalAddr().String(), nil}, nil)
    conn.Close() // TODO: is this right?
}

func handleListSongs() {
    if client == nil {
        fmt.Println("Error: not connected to a tracker")
        return
    }

    var res proto.TrackerSlice
    client.Call("list-songs", proto.ClientCmdMsg{""}, &res)
    fmt.Println(res.Res)
}

func handleListPeers() {
    if client == nil {
        fmt.Println("Error: not connected to a tracker")
        return
    }

    var res proto.TrackerSlice
    client.Call("list-peers", proto.ClientCmdMsg{""}, &res)
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
        client.Call("ping", proto.ClientInfoMsg{net.JoinHostPort(publicIp, port), nil}, &res)
        time.Sleep(2000 * time.Millisecond)
    }
}

// TODO: when song is done playing, set songBuf = songBuf[:0] to clear it
func listenForMp3Frames(publicIp string) {
    mp3FrameAddr  := net.UDPAddr{IP: net.ParseIP(publicIp), Port: 6122,}
    udpMp3Framer, _  = net.ListenUDP("udp", &mp3FrameAddr)

    // TODO: add second level seeding!

    //for {
    //    n, addr, err := udpHandshaker.ReadFromUDP(songBuf)
    //     go seedToPeers(currSong)
    //}
}

// Listens for udp request packets from peers in order to build the stream graph
// Handles all possible handshake packets that will be sent to this peer.
// Called in handleJoin when you join a tracker
func listenForPeers() {
    // listen to incoming udp packets
    pc, err := net.ListenPacket("udp", net.JoinHostPort(publicIp, "6121"))
    if err != nil {
        log.Fatal(err)
    }
    defer pc.Close()

    // Continously listen for handshake packets
    // Eventually after a successful round of handshaking, all peers will
    // be seeders and will block on the next ReadFrom() call until the next
    // round
    for connectedToTracker { // terminate when we leave a tracker
        // Read a packet
        buffer := make([]byte, 1024)
        n, addr, _ := pc.ReadFrom(buffer)
        s := string(byteArray[:n])

        // Process the packet and handle
        switch s {
        case "request":
            if isSeeder {
                pc.WriteTo([]byte("reject"), addr)
            } else {
                pc.WriteTo([]byte("accept"), addr)
            }
        case "confirm":
            if isSeeder {
                pc.WriteTo([]byte("reject"), addr)
            } else {
                // Set this to false after the song finishes playing
                isSeeder = true
            }
        case "accept":
            if isSeeder {
                pc.WriteTo([]byte("confirm"), addr)
            } else {
                // this shouldn't happen
            }
        case "reject":
            // stop sending to the peer
        }
    }
}

// If this client has access to mp3 stream, find peers to stream to.
// Broadcasts packets to peers until every peer has responded.
// Called by tracker rpc.
func seedToPeers(songFile string) {
    // Get list of peers from tracker
    client.Call("peers", proto.ClientCmdMsg{""}, &res)
    peers := res.Res

    peerToConn = make(map[string]net.Conn)

    // Seedee info
    maxSeedees := 1
    seedees := make([]string, 0)

    //var peerToResp map[string]string    // ip address to response from peer
    //peerToResp := make(map[string]string)

    // Loop to acquire udp connections to all other peers
    for _, peer := range peers {
        ip, _, _ := net.SplitHostPort(peer)
        if ip != publicIp { // check not this client
            // Connect to an available peer
            c, _ := net.Dial("udp", net.JoinHostPort(ip, "6121"))
            peerToConn[peer] = c

            // ARQ requests to the peer until we set its net.Conn to nil
            go func() {
                for peerToConn[peer] != nil {
                    c.Write([]byte("request"))
                }
            }
        }
    }

    // Clean up the udp connections when we're done
    /*defer func() {
        for _, conn := range peerToConn {
            conn.Close()
        }
    }()*/
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
