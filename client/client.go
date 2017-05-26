package main

import (
    "os"
    "fmt"
    "log"
    "bytes"
    "strings"
    "path/filepath"
    //"path"
    "encoding/gob"
    "net"
    "mob/proto"
    //"os/signal"
    //"syscall"
    //"mob/client/music"
)

func main() {
    // Setup music
    // music.Init()
    // defer music.Quit()
    
    fmt.Println("Starting client ...")
    conn, err := net.Dial("tcp", os.Args[1])
    if err != nil {
        // handle error
    }

    enc := gob.NewEncoder(conn) // Will write to network.
    //dec := gob.NewDecoder(conn) // Will read from network.

    // Encode (send) some values.
    enc.Encode(proto.Client_Init_Packet{"blah", getSongNames()})

    //fmt.Fprintf(conn, getSongNames() + "\n")
    //conn.Write(init_packet)

    // Play song
    // music.Play("../songs/east-asian.mp3")
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
        if strings.Compare(s, "songs") != 0 {
            buffer.WriteString(s + ";")
        }

        return nil
    })

    return buffer.String()
}
