// Package midi provides MIDI input handling
package midi

import (
	"fmt"
	"sync"
	"time"

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

func terminate() error {
	if initialized {
		return portmidi.Terminate()
	}
	return nil
}

func findInputDevice(name string) (portmidi.DeviceID, error) {
	if name == "" {
		return -1, fmt.Errorf("no device name specified")
	}

	var deviceID portmidi.DeviceID = -1
	for i := 0; i < portmidi.CountDevices(); i++ {
		id := portmidi.DeviceID(i)
		info := portmidi.Info(id)
		if info.Name == name && info.IsInputAvailable {
			deviceID = id
		}
	}

	if deviceID == -1 {
		return -1, fmt.Errorf(`unknown device "%s"`, name)
	}

	return deviceID, nil
}

func streamEvents(s *portmidi.Stream, stop <-chan struct{}) <-chan portmidi.Event {
	ch := make(chan portmidi.Event)
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				time.Sleep(10 * time.Millisecond)
				events, err := s.Read(1024)
				if err != nil {
					continue
				}
				for i := range events {
					ch <- events[i]
				}
			}
		}
	}()
	return ch
}
