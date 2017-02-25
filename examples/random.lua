return function(env)
    local synth  = require('eolian.synth')
    local theory = require('eolian.theory')

    local function build()
        return {
            clock  = {
                osc      = synth.Oscillator(),
                multiple = synth.Multiple(),
            },
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
                cutoff = synth.Oscillator { algorithm = 'simple' },
                gain   = synth.Oscillator { algorithm = 'simple' },
                filter = synth.Filter(),
                delay  = synth.FBLoopComb(),
            },
        }
    end

    local function patch(modules)
        with(modules.clock, function(c)
            c.osc:set { pitch = hz(5) }
            c.multiple:set { input = c.osc:out('pulse') }
        end)

        with(modules.random, function(r)
            local clock = modules.clock.multiple

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

        with(modules.voice, function(v)
            local gate  = modules.random.series:out('gate')
            local quant = modules.random.quant:out()

            v.adsr:set {
                gate    = gate,
                attack  = ms(30),
                decay   = ms(50),
                sustain = 0.3,
                release = ms(1000),
            }
            v.osc:set { pitch = quant }
            v.mix:set {
                { input = v.osc:out('sine') },
                { input = v.osc:out('saw'), level = 0.4 },
                { input = v.osc:out('sub') },
            }
            v.amp:set {
                input = v.mix:out(),
                ctrl  = v.adsr:out(),
            }
        end)

        with(modules.delay, function(d)
            local voice = modules.voice.amp:out()

            d.cutoff:set {
                pitch = hz(0.1),
                amp   = 0.1,
            }
            d.gain:set {
                pitch  = hz(0.2),
                amp    = 0.2,
                offset = 0.7
            }
            d.delay:set {
                input          = voice,
                gain           = d.gain:out('sine'),
                feedbackReturn = d.filter:out('lowpass'),
            }
            d.filter:set {
                input = d.delay:out('feedbackSend'),
                cutoff = hz(6000),
            }
        end)

        return modules.delay.delay:out()
    end

    return build, patch
end
