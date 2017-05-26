package music

import (
    "log"
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

    if err := mix.OpenAudio(22050, mix.DEFAULT_FORMAT, 2, 4096); err != nil {
        log.Println(err)
        return
    }
}

func Play(song string) {
	if music, err := mix.LoadMUS(song); err != nil {
		log.Println(err)
	} else if err = music.Play(1); err != nil {
		log.Println(err)
	} else {
		for mix.PlayingMusic() {} // block until song is done
		music.Free()
	}
}

// TODO: play an mp3 chunk; need to figure out the argument type
func PlayChunk() {

}

func Quit() {
    mix.CloseAudio()
    mix.Quit()
    sdl.Quit()
}
