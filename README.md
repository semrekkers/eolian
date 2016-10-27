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
> rack = Rack.load('racks/circles.lua')
>
> -- Build and mount the Rack to the engine (you should hear audio)
> rack:mount()
> 
> -- Reload the file, rebuild the Rack and remount it
> rack:rebuild()
>
> -- Reload the file and only repatch it
> rack:repatch()
> 
> -- Set inputs or repatch modules
> Rack.inspect(rack.modules.low.lpf)
inputs:
- input: &{0xc420282510 output 0xc420282510}
- cutoff: 3000.00Hz
- resonance: 0
outputs:
- output (active)
> rack.modules.low.lpf:set { cutoff = hz(2000) }
...
```

You can find some example rack setups in the [racks](https://github.com/brettbuddin/eolian/tree/master/racks) directory.

### Example Sounds

Here's some example videos of things I've been able to get it to do at various stages during development:

- [Tinkering with FilteredReverb](https://www.instagram.com/p/BLxTrABjGhG/)
- [4-Pole filter](https://www.instagram.com/p/BKCLIGYjU_F/)
- [Karplus-Strong #1](https://www.instagram.com/p/BKx7jIpjL4O/)
- [Karplus-Strong #2](https://www.instagram.com/p/BKzAeRQjZ7N/)
- [MIDI controller](https://www.instagram.com/p/BKhJ42FDnSY/)
- [Static clicks #2](https://www.instagram.com/p/BKZQbtfj0OM/)
- [Reverb exploration](https://www.instagram.com/p/BKCJ98Dj2RS/)

## Contributing

This project is very much in its infancy and there is still lots of work to be done. Wanna help out? Awesome! Mosey on over to
[CONTRIBUTING.md](https://github.com/brettbuddin/eolian/tree/master/CONTRIBUTING.md) and submit your first Pull Request.

## TODO

- Documentation!
- Live-remapping of MIDI controller inputs to controls in a patch (what does the Lua interface look like?)
- REST API to view and manipulate the patch graph. D3 module graphs?
- Improved filters (Butterworth or Chebyshev)
- Different classifications of inputs (when they get updated)
- Band-pass filter
- Wavetable oscillators
- Value range standardization
- Better module introspection in the REPL
- Other weird and groovy modules
- More tests... of course...
- More friendly error handling
