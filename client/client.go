package main

import (
    "os"
    "os/signal"
    "syscall"
    "bufio"
    "fmt"
    "log"
    //"bytes"
    //"encoding/binary"
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

// UDP handler for handshake packets
var packetConn net.PacketConn

// This client's LAN IP address
// TODO: this is a cmd line arg; want to discover this automatically
var publicIp string

// Control flags
var connectedToTracker bool // did join a tracker
var isSeeder bool           // tells us if we have access to the mp3 or not
var alreadySeeding bool     // prevent tracker rpc from being over called

// Seeder's data structures
var peerToConn map[string]bool
var seedees []string
var currentSong string

var maxSeedees int

// Assume mp3 is no larger than 50MB. We reuse this buffer for each song we play.
// Don't need to worry when it gets GCed since we're using it the whole time
var songBuf [20 * 1024 * 1024]byte


// TODO: clients make rpc to say that they are done with the song
func main() {
    // Handle kill signal gracefully
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

    // Set max number of seedees to our stream to prevent congestion on peers
    // TODO: want to reset all these variables when done playing the song
    maxSeedees = 1
    seedees = make([]string, 0)
    peerToConn = make(map[string]bool)
    connectedToTracker = false
    isSeeder = false
    alreadySeeding = false
    currentSong = ""

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
            if connectedToTracker {
                handleLeave()
            }
            return
        case "help": // help
            handleHelp()
        default:     // error message continue
            fmt.Println("Error: not a valid command")
            //handleHelp()
        }
    }
}

func handleJoin(input string) {
    if connectedToTracker {
        handleLeave()
    }

    var err error
    conn, err = net.Dial("tcp", input)
    if err != nil {
        log.Println(err)
    }

    client = rpc2.NewClient(conn)

    // Register the rpc handlers for seedToPeers() so that tracker can notify
    // client when to start seeding
    client.Handle("seed", func(client *rpc2.Client, args *proto.TrackerRes, reply *proto.HandshakePacket) error {
        if (alreadySeeding) {
          return nil
        }

        // Start seeding
        alreadySeeding = true
        isSeeder = true
        currentSong = args.Res
        // TODO: set alreadyseeding to false once a done rpc is issued to the tracker
        go seedToPeers(currentSong)
        return nil
    })

    client.Handle("start-playing", func(client *rpc2.Client, args *proto.ClientInfoMsg, reply *proto.TrackerRes) error {
        // TODO: rpc for client to say song is done playing
        // currSong = ""
        return nil
    })

    go client.Run()

    connectedToTracker = true
    fmt.Println("Begin listen for peers")
    go listenForPeers()
    go handlePing()

    _, port, _ := net.SplitHostPort(conn.LocalAddr().String())
    client.Call("join", proto.ClientInfoMsg{net.JoinHostPort(publicIp, port), getSongNames()}, nil)
}

func handleLeave() {
    if !connectedToTracker {
        fmt.Println("Error: not connected to a tracker")
        return
    }

    connectedToTracker = false
    client.Call("leave", proto.ClientInfoMsg{conn.LocalAddr().String(), nil}, nil)
    fmt.Println("Leaving the tracker in 5 sec ...")
    time.Sleep(5000 * time.Millisecond)
    packetConn.Close()
    fmt.Println("done")
    //conn.Close() // TODO: is this right?
}

func handleListSongs() {
    if !connectedToTracker {
        fmt.Println("Error: not connected to a tracker")
        return
    }

    var res proto.TrackerSlice
    client.Call("list-songs", proto.ClientCmdMsg{""}, &res)
    fmt.Println(res.Res)
}

func handleListPeers() {
    if !connectedToTracker {
        fmt.Println("Error: not connected to a tracker")
        return
    }

    var res proto.TrackerSlice
    client.Call("list-peers", proto.ClientCmdMsg{""}, &res)
    fmt.Println(res.Res)
}

// need to listen for when to start playing to others
func handlePlay(input string) {
    if !connectedToTracker {
        fmt.Println("Error: not connected to a tracker")
        return
    }

    client.Call("play", proto.ClientCmdMsg{input}, nil)
}

func handleHelp() {
    fmt.Print(
`commands:
    join  - connect to a tracker
    leave - disconnect from a tracker
    list-songs - list all available songs
    list-peers - list all peers on the network
    play - enqueue a song to be played
    help - show commands
    quit - exit the program
`)
}

func handlePing() {
    _, port, _ := net.SplitHostPort(conn.LocalAddr().String())
    for {
        client.Call("ping", proto.ClientInfoMsg{net.JoinHostPort(publicIp, port), nil}, nil)
        time.Sleep(500 * time.Millisecond)
    }
}

// TODO: when song is done playing, set songBuf = songBuf[:0] to clear it
func listenForMp3Frames(publicIp string) {
    //mp3FrameAddr  := net.UDPAddr{IP: net.ParseIP(publicIp), Port: 6122,}
    //udpMp3Framer, _  = net.ListenUDP("udp", &mp3FrameAddr)

}

