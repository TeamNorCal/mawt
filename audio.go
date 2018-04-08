package mawt

// This module is responsible for driving the audio
// output of this project.
//
// The audio portion of this project requires that
// files be converted to aiff files in 2 channel,
// 44100 Hz, pcm_s16le (16 Bit Signed Little Endian).
//
// The conversion from ogg format files to this format
// can be done using the libav-tools package installed
// using "sudo apt-get install libav-tools".  The
// conversion is done using a command line such as,
// "avconv -i assets/sounds/e-ambient.ogg -ar 44100 -ac 2 -acodec pcm_s16le assets/sounds/e-ambient.aiff".
//
// Playback using the same tools for testing purposes
// can be done using
// "aplay -D plug:dmix -f S16_LE -c 2 -r 44100 assets/sounds/e-ambient.aiff"

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cvanderschuere/alsa-go"

	"github.com/go-stack/stack"
	"github.com/karlmutch/errors"
)

var (
	audioDir = flag.String("audioDir", "assets/sounds", "The directory in which the audio aiff formatted event files can be found")
)

func InitAudio(ambientC <-chan string, sfxC <-chan []string, errorC chan<- errors.Error, quitC <-chan struct{}) (err errors.Error) {

	go runAudio(ambientC, sfxC, errorC, quitC)

	return nil
}

type effects struct {
	wakeup chan struct{}
	sfxs   []string
	stream alsa.AudioStream
	sync.Mutex
}

var (
	sfxs = effects{
		wakeup: make(chan struct{}, 1),
		sfxs:   []string{},
	}
)

func reportError(err errors.Error, errorC chan<- errors.Error) {
	select {
	case errorC <- err:
	case <-time.After(20 * time.Millisecond):
	}
}

func (sfxs *effects) playfile(fp string, quitC <-chan struct{}) (err errors.Error) {
	file, errGo := os.Open(fp)
	if errGo != nil {
		return errors.Wrap(errGo).With("file", fp).With("stack", stack.Trace().TrimRuntime())
	}
	defer file.Close()

	data := make([]byte, 8192)

	for {
		data = data[:cap(data)]
		n, errGo := file.Read(data)
		if errGo != nil {
			if errGo == io.EOF {
				return
			}
			return errors.Wrap(errGo).With("file", fp).With("stack", stack.Trace().TrimRuntime())
		}
		data = data[:n]

		select {
		case sfxs.stream.DataStream <- append([]byte(nil), data[:n]...):
		case <-quitC:
			return
		}
	}
}

func (sfxs *effects) playback(quitC <-chan struct{}) (err errors.Error) {
	for {
		fp := ""
		sfxs.Lock()
		if len(sfxs.sfxs) != 0 {
			fp = sfxs.sfxs[0]
			sfxs.sfxs = sfxs.sfxs[1:]
		}
		sfxs.Unlock()

		if len(fp) == 0 {
			return nil
		}
		if err = sfxs.playfile(fp, quitC); err != nil {
			return err
		}
	}
}

func playSFX(errorC chan<- errors.Error, quitC <-chan struct{}) {
	//Open ALSA pipe
	controlC := make(chan bool)
	//Create stream
	streamC := alsa.Init(controlC)

	sfxs.stream = alsa.AudioStream{Channels: 2,
		Rate:         int(44100),
		SampleFormat: alsa.INT16_TYPE,
		DataStream:   make(chan alsa.AudioData, 100),
	}

	streamC <- sfxs.stream

	defer func() {
		sfxs.Lock()
		close(sfxs.wakeup)
		sfxs.Unlock()
	}()

	sfxs.Lock()
	wakeup := sfxs.wakeup
	sfxs.Unlock()

	for {
		if err := sfxs.playback(quitC); err != nil {
			if errorC != nil {
				reportError(err, errorC)
			} else {
				fmt.Fprintln(os.Stderr, err.Error())
			}
		}

		select {
		case <-wakeup:
		case <-time.After(time.Second):
		case <-quitC:
			return
		}
	}
}

// Sounds possible at this point
//
// e-ambient, r-ambient, n-ambient
//
// e-capture, r-capture, n-capture
// e-loss, r-loss, n-loss
// e-resonator-deployed, r-resonator-deployed
// e-resonator-destroyed, r-resonator-destroyed

func runAudio(ambientC <-chan string, sfxC <-chan []string, errorC chan<- errors.Error, quitC <-chan struct{}) {

	go playAmbient(ambientC, errorC, quitC)

	go playSFX(errorC, quitC)

	for {
		select {

		case fns := <-sfxC:
			if len(fns) != 0 {
				sfxs.Lock()
				for _, fn := range fns {
					sfxs.sfxs = append(sfxs.sfxs, filepath.Join(*audioDir, fn+".aiff"))
				}
				// Wait a maximum of three seconds to wake up the audio
				// player for sound effects
				select {
				case sfxs.wakeup <- struct{}{}:
				case <-time.After(3 * time.Second):
				}
				sfxs.Unlock()
			}
		case <-quitC:
			return
		}
	}
}

type ambientFP struct {
	fp   string
	file *os.File
	sync.Mutex
}

func playAmbient(ambientC <-chan string, errorC chan<- errors.Error, quitC <-chan struct{}) {

	ambient := ambientFP{}

	go func() {
		for {
			select {
			case fn := <-ambientC:
				ambient.Lock()
				ambient.fp = filepath.Join(*audioDir, fn)
				ambient.fp += ".aiff"
				ambient.Unlock()
			case <-quitC:
				return
			}
		}
	}()

	//Open ALSA pipe
	controlC := make(chan bool)
	//Create stream
	streamC := alsa.Init(controlC)

	stream := alsa.AudioStream{Channels: 2,
		Rate:         int(44100),
		SampleFormat: alsa.INT16_TYPE,
		DataStream:   make(chan alsa.AudioData, 100),
	}

	streamC <- stream

	errGo := fmt.Errorf("")
	data := make([]byte, 8192)

	func() {
		fp := ""
		for {
			ambient.Lock()
			if fp != ambient.fp {
				if ambient.file != nil {
					ambient.file.Close()
					ambient.file = nil
				}
				if ambient.fp != "" {
					if ambient.file, errGo = os.Open(ambient.fp); errGo != nil {

						err := errors.Wrap(errGo).With("file", fp).With("stack", stack.Trace().TrimRuntime())
						reportError(err, errorC)

						ambient.fp = ""
						continue
					}
					fp = ambient.fp
				}

			}
			ambient.Unlock()
			if ambient.file == nil {
				select {
				case <-time.After(250 * time.Millisecond):
				}
				continue
			}

			data = data[:cap(data)]
			n, errGo := ambient.file.Read(data)
			if errGo != nil {
				if errGo == io.EOF {
					ambient.file.Seek(0, 0)
					continue
				}
				err := errors.Wrap(errGo).With("file", fp).With("stack", stack.Trace().TrimRuntime())
				reportError(err, errorC)
				continue
			}

			select {
			case stream.DataStream <- append([]byte(nil), data[:n]...):
			case <-quitC:
				return
			}
		}
	}()
}
