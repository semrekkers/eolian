// Package engine provides output through PortAudio
package engine

import (
	"fmt"
	"sync"
	"time"

	"buddin.us/eolian/module"
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
	timings        chan metrics
	timingRequests chan chan metrics
	stream         *portaudio.Stream
	originTime     time.Duration
}

// New returns a new Enngine
func New(deviceIndex int) (*Engine, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, err
	}

	devices, err := portaudio.Devices()
	if err != nil {
		return nil, err
	}

	if deviceIndex >= len(devices) {
		return nil, fmt.Errorf("device index out of range")
	}

	dev := devices[deviceIndex]
	fmt.Println("Output:", dev.Name)
	fmt.Println("Sample Rate:", module.SampleRate)
	fmt.Println("Frame Size:", module.FrameSize)
	fmt.Println("Latency:", dev.DefaultLowOutputLatency)

	m := &Engine{
		in:             &module.In{Name: "input", Source: module.NewBuffer(module.Value(0)), ForceSinking: true},
		errors:         make(chan error),
		stop:           make(chan error),
		timings:        make(chan metrics),
		timingRequests: make(chan chan metrics),
		device:         devices[deviceIndex],
	}

	go collectTimings(m.timings, m.timingRequests)

	err = m.Expose("Engine", []*module.In{m.in}, nil)
	return m, err
}

// TotalElapsed returns the current wallclock duration of the session
func (e *Engine) TotalElapsed() time.Duration {
	r := make(chan metrics)
	e.timingRequests <- r
	return (<-r).TotalElapsed - e.originTime
}

// Latency returns the current latency within the PortAudio callback. It's an indicator of how computationally expensive
// your Rack is, and does not include any latency between PortAudio and your speakers.
func (e *Engine) Latency() time.Duration {
	r := make(chan metrics)
	e.timingRequests <- r
	return (<-r).Callback
}

// Load returns the current CPU load of the underlying audio engine
func (e *Engine) Load() float64 {
	r := make(chan metrics)
	e.timingRequests <- r
	return (<-r).Load
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
	var err error
	e.stream, err = portaudio.OpenStream(e.params(), e.portAudioCallback)
	if err != nil {
		e.errors <- err
		return
	}
	e.originTime = e.stream.Time()

	if err = e.stream.Start(); err != nil {
		e.errors <- err
		return
	}
	<-e.stop

	err = e.stream.Stop()
	if err == nil {
		err = e.stream.Close()
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
	e.stream.Time()
	now := time.Now()
	frame := e.in.ReadFrame()
	for i := range out {
		out[i] = float32(frame[i])
	}
	e.timings <- metrics{
		Callback:     time.Since(now),
		TotalElapsed: e.stream.Time(),
		Load:         e.stream.CpuLoad(),
	}
	e.Unlock()
}

func collectTimings(timings <-chan metrics, requests chan chan metrics) {
	var current metrics
	for {
		select {
		case d := <-timings:
			current = d
		case ch := <-requests:
			ch <- current
		}
	}
}

type metrics struct {
	TotalElapsed, Callback time.Duration
	Load                   float64
}