// Listens for udp request packets from peers in order to build the stream graph
// Handles all possible handshake packets that will be sent to this peer.
// Called in handleJoin when you join a tracker
func listenForPeers() {
    // listen to incoming udp packets
    fmt.Println("listening for peers!")
    var err error
    packetConn, err = net.ListenPacket("udp", net.JoinHostPort(publicIp, "6121"))
    if err != nil {
        log.Fatal(err)
    }
    //defer pc.Close()

    // Continously listen for handshake packets
    // Eventually after a successful round of handshaking, all peers will
    // be seeders and will block on the next ReadFrom() call until the next
    // round
    for connectedToTracker { // terminate when we leave a tracker
        // Read a packet
        buffer := make([]byte, 2048)
        n, addr, e := packetConn.ReadFrom(buffer) // block here
        if e != nil {
            log.Fatal("error when reading packet")
            return
        }

        s := string(buffer[:n])
        substrs := strings.Split(s, ":")
        ip, _, _ := net.SplitHostPort(addr.String())
        //dest := net.JoinHostPort(ip, "6121")
        //fmt.Println(ip)
        //fmt.Println(port)
        //fmt.Println(s)
        raddr := net.UDPAddr{IP: net.ParseIP(ip), Port: 6121}

        // Process the packet and handle
        switch substrs[0] {
        case "request": // where this client is a non-seeder
            if currentSong == "" {
                currentSong = substrs[1]
            }

            if isSeeder || hasSongLocally(substrs[1]) {
                go func() {
                    for i := 0; i < 3; i++ { // redundancy
                        packetConn.WriteTo([]byte("reject"), &raddr)
                        time.Sleep(500 * time.Microsecond)
                    }
                }()
            } else {
                go func() {
                    for i := 0; i < 3; i++ { // redundancy
                        packetConn.WriteTo([]byte("accept"), &raddr)
                        time.Sleep(500 * time.Microsecond)
                    }
                }()
            }
        case "confirm": // where this client is a non-seeder
            if isSeeder {
                go func() {
                    for i := 0; i < 3; i++ { // redundancy
                        packetConn.WriteTo([]byte("reject"), &raddr)
                        time.Sleep(500 * time.Microsecond)
                    }
                }()
            } else {
                // Set this to false after the song finishes playing
                isSeeder = true
                go seedToPeers(currentSong)
            }
        case "accept": // where this client is a seeder
            if isSeeder && len(seedees) < maxSeedees {
                ip, _, _ := net.SplitHostPort(addr.String())
                seedees = append(seedees, ip)
                go func() {
                    for i := 0; i < 3; i++ { // redundancy
                        packetConn.WriteTo([]byte("confirm"), &raddr)
                        time.Sleep(500 * time.Microsecond)
                    }
                }()
                peerToConn[ip] = true
            } else if isSeeder && len(seedees) >= maxSeedees {
                peerToConn[ip] = true
            } else {
                // is a non-seeder; shouldn't get here; sanity check
                log.Fatal("non-seeder tried to accept other non-seeder")
            }
        case "reject": // where this client is a seeder
            peerToConn[ip] = true
            // remove this peer from our seedee list if they are there
        }
    }
}

// If this client has access to mp3 stream, find peers to stream to.
// Broadcasts packets to peers until every peer has responded.
// Called by tracker rpc.
func seedToPeers(songFile string) {
    var wg sync.WaitGroup

    // Get list of peers from tracker
    var peers proto.TrackerSlice
    client.Call("list-peers", proto.ClientCmdMsg{""}, &peers)

    peerToConn = make(map[string]bool)

    // Loop to acquire udp connections to all other peers
    for _, peer := range peers.Res {
        ip, _, _ := net.SplitHostPort(peer)

        if ip != publicIp { // check not this client
            // Connect to an available peer
            pc, _ := net.Dial("udp", net.JoinHostPort(ip, "6121"))
            fmt.Println("contacting " + net.JoinHostPort(ip, "6121") + " ...")
            peerToConn[ip] = false

            wg.Add(1)
            // ARQ requests to the peer until we set its response bool to nil
            go func() {
                defer wg.Done()
                defer pc.Close()
                for !peerToConn[ip] {
                    pc.Write([]byte("request:" + songFile))
                    time.Sleep(500 * time.Microsecond)
                }
            }()
        }
    }

    wg.Wait()
    fmt.Println(seedees)

    fmt.Println("This client is ready to play!")
    // Done our job in stream graph; make rpc to tracker?
    // Or have ping have us move forward
    // TODO: make rpc to say ready to play.
}

func handleStartPlaying() {
    // tracker will call this client rpc on ping when all peers are ready to
    // play

}

func handleDonePlaying() {
    // reset globals

    // make rpc call to tracker
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

func hasSongLocally(songFile string) bool {
    for _, song := range getSongNames() {
        if song == songFile {
            return true
        }
    }

    return false
}
