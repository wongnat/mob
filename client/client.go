package main

import (
    "os"
    "fmt"
    "log"
    "bytes"
    "strings"
    "path/filepath"
    "path"
    "net"
    //"os/signal"
    //"syscall"
    "mob/client/music"
)

//type Packet struct {

//}

func main() {
    music.Init()
    defer music.Quit()

    /*
    c := make(chan os.Signal, 2)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-c
        music.Quit()
        os.Exit(1)
    }()*/

    conn, err := net.Dial("tcp", os.Args[1])
    if err != nil {
        // handle error
    }

    fmt.Fprintf(conn, getSongNames() + "\n")

    // Play song
    music.Play("../songs/east-asian.mp3")
}

func getSongNames() string {
    var buffer bytes.Buffer
    filepath.Walk("../songs", func (p string, i os.FileInfo, err error) error {
        if err != nil {
            log.Println(err)
            return nil
        }

        s := path.Base(p)
        if strings.Compare(s, "songs") != 0 {
            buffer.WriteString(s + ";")
        }

        return nil
    })

    return buffer.String()
}
