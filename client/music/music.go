package music

/*
 #include <stdlib.h>
*/
import (
    "log"
    "unsafe"
    "C"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_mixer"
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

// Plays mp3 from specified file path
func Play(song string) {
	if m, err := mix.LoadMUS(song); err != nil {
		log.Println(err)
	} else if err = m.Play(1); err != nil {
		log.Println(err)
	} else {
        // block until song is done
		for mix.PlayingMusic() {}

		m.Free()
	}
}

// Plays mp3 from in-memory byte buffer
func PlayBuffer(mp3_bytes *[]byte) {
    /*
    p := C.malloc(C.size_t(len(*mp3_bytes)))
    defer C.free(p)

    // copy the data into the buffer, by converting it to a Go array
    cBuf := (*[1 << 30]byte)(p)
    copy(cBuf[:], *mp3_bytes)
    //rc = C.the_function(p, C.int(buf.Len()))
    bytes_buffer := sdl.RWFromMem(p, len(*mp3_bytes))
*/

    bytes_buffer := sdl.RWFromMem(unsafe.Pointer(&(*mp3_bytes)[0]), len(*mp3_bytes))
    if m, err := mix.LoadMUS_RW(bytes_buffer, 0); err != nil {
        log.Println(err)
    } else if err = m.Play(1); err != nil {
        log.Println(err)
    } else {
        // block until song is done
        for mix.PlayingMusic() {}
        m.Free()
    }
}

func Quit() {
    mix.CloseAudio()
    mix.Quit()
    sdl.Quit()
}
