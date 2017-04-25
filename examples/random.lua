local synth  = require('eolian.synth')
local theory = require('eolian.theory')

return function(_)
    local function build()
        return {
            clock  = synth.Clock(),
            random = {
                trigger = synth.ClockDivide(),
                series  = synth.RandomSeries(),
                quant   = synth.Quantize(),
            },
            voice = {
                lfo     = synth.Oscillator { algorithm = 'simple' },
                adsr    = synth.ADSR(),
                osc     = synth.Oscillator(),
                mix     = synth.Mix(),
                amp     = synth.LPGate(),
                distort = synth.Distort(),
            },
            delay = {
                gain   = synth.Oscillator { algorithm = 'simple' },
                filter = synth.Filter(),
                delay  = synth.FBLoopDelay(),
            },
            filter    = synth.Filter(),
            mix       = synth.Mix(),
            panLFO    = synth.Oscillator { algorithm = 'simple' },
            pan       = synth.Pan(),
            reverb    = synth.TankReverb(),
            crossfeed = synth.Crossfeed(),
        }
    end

    local function patch(rack)
        rack.clock:set { tempo = hz(7) }

        with(rack.random, function(r)
            r.trigger:set {
                input   = rack.clock:out(),
                divisor = 30,
            }
            r.series:set {
                clock   = rack.clock:out(),
                trigger = r.trigger:out(),
                size    = 16,
            }
            r.quant:set { input = r.series:out('value') }

            local scale = theory.scale('C3', 'minorPentatonic', 2)
            for i,p in ipairs(scale) do
                r.quant:set(i-1 .. '/pitch', p)
            end
        end)

        with(rack.voice, function(v)
            local gate  = rack.clock:out()
            local quant = rack.random.quant:out()

            v.adsr:set {
                gate    = gate,
                attack  = ms(20),
                decay   = ms(100),
                sustain = 0.1,
                release = ms(1000),
            }
            v.lfo:set { pitch = hz(0.5), amp = 0.1, offset = 0.6 }
            v.osc:set { pitch = quant, pulseWidth = v.lfo:out('sine') }
            v.mix:set {
                { input = v.osc:out('sine') },
                { input = v.osc:out('pulse') },
                { input = v.osc:out('saw'), level = 0.5 },
                { input = v.osc:out('sub') },
            }
            v.distort:set { input = v.mix:out(), gain = 8 }
            v.amp:set {
                input   = v.distort:out(),
                control = v.adsr:out(),
            }
        end)

        rack.filter:set {
            input     = rack.voice.amp:out(),
            cutoff    = hz(7000),
        }

        rack.mix:set { master = 0.15 }
        rack.mix:set {
            { input = rack.filter:out('lowpass') }
        }

        rack.panLFO:set { pitch = hz(0.5) }
        rack.pan:set { input = rack.mix:out(), bias = rack.panLFO:out('sine') }
        rack.crossfeed:set { a = rack.pan:out('a'), b = rack.pan:out('b'), amount = 0.3 }
        rack.reverb:set {
            a = rack.crossfeed:out('a'),
            b = rack.crossfeed:out('b'),
            cutoff = hz(500),
        }

        return rack.reverb:out('a'), rack.reverb:out('b')
    end

    return build, patch
end
