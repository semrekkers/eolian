# Eolian

[![Build Status](https://travis-ci.org/brettbuddin/eolian.svg?branch=master)](https://travis-ci.org/brettbuddin/eolian)

**Eolian** is a modular synthesizer. It provides a [variety of different
modules](https://github.com/brettbuddin/eolian/wiki/eolian.synth) and [music theory
primitives](https://github.com/brettbuddin/eolian/wiki/eolian.theory) that can be patched together to create music.  While the program
is written in [Go](https://golang.org/), you don't need to know Go to use it. Patches for the synthesizer are written in
[Lua](https://www.lua.org/). A [REPL](https://en.wikipedia.org/wiki/Read%E2%80%93eval%E2%80%93print_loop) is also
provided for interacting with the patches in real-time.

I started this project in attempt to learn more about digital signal processing and music theory. The project is named
after the music hall, **The Eolian**, in the book [*The Name of the
Wind*](https://www.amazon.com/Name-Wind-Patrick-Rothfuss/dp/0756404746/) by [Patrick
Rothfuss](http://patrickrothfuss.com).

## Dependencies

- [Go 1.7](http://golang.org)+
- [PortAudio](http://www.portaudio.com/)
- [PortMIDI](http://portmedia.sourceforge.net/portmidi/)
- [govendor](https://github.com/kardianos/govendor) (development only)

On macOS you can install these dependencies with: `brew install go portaudio portmidi`

## Installing

```
$ go get https://buddin.us/eolian
$ cd $GOPATH/src/buddin.us/eolian
$ make install
```

## Usage

```
$ eolian examples/random.lua
> -- Reload the file, rebuild the Rack and remount it
> Rack.build()
>
> -- Reload the file and only repatch it
> Rack.patch()
> 
> -- Set inputs or repatch modules
> Rack.modules
clock   table: 0xc420273680
random  table: 0xc420273980
voice   table: 0xc420273d40
delay   table: 0xc4202e2300
filter  (module)
mix     (module)
> Rack.modules.filter
cutoff          <--     7000.00Hz
input           <--     delay.delay/output
resonance       <--     10.00
bandpass        -->     (none)
highpass        -->     (none)
lowpass         -->     mix/0/input
> Rack.modules.filter:set { cutoff = hz(1000) }
cutoff          <--     1000.00Hz
input           <--     delay.delay/output
resonance       <--     10.00
bandpass        -->     (none)
highpass        -->     (none)
lowpass         -->     mix/0/input
```

### Building a Rack

Documentation for all modules and synthesizer features can be found on the
[Wiki](https://github.com/brettbuddin/eolian/wiki). I'll also be posting tutorials there soon.

A small set of example racks are maintained in the [examples directory](https://github.com/brettbuddin/eolian/tree/master/examples).

I keep all of my personal racks at [brettbuddin/eolian-racks](https://github.com/brettbuddin/eolian-racks). Be warned
that some of them might be out of date due to changes and new features in Eolian. I also frequently post [short videos
on Instagram](https://www.instagram.com/brettbuddin) of things I've been able to get out of it.

## Contributing

This project is very much in its infancy and there is still lots of work to be done. Wanna help out? Awesome! Mosey on over to
[CONTRIBUTING.md](https://github.com/brettbuddin/eolian/blob/master/CONTRIBUTING.md) and submit your first Pull Request.
