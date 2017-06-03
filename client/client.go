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
    "net/http"
    "mob/proto"
    "mob/client/music"
    //"github.com/tcolgate/mp3"
    "github.com/cenkalti/rpc2"
    "time"
    "sync"
)

var peers []string

var conn net.Conn
var client *rpc2.Client

var publicIp string

var udpHandshaker *net.UDPConn
var udpMp3Framer  *net.UDPConn

var needsSeeder bool
var alreadylistening bool
var alreadyseeding bool

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

    music.Init() // initialize SDL audio
    defer music.Quit()

    resp, err := http.Get("http://myexternalip.com/raw")
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Stderr.WriteString("\n")
		os.Exit(1)
	}
	defer resp.Body.Close()
    ipBuf := bytes.Buffer{}
    ipBuf.ReadFrom(resp.Body)
    publicIp = ipBuf.String()

    needsSeeder = true

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

    //var res proto.TrackerSlice
    var err error

    conn, err = net.Dial("tcp", input)
    if err != nil {
        log.Println(err)
    }

    client = rpc2.NewClient(conn)
    // Register the rpc handlers; seedToPeers and listenForSeeders
    client.Handle("seedToPeers", func(client *rpc2.Client, args *proto.SeedToPeersPacket, reply *proto.SeedToPeersReply) error {
        // TODO: make alreadyseeding a global variable
        // Avoid duplicate sendToPeers requests
        if (alreadyseeding) {
          return nil
        }

        // Start seeding
        alreadyseeding = true
        // TODO: set alreadyseeding to false once a done rpc is issued to the tracker
        go seedToPeers(args.SongFile)
        return nil
    })

    client.Handle("listenForSeeders", func(client *rpc2.Client, args *proto.listenForSeedersPacket, reply *proto.listenForSeedersReply) erro {
      // TODO: make alreadylistening a global variable
      if (alreadylistening) {
        return nil
      }

      // Start listening
      alreadylistening = true
      // TODO: set alreadylistening to false once ListenForSeeders() wants to return
      go listenForSeeders(publicIp)
      return nil
    })

    go client.Run()
    go handlePing()
    // go listenForSeeders(publicIp)

    //addr := strings.Split(conn.LocalAddr().String(), ":")[0]
    _, port, _ := net.SplitHostPort(conn.LocalAddr().String())
    client.Call("join", proto.ClientInfoMsg{net.JoinHostPort(publicIp, port), getSongNames()}, nil)
    //client.Call("peers", proto.ClientCmdMsg{""}, &res)
    //peers = res.Res
    //fmt.Println(peers)
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
        client.Call("ping", proto.ClientInfoMsg{net.JoinHostPort(publicIp, port), nil}, &res)
        // if res.Res != "" {
        //     go seedToPeers(res.Res)
        // }
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

// TODO: we want to make listenForSeeders and seedToPeers rpc's for the tracker
// that will be called in response to a ping rpc

// TODO: maybe make this an rpc for the tracker, so that we don't have to
// continuously listen, and instead call it when the next song to be played is
// ready
func listenForSeeders(publicIp string) {
    handshakeAddr := net.UDPAddr{IP: net.ParseIP(publicIp), Port: 6121,}
    udpHandshaker, _ = net.ListenUDP("udp", &handshakeAddr)

    peerMap := make(map[string]bool)

    // Listen for handshakes
    for {
        // Read a seeder request
        buf := make([]byte, 1024)
        n, addr, err := udpHandshaker.ReadFromUDP(buf)

        var tmpBuf *bytes.Reader
        res := proto.HandshakePacket{}
        tmpBuf = bytes.NewReader(buf)
        binary.Read(tmpBuf, binary.BigEndian, &res)

        if peerMap[res.Ip] {
          needsSeeder = false
          udpHandshaker.Close()
          alreadylistening = false
          fmt.Println("Confirming a seeder!")
          break
        }

        peerMap[res.Ip] = true

        // Send an ACK
        var ack proto.HandshakePacket
        ack = proto.HandshakePacket{"accept"}
        // if needsSeeder {
        //     ack = proto.HandshakePacket{"accept"}
        // } else {
        //     ack = proto.HandshakePacket{"reject"}
        // }

        sendBuf = &bytes.Buffer{}
        err := binary.Write(sendBuf, binary.BigEndian, &ack)

        m, err := handshakeAddr.WriteToUDP(sendBuf.Bytes(), addr)
    }
}

// udp handshake receive on port 6121
// udp song receive on port 6122
func seedToPeers(songFile string) {
    // Handle handshakes
    fmt.Println("Sending to peers")
    req := proto.HandshakePacket{publicIp}

    buf := &bytes.Buffer{}
    err := binary.Write(buf, binary.BigEndian, &req)
    if err != nil {
        log.Println(err)
    }

    var peerToConn map[string]net.Conn  // ip address to a connection
    var res proto.TrackerSlice

    client.Call("peers", proto.ClientCmdMsg{""}, &res)
    peers = res.Res

    // loop to broadcast
    for _, peer := range peers {
        addr, _, _ := net.SplitHostPort(peer)
        udpConn, _ := net.Dial("udp", net.JoinHostPort(addr, "6121"))
        peerToConn[peer] = udpConn
        peerToConn[peer].SetReadDeadline(time.Millisecond * 500)
        udpConn.Write(buf.Bytes())
    }
    fmt.Println("Broadcast to all peers done")

    var seedees []string
    seedeeCount := 0
    for peer, _ := range peerToConn {
        go func () {
            recvBuf := make([]byte, 2048)
            var tmpBuf *bytes.Reader
            ack := proto.HandshakePacket{}
            _, _, err := peerToConn[peer].ReadFromUDP(recvBuf)
            if (err != nil) {
              return
            }
            tmpBuf = bytes.NewReader(recvBuf)
            binary.Read(tmpBuf, binary.BigEndian, &ack)
            mu.Lock()
            defer mu.Unlock()
            if (seedeeCount < 2) {
                fmt.Println("Received accept")
                seedees = append(seedees, peer)
                seedeeCount++
            }
        }()
    }

    req = proto.HandshakePacket{publicIp}
    buf = &bytes.Buffer{}
    err = binary.Write(buf, binary.BigEndian, &req)
    for _, seedee := range seedees {
        peerToConn[seedee].Write(buf.Bytes())
    }
    fmt.Println("Done with handshake")

    found := false
    if _, song := range getSongNames() {
        if song == songFile {
            found = true
            break
        }
    }

    if found {
        r, err := os.Open("../songs/" + songFile)
        if err != nil {
            fmt.Println(err)
            return
        }

        d := mp3.NewDecoder(r)
        // buffer to songBuf
        //music.PlayFromMp3Dec(d, &songBuf)
    }

    // stream from songBuf
    // TODO actually start streaming to peers

    // we want to make an rpc to let the tracker know we're ready to start playing,
    // then the tracker will notifies when to play
    //
    // music.Play // this will block until song is done
    // free the music
    needsSeeder = true

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
