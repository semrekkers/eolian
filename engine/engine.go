// Package engine provides output through PortAudio
package engine

import (
	"fmt"
	"sync"
	"time"

	"github.com/brettbuddin/eolian/module"
	"github.com/gordonklaus/portaudio"
)

// Engine is the connection of the synthesizer to PortAudio
type Engine struct {
	sync.Mutex
	module.IO
	in *module.In

	device         *portaudio.DeviceInfo
	errors         chan error
	stop           chan error
	timings        chan time.Duration
	timingRequests chan chan time.Duration
}

// New returns a new Enngine
func New(deviceIndex int) (*Engine, error) {
	portaudio.Initialize()

	devices, err := portaudio.Devices()
	if err != nil {
		return nil, err
	}

	if deviceIndex >= len(devices) {
		return nil, fmt.Errorf("device index out of range")
	}

	fmt.Println("Output:", devices[deviceIndex].Name)

	m := &Engine{
		in:             &module.In{Name: "input", Source: module.NewBuffer(module.Value(0))},
		errors:         make(chan error),
		stop:           make(chan error),
		timings:        make(chan time.Duration),
		timingRequests: make(chan chan time.Duration),
		device:         devices[deviceIndex],
	}

	go collectTimings(m.timings, m.timingRequests)

	err = m.Expose("Engine", []*module.In{m.in}, nil)
	return m, err
}

func (e *Engine) CurrentLatency() time.Duration {
	r := make(chan time.Duration)
	e.timingRequests <- r
	return <-r
}

// Errors returns a channel that expresses any errors during operation of the Engine
func (e *Engine) Errors() chan error {
	return e.errors
}

func (e *Engine) params() portaudio.StreamParameters {
	params := portaudio.LowLatencyParameters(nil, e.device)
	params.Output.Channels = 1
	params.SampleRate = module.SampleRate
	params.FramesPerBuffer = module.FrameSize
	return params
}

// Run starts the Engine; running the audio stream
func (e *Engine) Run() {
	stream, err := portaudio.OpenStream(e.params(), e.portAudioCallback)

	if err != nil {
		e.errors <- err
		return
	}

	if err = stream.Start(); err != nil {
		e.errors <- err
		return
	}
	<-e.stop

	err = stream.Stop()
	if err == nil {
		err = stream.Close()
	}
	e.stop <- err
}

// Stop shuts down the Engine
func (e *Engine) Stop() error {
	defer portaudio.Terminate()
	e.stop <- nil
	err := <-e.stop
	close(e.errors)
	close(e.stop)
	return err
}

func (e *Engine) portAudioCallback(_, out []float32) {
	e.Lock()
	now := time.Now()
	frame := e.in.ReadFrame()
	for i := range out {
		out[i] = float32(frame[i])
	}
	e.timings <- time.Since(now)
	e.Unlock()
}

func collectTimings(timings <-chan time.Duration, requests chan chan time.Duration) {
	var current time.Duration
	for {
		select {
		case d := <-timings:
			current = d
		case ch := <-requests:
			ch <- current
		}
	}
}
