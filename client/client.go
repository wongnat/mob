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
    //"net/http"
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

var readyToPlayMusic bool
var needsSeeder bool // tells us if we have access to the mp3 or not
//var alreadylistening bool
//var alreadyseeding bool

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

/*
    resp, err := http.Get("http://myexternalip.com/raw")
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Stderr.WriteString("\n")
		os.Exit(1)
	}
	defer resp.Body.Close()
    ipBuf := bytes.Buffer{}
    ipBuf.ReadFrom(resp.Body)*/

    //publicIp = ipBuf.String()

    publicIp = os.Args[1]

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

    client.Handle("listenForSeeders", func(client *rpc2.Client, args *proto.ListenForSeedersPacket, reply *proto.ListenForSeedersReply) error {
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

    // This seems wrong
    _, port, _ := net.SplitHostPort(conn.LocalAddr().String())
    client.Call("join", proto.ClientInfoMsg{net.JoinHostPort(publicIp, port), getSongNames()}, nil)
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

// Listens for udp request packets from seeders in order to become a seeder
func listenForSeeders() {
    // listen to incoming udp packets
    pc, err := net.ListenPacket("udp", net.JoinHostPort(publicIp, "6121"))
    if err != nil {
    	log.Fatal(err)
    }
    defer pc.Close()

    // loop continuously until we get the go ahead from the tracker to start
    // playing music.
    for !readyToPlayMusic {
        // Read a request to seed from a seeder
        buffer := make([]byte, 1024)
        n, addr, _ := pc.ReadFrom(buffer)
        s := string(byteArray[:n])

        if s == "request" { // sanity check
            if needsSeeder {
                pc.WriteTo([]byte("accept"), addr) // become a seeder
                needsSeeder = false
            } else {
                pc.WriteTo([]byte("reject"), addr) // already a seeder
            }
        }
    }
}

// udp handshake receive on port 6121
// udp song receive on port 6122
func seedToPeers(songFile string) {
    // Listen for accepts or rejects from the seeders
    pc, err := net.ListenPacket("udp", net.JoinHostPort(publicIp, "6121"))
    if err != nil {
    	log.Fatal(err)
    }
    defer pc.Close()



    //Connect udp
    conn, err := net.Dial("udp", "host:port")
    if err != nil {
    	return err
    }
    defer conn.Close()

    //simple Read
    buffer := make([]byte, 1024)
    conn.Read(buffer)

    //simple write
    conn.Write([]byte("Hello from client"))





    // get list of peers from tracker
    client.Call("peers", proto.ClientCmdMsg{""}, &res)
    peers = res.Res

    var peerToConn map[string]net.Conn  // ip address to a connection
    peerToConn := make(map[string]net.Conn)

    var peerToResp map[string]string    // ip address to response from peer
    peerToResp := make(map[string]string)

    // loop to acquire udp connections to all other peers
    for _, peer := range peers {
        ip, _, _ := net.SplitHostPort(peer)
        if ip != publicIp { // check not this client
            c, _ := net.Dial("udp", net.JoinHostPort(ip, "6121"))
            peerToConn[peer] = c
        }
    }

    // Clean up the udp connections when we're done
    defer func() {
        for _, conn := range peerToConn {
            conn.Close()
        }
    }()

    // ARQ udp request packets to peers
    maxSeedees := 1
    seedees := make([]string, 0)
    var gotResFromAllPeers := false
    go func() {

        // need a list of peers to stop sending to
        // communicate with


        for {
            for peer, conn := range peerToConn {
                conn.Write([]byte("request"))
            }

            time.Sleep(2000 * time.Millisecond) // wait a bit each burst
        }
    }()


    // exit if all peers have responsed to me
    for !gotResFromAllPeers { // read responses until we have all our seedees
        buffer := make([]byte, 1024)
        n, addr, _ := pc.ReadFrom(buffer)
        s := string(byteArray[:n])

        switch s {
        case "accept":
            peerIp, peerPort, _ := net.SplitHostPort(addr)
            seedees = append(seedees, peerIp)
        case "reject":

        }
    }



    // Handle handshakes
    fmt.Println("Sending to peers")
    req := proto.HandshakePacket{publicIp}

    buf := &bytes.Buffer{}
    err := binary.Write(buf, binary.BigEndian, &req)
    if err != nil {
        log.Println(err)
    }



    var res proto.TrackerSlice

    client.Call("peers", proto.ClientCmdMsg{""}, &res)
    peers = res.Res

    // loop to broadcast
    for _, peer := range peers {
        addr, _, _ := net.SplitHostPort(peer)
        udpConn, _ := net.Dial("udp", net.JoinHostPort(addr, "6121"))
        peerToConn[peer] = udpConn

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
            peerToConn[peer].SetReadDeadline(time.Now().Add(time.Millisecond * 500))
            _, err := peerToConn[peer].Read(recvBuf)
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
    for _, song := range getSongNames() {
        if song == songFile {
            found = true
            break
        }
    }

    if found {
        /*r, err := os.Open("../songs/" + songFile)
        if err != nil {
            fmt.Println(err)
            return
        }*/

        //d := mp3.NewDecoder(r)
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
