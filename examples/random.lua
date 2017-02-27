return function(env)
    local synth  = require('eolian.synth')
    local theory = require('eolian.theory')

    local function build()
        return {
            clock  = {
                osc      = synth.Oscillator(),
                multiple = synth.Multiple(),
            },
            recordTrigger = synth.ClockDivide(),
            random = {
                trigger = synth.ClockDivide(),
                series  = synth.RandomSeries(),
                quant   = synth.Quantize(),
            },
            voice = {
                adsr = synth.ADSR(),
                osc  = synth.Oscillator(),
                mix  = synth.Mix(),
                amp  = synth.LPGate(),
            },
            delay = {
                gain   = synth.Oscillator { algorithm = 'simple' },
                filter = synth.Filter(),
                delay  = synth.FBLoopComb(),
            },
            tape   = synth.Tape(),
            filter = synth.Filter(),
            mix    = synth.Mix(),
        }
    end

    local function patch(rack)
        with(rack.clock, function(c)
            c.osc:set { pitch = hz(5) }
            c.multiple:set { input = c.osc:out('pulse') }
        end)

        with(rack.random, function(r)
            local clock = rack.clock.multiple

            r.trigger:set {
                input   = clock:out(0),
                divisor = 16,
            }
            r.series:set {
                clock   = clock:out(1),
                trigger = r.trigger:out(),
                size    = 8,
            }
            r.quant:set { input = r.series:out('values') }

            local scale = theory.scale('C3', 'minorPentatonic', 2)
            for i,p in ipairs(scale) do
                r.quant:set(i-1 .. '/pitch', p)
            end
        end)

        with(rack.voice, function(v)
            local gate  = rack.random.series:out('gate')
            local quant = rack.random.quant:out()

            v.adsr:set {
                gate    = gate,
                attack  = ms(20),
                decay   = ms(100),
                sustain = 0.1,
                release = ms(1000),
            }
            v.osc:set { pitch = quant }
            v.mix:set {
                { input = v.osc:out('sine') },
                { input = v.osc:out('saw'), level = 0.1 },
                { input = v.osc:out('sub'), level = 0.5 },
            }
            v.amp:set {
                input = v.mix:out(),
                ctrl  = v.adsr:out(),
            }
        end)

        with(rack.delay, function(d)
            local voice = rack.voice.amp:out()

            d.gain:set {
                pitch  = hz(0.2),
                amp    = 0.5,
                offset = 0.7
            }
            d.delay:set {
                input          = voice,
                gain           = d.gain:out('sine'),
                feedbackReturn = d.filter:out('bandpass'),
            }
            d.filter:set {
                input = d.delay:out('feedbackSend'),
                cutoff = hz(1000),
            }
        end)

        rack.recordTrigger:set {
            input   = rack.clock.multiple:out(2),
            divisor = 16,
        }

        rack.tape:set {
            input  = rack.delay.delay:out(),
            record = rack.recordTrigger:out(),
        }
        rack.filter:set {
            input  = rack.tape:out(),
            cutoff = hz(7000),
            resonance = 10,
        }

        rack.mix:set { master = 0.15 }
        rack.mix:set {
            { input = rack.filter:out('lowpass') }
        }

        return rack.mix:out()
    end

    return build, patch
end
