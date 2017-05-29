package main

import (
    "os"
    "fmt"
    "bytes"
    "mob/client/music"
    "github.com/tcolgate/mp3"
    //"log"
    "io/ioutil"
    //"mob/proto"
)

// Program to make sure songs work
/*
func main() {
    music.Init()
    defer music.Quit()

    fmt.Println("Now playing a song ...")
    music.Play("../songs/" + os.Args[1] + ".mp3")
    fmt.Println("done")
}
*/

// Plays a song with in-memory byte buffer
// TODO: figure out how to make it not choppy
// IDEA: once I have the first N frames, begin playing the music, while the
// music is playing, get the next N frames and repeat i.e. pre-buffer N frames
func main() {
    music.Init()
    defer music.Quit()


    // Open the mp3 file
    r, err := os.Open("../songs/The-entertainer-piano.mp3")
    if err != nil {
        fmt.Println(err)
        return
    }

    skipped := 0


    d := mp3.NewDecoder(r)
    var f mp3.Frame
    var playable_bytes bytes.Buffer

    for {
        counter := 0
        for {
            // Get next frame
            if err := d.Decode(&f, &skipped); err != nil {
                fmt.Println(err)
                break
            }

            // Stop if counter is target value
            if counter == 100 { // gather 100 mp3 frames into a buffer and have SDL play it
                break
            }

            byte_reader := f.Reader()
            frame_bytes, _ := ioutil.ReadAll(byte_reader)
            playable_bytes.Write(frame_bytes)

            counter++
        }

        result := playable_bytes.Bytes()
        music.PlayBuffer(&result)

        playable_bytes.Truncate(0)
    }
}
