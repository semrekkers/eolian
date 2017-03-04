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
	timings        chan EngineMetrics
	timingRequests chan chan EngineMetrics
	stream         *portaudio.Stream
	originTime     time.Duration
}

type EngineMetrics struct {
	TotalElapsed, Callback time.Duration
	Load                   float64
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

	dev := devices[deviceIndex]
	fmt.Println("Output:", dev.Name)
	fmt.Println("Sample Rate:", module.SampleRate)
	fmt.Println("Frame Size:", module.FrameSize)
	fmt.Println("Latency:", dev.DefaultLowOutputLatency)

	m := &Engine{
		in:             &module.In{Name: "input", Source: module.NewBuffer(module.Value(0)), ForceSinking: true},
		errors:         make(chan error),
		stop:           make(chan error),
		timings:        make(chan EngineMetrics),
		timingRequests: make(chan chan EngineMetrics),
		device:         devices[deviceIndex],
	}

	go collectTimings(m.timings, m.timingRequests)

	err = m.Expose("Engine", []*module.In{m.in}, nil)
	return m, err
}

func (e *Engine) TotalElapsed() time.Duration {
	r := make(chan EngineMetrics)
	e.timingRequests <- r
	return (<-r).TotalElapsed - e.originTime
}

func (e *Engine) CurrentLatency() time.Duration {
	r := make(chan EngineMetrics)
	e.timingRequests <- r
	return (<-r).Callback
}

func (e *Engine) Load() float64 {
	r := make(chan EngineMetrics)
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
	e.timings <- EngineMetrics{
		Callback:     time.Since(now),
		TotalElapsed: e.stream.Time(),
		Load:         e.stream.CpuLoad(),
	}
	e.Unlock()
}

func collectTimings(timings <-chan EngineMetrics, requests chan chan EngineMetrics) {
	var current EngineMetrics
	for {
		select {
		case d := <-timings:
			current = d
		case ch := <-requests:
			ch <- current
		}
	}
}
