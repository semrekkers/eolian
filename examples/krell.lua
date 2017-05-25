local synth  = require('eolian.synth')
local theory = require('eolian.theory')
local value  = require('eolian.value')
local hz, ms = value.hz, value.ms

-- Invoke `Rack.modules:start()` to get it started :)

return function(_)
    local function build()
        return {
            debug    = synth.Debug(),
            clock    = synth.OR(),
            clockDiv = synth.ClockDivide(),
            random   = synth.RandomSeries(),
            random2  = synth.Random(),

            rise    = synth.Control(),
            fall    = synth.Control(),
            tap     = synth.Tap(),
            shape   = synth.Shape(),
            gate    = synth.LPGate(),

            quant   = synth.Quantize { size = 20 },
            osc     = synth.Oscillator(),
            waveMix = synth.Mix { size = 4 },
            filter  = synth.Filter(),
            noise   = synth.Noise(),

            delay    = synth.FilteredFBDelay(),
            compress = synth.Compress(),
            mono     = synth.Multiple { size = 2 },
            reverb   = synth.TankReverb(),

            start = function(self)
                require('eolian.func').ping(self.clock, 'a')
            end,
        }
    end

    local function patch(r)
        r.clock:set       { a = 0, b = r.tap:out('tap') }
        r.clockDiv:set    { input = r.clock:out(), divisor = 20 }
        r.random:set      { clock = r.clock:out(), trigger = r.clockDiv:out() }
        r.random2:set     { clock = r.clock:out(), min = 0.1, max = 1 }

        -- Quantized steps to Eb minor pentatonic
        r.quant:set { input = r.random:out('value') }
        local scale = theory.scale('Eb2', 'minorPentatonic', 4)
        for i,p in ipairs(scale:pitches()) do
            r.quant:set(i-1 .. '/pitch', p)
        end

        -- Tone
        r.osc:set { pitch = r.quant:out(), pulseWidth = 0.2 }
        r.waveMix:set {
            { input = r.osc:out('triangle') },
            { input = r.osc:out('sub') },
        }
        r.filter:set { input = r.waveMix:out(), cutoff = hz(500) }

        -- Envelope length
        r.rise:set { input = ms(400), mod = r.random2:out('stepped') }
        r.fall:set { input = ms(1000), mod = r.random2:out('stepped') }

        -- Envelope
        r.shape:set  {
            trigger = r.clock:out(),
            rise    = r.rise:out(),
            fall    = r.fall:out(),
            ratio   = 0.001,
            cycle   = 1,
        }
        r.tap:set   { input = r.shape:out(), tap = r.shape:out('endcycle') }
        r.gate:set  { input = r.filter:out('lowpass'), control = r.tap:out() }
        r.delay:set { input = r.gate:out(), duration = ms(500), gain = 0.5 }

        r.compress:set { input = r.delay:out() }
        r.mono:set     { input = r.compress:out() }
        r.reverb:set   { a = r.mono:out(0), b = r.mono:out(1), cutoff = hz(700), decay = 0.8 }

        return r.reverb:out('a', 'b')
    end

    return build, patch
end
