package midi

import (
	"sync"

	"github.com/rakyll/portmidi"
)

var (
	initialized bool
	once        = sync.Once{}
)

func initMIDI() {
	once.Do(func() {
		if err := portmidi.Initialize(); err != nil {
			panic(err)
		}
		initialized = true
	})
}

func Terminate() error {
	if initialized {
		return portmidi.Terminate()
	}
	return nil
}
