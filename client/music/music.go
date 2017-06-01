package music

import (
    "fmt"
    "log"
    "unsafe"
    "io/ioutil"
    //"bytes"
    "encoding/gob"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_mixer"
    "github.com/tcolgate/mp3"
)

func Init() {
    if err := sdl.Init(sdl.INIT_AUDIO); err != nil {
    	log.Println(err)
    	return
    }

    if err := mix.Init(mix.INIT_MP3); err != nil {
    	log.Println(err)
    	return
    }

    // Default: 22050, mix.DEFAULT_FORMAT, 2, 4096
    // we want 44.1 kHz/16 bit quality for our songs
    if err := mix.OpenAudio(44100, mix.DEFAULT_FORMAT, 2, 4096); err != nil {
        log.Println(err)
        return
    }
}

func Quit() {
    mix.CloseAudio()
    mix.Quit()
    sdl.Quit()
}

// Concurrently writes to the given song buffer as the song is being played.
// Assumes song will be <= 50MB
func PlayFromMp3Dec(dec *mp3.Decoder, buf *[20 * 1024 * 1024]byte) {
    var currIndex int
    var m *mix.Music

    currIndex = 0
    bufferFramesToByteArray(dec, buf, &currIndex)

    ptrToBuf := sdl.RWFromMem(unsafe.Pointer(&(buf)[0]), len(buf))
    m, _ = mix.LoadMUS_RW(ptrToBuf, 0)

    go func() {
        for { // write N byte sized chunks of frames to the song buffer
            done := bufferFramesToByteArray(dec, buf, &currIndex)
            if done {
                break
            }
        }
    }()

    // TODO: Delay until everyone is ready!

    m.Play(1)
    for mix.PlayingMusic() {}
    m.Free()

    fmt.Println("Done Playing")
}

func PlayFromGobDec(dec *gob.Decoder, buf *[20 * 1024 * 1024]byte) {

}

// Buffer frames from the decoder into the bytes array.
// Returns true if nothing more to buffer.
// Better quality music.
func bufferFramesToByteArray(dec *mp3.Decoder, buf *[20 * 1024 * 1024]byte, currIndex *int) (bool) {
    skipped := 0
    var frame mp3.Frame
    for i := 0; i < 300; i++ {
        if err := dec.Decode(&frame, &skipped); err != nil {
            return true
        }

        reader := frame.Reader()
        frame_bytes, _ := ioutil.ReadAll(reader)

        for j := 0; j < len(frame_bytes); j++ {
            buf[*currIndex + j] = frame_bytes[j]
        }

        *currIndex = *currIndex + len(frame_bytes)
    }

    return false
}
