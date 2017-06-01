package music

import (
    "fmt"
    "log"
    "unsafe"
    "io/ioutil"
    "bytes"
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

// Plays mp3 from specified file path
func PlaySongFromFile(filePath string) {
	if m, err := mix.LoadMUS(filePath); err != nil {
		log.Println(err)
	} else if err = m.Play(1); err != nil {
		log.Println(err)
	} else {
        // block until song is done
		for mix.PlayingMusic() {}
		m.Free()
	}
}

// Buffer mp3 frames and then play the chunk. While the song is playing,
// pre-emptively buffer the next chunk
func PlayFromSongChunks(dec *mp3.Decoder, framesPerChunk int) {
    var playingBuf bytes.Buffer
    var standbyBuf bytes.Buffer

    var playingBytes []byte
    var standbyBytes []byte

    var m1 *mix.Music // current music to play
    var m2 *mix.Music // next music to play

    var done bool
    done = false

    bufferFramesToByteBuffer(dec, &playingBuf, framesPerChunk)
    playingBuf.Truncate(playingBuf.Len())
    playingBytes = playingBuf.Bytes() // is this slow?
    buf := sdl.RWFromMem(unsafe.Pointer(&(playingBytes)[0]), len(playingBytes))
    m1, _ = mix.LoadMUS_RW(buf, 0)
    m2, _ = mix.LoadMUS_RW(buf, 0)

    for {
        // pre-emptively load next chunk of music
        go func() {
            m2.Free()
            done = bufferFramesToByteBuffer(dec, &standbyBuf, framesPerChunk)
            standbyBuf.Truncate(standbyBuf.Len())
            standbyBytes = standbyBuf.Bytes() // is this slow?
            buf := sdl.RWFromMem(unsafe.Pointer(&(standbyBytes)[0]), len(standbyBytes))
            m2, _ = mix.LoadMUS_RW(buf, 0)
        }()

        // play music
        m1.Play(1)
        for mix.PlayingMusic() {}

        if done {
            break
        }

        // pre-emptively load next chunk of music
        go func() {
            m1.Free()
            done = bufferFramesToByteBuffer(dec, &playingBuf, framesPerChunk)
            playingBuf.Truncate(playingBuf.Len())
            playingBytes = playingBuf.Bytes() // is this slow?
            buf := sdl.RWFromMem(unsafe.Pointer(&(playingBytes)[0]), len(playingBytes))
            m1, _ = mix.LoadMUS_RW(buf, 0)
        }()

        // play music
        m2.Play(1)
        for mix.PlayingMusic() {}

        if done {
            break
        }
    }

    fmt.Println("Done playing!")
}

// Concurrently writes to the given song buffer as the song is being played.
// Assumes song will be <= 50MB
func PlayFromSongBuf(dec *mp3.Decoder, buf *[50 * 1024 * 1024]byte) {
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

    m.Play(1)
    for mix.PlayingMusic() {}
    m.Free()

    fmt.Println("Done Playing")
}

// Buffer frames from the decoder into the bytes buffer.
// Returns true if nothing more to buffer.
// Better memory usage.
func bufferFramesToByteBuffer(dec *mp3.Decoder, buf *bytes.Buffer, framesPerChunk int) (bool) {
    buf.Reset() // empty the buffer

    skipped := 0
    var frame mp3.Frame
    for i := 0; i < framesPerChunk; i++ {
        if err := dec.Decode(&frame, &skipped); err != nil {
            return true
        }

        reader := frame.Reader()
        frame_bytes, _ := ioutil.ReadAll(reader)
        buf.Write(frame_bytes)
    }

    return false
}

// Buffer frames from the decoder into the bytes array.
// Returns true if nothing more to buffer.
// Better quality music.
func bufferFramesToByteArray(dec *mp3.Decoder, buf *[50 * 1024 * 1024]byte, currIndex *int) (bool) {
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
