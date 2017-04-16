local synth   = require('eolian.synth')
local theory = require('eolian.theory')

-- `ping(Rack.modules.clock, 'a')` to get it started :)

return function(_)
    local function build()
        return {
            debug       = synth.Debug(),
            clock       = synth.OR(),
            clockMult   = synth.Multiple { size = 3 },
            random      = synth.Random(),
            random2     = synth.Random(),
            random2Mult = synth.Multiple(),

            rise    = synth.Control(),
            fall    = synth.Control(),
            both    = synth.Multiple { size = 2 },
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
        }
    end

    local function patch(r)
        r.clock:set       { a = 0, b = r.tap:out('tap') }
        r.clockMult:set   { input = r.clock:out() }
        r.random:set      { clock = r.clockMult:out(0) }
        r.random2:set     { clock = r.clockMult:out(1), min = 0.1, max = 1 }
        r.random2Mult:set { input = r.random2:out('stepped') }

        -- Quantized steps to Eb minor pentatonic
        r.quant:set { input = r.random:out('stepped') }
        local scale = theory.scale('Eb2', 'minorPentatonic', 4)
        for i,p in ipairs(scale) do
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
        r.rise:set { input = ms(400), mod = r.random2Mult:out(0) }
        r.fall:set { input = ms(1000), mod = r.random2Mult:out(1) }

        -- Envelope
        r.shape:set  {
            trigger = r.clockMult:out(2),
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

        return r.reverb:out('a'), r.reverb:out('b')
    end

    return build, patch
end
