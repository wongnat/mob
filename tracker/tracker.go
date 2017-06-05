package main

import (
    "os"
    "fmt"
    "log"
    "net"
    //"strings"
    //"io/ioutil"
    "mob/proto"
    //"encoding/gob"
    //"github.com/tcolgate/mp3"
    //"time"
    "sync/atomic"
    "github.com/cenkalti/rpc2"
)

var peerMap map[string][]string
var songQueue []string

var currSong string
var clientsPlaying uint64

// TODO: when all clients in peerMap make rpc to say that they are done with the song
// notify the next set of seeders to begin seeding
func main() {
    peerMap   = make(map[string][]string)
    songQueue = make([]string, 0)
    currSong = ""
    clientsPlaying = 0
    //currentlyplaying = false

    srv := rpc2.NewServer()

    // join the peer network
    srv.Handle("join", func(client *rpc2.Client, args *proto.ClientInfoMsg, reply *proto.TrackerRes) error {
        peerMap[args.Ip] = args.List
        return nil
    })

    // Return list of songs available to be played
    srv.Handle("list-songs", func(client *rpc2.Client, args *proto.ClientCmdMsg, reply *proto.TrackerSlice) error {
        reply.Res = getSongList()
        return nil
    })

    // Return list of peers connected to tracker
    srv.Handle("list-peers", func(client *rpc2.Client, args *proto.ClientCmdMsg, reply *proto.TrackerSlice) error {
        keys := make([]string, 0, len(peerMap))
        for k := range peerMap {
            keys = append(keys, k)
        }
        reply.Res = keys
        return nil
    })

    // Enqueue song into song queue
    srv.Handle("play", func(client *rpc2.Client, args *proto.ClientCmdMsg, reply *proto.TrackerRes) error {
        for _, song := range getSongList() {
            if args.Arg == song {
                songQueue = append(songQueue, args.Arg)
                break
            }
        }

        return nil
    })

    srv.Handle("leave", func(client *rpc2.Client, args *proto.ClientInfoMsg, reply *proto.TrackerRes) error {
        delete(peerMap, args.Ip)
        return nil
    })

    // Contact peers with the song locally to start seeding
    // Clients ask tracker when they can start seeding and when they can start
    // playing the buffered mp3 frames
    // TODO: Synchronization by including a time delay to "start-playing" rpc
    srv.Handle("ping", func(client *rpc2.Client, args *proto.ClientInfoMsg, reply *proto.TrackerRes) error {
        // we are still playing a song
        // TODO: this won't allow clients to join mid-stream
        if clientsPlaying != 0 {
            return nil
        }

        // not playing a song; set currSong if not already set
        if currSong == "" && len(songQueue) > 0 {
            currSong = songQueue[0]
            songQueue = append(songQueue[:0], songQueue[1:]...)
        }

        // contact source seeders to start seeding
        for _, song := range peerMap[args.Ip] {
            if song == currSong {
                client.Call("seed", proto.TrackerRes{currSong}, nil)
                break
            }
        }

        return nil
    })

    // Notify the tracker that the client ready to start playing the song
    srv.Handle("ready-to-play", func(client *rpc2.Client, args *proto.ClientInfoMsg, reply *proto.TrackerRes) error {
        client.Call("start-playing", proto.TrackerRes{""}, nil)
        atomic.AddUint64(&clientsPlaying, 1)
        return nil
    })

    // Notify the tracker that the client is done playing the audio for the mp3
    srv.Handle("done-playing", func(client *rpc2.Client, args *proto.ClientInfoMsg, reply *proto.TrackerRes) error {
        atomic.StoreUint64(&clientsPlaying, clientsPlaying - 1)

        if (clientsPlaying == 0) { // on the last done-playing, we reset the currSong
            currSong = ""
        }

        return nil
    })

    ln, err := net.Listen("tcp", ":" + os.Args[1])
    if err != nil {
        log.Println(err)
    }

    fmt.Println("mob tracker listening on port: " + os.Args[1] + " ...")

    for {
        srv.Accept(ln)
    }
}

// TODO: maybe return unique song list
func getSongList() ([]string) {
    var songs []string

    keys := make([]string, 0, len(peerMap))
    for k := range peerMap {
        keys = append(keys, k)
    }

    for i := 0; i < len(keys); i++ {
        songs = append(songs, peerMap[keys[i]]...)
    }

    return songs
}
