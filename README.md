# Eolian

[![Build Status](https://travis-ci.org/brettbuddin/eolian.svg?branch=master)](https://travis-ci.org/brettbuddin/eolian)

**Eolian** is a modular synthesizer. It provides a [variety of different
modules](https://github.com/brettbuddin/eolian/wiki/eolian.synth) and music theory primitives that can be patched
together to create music.  While the program is written in [Go](https://golang.org/), you don't need to know Go to use
it. Patches for the synthesizer are written in [Lua](https://www.lua.org/). A
[REPL](https://en.wikipedia.org/wiki/Read%E2%80%93eval%E2%80%93print_loop) is also provided for interacting with the
patches in real-time.

This project is named after the music hall, **The Eolian**, in the book [*The Name of the
Wind*](https://www.amazon.com/Name-Wind-Patrick-Rothfuss/dp/0756404746/) by [Patrick Rothfuss](http://patrickrothfuss.com).

## Dependencies

- [Go 1.7](http://golang.org)+
- [PortAudio](http://www.portaudio.com/)
- [PortMIDI](http://portmedia.sourceforge.net/portmidi/)
- [govendor](https://github.com/kardianos/govendor) (development only)

On macOS you can install these dependencies with: `brew install go portaudio portmidi`

## Installing

```
$ git clone https://github.com/brettbuddin/eolian.git $GOPATH/src/github.com/brettbuddin/eolian
$ cd $GOPATH/src/github.com/brettbuddin/eolian
$ make install
```

## Usage

```
$ eolian racks/circles.lua
> -- Reload the file, rebuild the Rack and remount it
> Rack.build()
> -- Reload the file and only repatch it
> Rack.patch()
> -- Set inputs or repatch modules
> Rack.modules.low.filter
cutoff          <--     3000.00Hz
input           <--     low.mix/output
resonance       <--     1
output          -->     mix/1/input
> Rack.modules.low.filter:set('cutoff', hz(2000))
cutoff          <--     2000.00Hz
input           <--     low.mix/output
resonance       <--     1
output          -->     mix/1/input
...
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
