# Eolian

**Eolian** is a modular synthesizer. It provides a variety of different modules that can be patched together to create
music. While the program is written in [Go](https://golang.org/), you don't need to know Go to use it. Patches for the
synthesizer are written in [Lua](https://www.lua.org/). A
[REPL](https://en.wikipedia.org/wiki/Read%E2%80%93eval%E2%80%93print_loop) is also provided for interacting with the
patches in real-time.

## Eolian? Don't you mean Aeolian?

**The Eolian** is a music hall in the book [*The Name of the
Wind*](https://www.amazon.com/Name-Wind-Patrick-Rothfuss/dp/0756404746/) by [Patrick Rothfuss](http://patrickrothfuss.com).

## Dependencies

- [Go 1.6](http://golang.org)+
- [PortAudio](http://www.portaudio.com/)
- [PortMIDI](http://portmedia.sourceforge.net/portmidi/)

On macOS you can install these dependencies with: `brew install go portaudio portmidi`

## Building

```
$ make
```

## Usage

```
$ make install
$ eolian
>> rack = Rack.load('racks/circles.lua')
>> Rack.inspect(rack.modules)
>> rack:rebuild()
>> rack:repatch()
...
```

You can find some example rack setups in the [racks](https://github.com/brettbuddin/eolian/tree/master/racks) directory.

## TODO

- Improved filters (Butterworth or Chebyshev)
- Different classifications of inputs (when they get updated)
- Band-pass filter
- Wavetable oscillators
- Value range standardization
- Better module introspection in the REPL
- Other weird and groovy modules
- More tests... of course...
- More friendly error handling
