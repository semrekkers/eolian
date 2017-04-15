local synth = require('eolian.synth')
local clock = require('eolian.synth.clock')

return function(_)
    local function build()
        return {
            func = {
                {
                    input = synth.Control(),
                    shape = synth.Shape(),
                    rise  = synth.Control(),
                    fall  = synth.Control(),
                    mult  = synth.Multiple(),
                    amp   = synth.Multiply(),
                },
                {
                    input = synth.Control(),
                    mult  = synth.Multiple()
                },
                {
                    input = synth.Control(),
                    shape = synth.Shape(),
                    rise  = synth.Control(),
                    fall  = synth.Control(),
                    mult  = synth.Multiple(),
                    amp   = synth.Multiply(),
                },
            },

            clock   = clock(2),
            shorten = synth.Multiple(),
            osc     = synth.Oscillator(),
            waveMix = synth.Mix { size = 3 },
            noise   = synth.Noise(),
            amp1    = synth.Multiply(),
            amp2    = synth.Multiply(),
            mono    = synth.Multiple { size = 2 },
            reverb  = synth.TankReverb(),
        }
    end

    local function patch(r)
        r.clock:set { tempo = hz(0.3) }

        -- Loose clone of Make Noise Maths function generator
        with(r.func, function(f)
            with(f[1], function(ch)
                ch.input:set { input = 1 }
                ch.rise:set  { input = ms(10) }
                ch.fall:set  { input = ms(10000) }
                ch.shape:set {
                    ratio   = 0.001,
                    rise    = ch.rise:out(),
                    fall    = ch.fall:out(),
                }
                ch.amp:set  { a = ch.input:out(), b = ch.shape:out() }
                ch.mult:set { input = ch.amp:out() }
            end)
            with(f[2], function(ch)
                ch.input:set { mod = 0.1 }
                ch.mult:set { input = ch.input:out() }
            end)
            with(f[3], function(ch)
                ch.input:set { input = 1 }
                ch.rise:set  { input = ms(1) }
                ch.fall:set  { input = ms(5000) }
                ch.shape:set {
                    ratio   = 0.01,
                    rise    = ch.rise:out(),
                    fall    = ch.fall:out(),
                    cycle   = 1,
                }
                ch.amp:set  { a = ch.input:out(), b = ch.shape:out() }
                ch.mult:set { input = ch.amp:out() }
            end)
        end)

        -- Tone
        r.osc:set   { pitch = pitch('Eb2'), pulseWidth = 0.3 }
        r.waveMix:set { 
            { input = r.osc:out('sine') },
            { input = r.osc:out('pulse'), level = 0.5 },
            { input = r.osc:out('sub'), level = 0.5 },
        }
        r.noise:set { input = r.waveMix:out(), gain = 0.1 }

        -- Trigger both channel 1 and 3 envelopes
        r.func[1].shape:set { trigger = r.clock:out(0) }
        r.func[3].shape:set { trigger = r.clock:out(1) }

        -- Initial envelope onset generated by channel 1 is attenuated by channel 2
        -- The slope of the channel 1's fall attenuates both the rise and fall of channel 3
        -- Channel 3 is cycling which creates the bounces
        r.func[2].input:set { input = r.func[1].mult:out(0) }
        r.shorten:set       { input = r.func[2].mult:out(0) }
        r.func[3].rise:set  { mod = r.shorten:out(0) }
        r.func[3].fall:set  { mod = r.shorten:out(1) }

        -- Decrease the volume as the intitial onset plays itself out and apply that to our tone.
        r.amp1:set { a = r.func[3].mult:out(1), b = r.func[1].mult:out(1) }
        r.amp2:set { a = r.noise:out(), b = r.amp1:out() }

        r.mono:set { input = r.amp2:out() }
        r.reverb:set { a = r.mono:out(0), b = r.mono:out(1), decay = 0.1 }
        return r.reverb:out('a'), r.reverb:out('b')
    end

    return build, patch
end
