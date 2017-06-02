package main

import (
    "os"
    "fmt"
    "log"
    "net"
    "strings"
    //"io/ioutil"
    "mob/proto"
    //"encoding/gob"
    //"github.com/tcolgate/mp3"
    "time"
    "github.com/cenkalti/rpc2"
)

// IP -> array of songs
var peerMap map[string][]string
// TODO: var liveMap map[string]time.Time
var songQueue []string

// var playing bool

// TODO: when all clients in peerMap make rpc to say that they are done with the song
// notify the next set of seeders to begin seeding
func main() {
    peerMap   = make(map[string][]string)
    songQueue = make([]string, 0)
    //playing = false

    srv := rpc2.NewServer()

    srv.Handle("join", func(client *rpc2.Client, args *proto.ClientInfoMsg, reply *proto.TrackerRes) error {
        peerMap[args.Ip] = args.List
        //fmt.Println("Handling join ...")
        return nil
    })

    srv.Handle("list", func(client *rpc2.Client, args *proto.ClientCmdMsg, reply *proto.TrackerSlice) error {
        reply.Res = getSongList()
        fmt.Println(getSongList())
        //fmt.Println("Handling list ...")
        return nil
    })

    srv.Handle("play", func(client *rpc2.Client, args *proto.ClientCmdMsg, reply *proto.TrackerRes) error {
        for _, song := range getSongList() {
            if args.Arg == strings.ToLower(song) {
                songQueue = append(songQueue, args.Arg)
                break
            }
        }

        // TODO: if no song is playing currently, reply to clients with the song to start seeding

        return nil
    })

    srv.Handle("leave", func(client *rpc2.Client, args *proto.ClientInfoMsg, reply *proto.TrackerRes) error {
        delete(peerMap, args.Ip)
        return nil
    })

    srv.Handle("peers", func(client *rpc2.Client, args *proto.ClientCmdMsg, reply *proto.TrackerSlice) error {
        //fmt.Println("Handling peers ...")
        keys := make([]string, 0, len(peerMap))
        for k := range peerMap {
            keys = append(keys, k)
        }
        reply.Res = keys
        return nil
    })

    srv.Handle("ping", func(client *rpc2.Client, args *proto.ClientInfoMsg, reply *proto.TrackerRes) error {
        //fmt.Println("Handling peers ...")
        keys := make([]string, 0, len(peerMap))
        for k := range peerMap {
            keys = append(keys, k)
        }
        reply.Res = keys
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
