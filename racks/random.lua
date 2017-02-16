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
                amp  = synth.Multiply(),
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
            c.multiple:set { input = out(c.osc, 'pulse') }
        end)

        with(modules.random, function(r)
            local clock = modules.clock.multiple

            r.trigger:set {
                input   = out(clock, 0),
                divisor = 16,
            }
            r.series:set {
                clock   = out(clock, 1),
                trigger = out(r.trigger),
                size    = 8,
            }
            r.quant:set { input = out(r.series, 'values') }

            local scale = theory.scale('C3', 'minorPentatonic', 2)
            for i,p in ipairs(scale) do
                r.quant:set(i-1 .. '/pitch', p)
            end
        end)

        with(modules.voice, function(v)
            local gate  = out(modules.random.series, 'gate')
            local quant = out(modules.random.quant)

            v.adsr:set {
                gate    = gate,
                attack  = ms(30),
                decay   = ms(50),
                sustain = 0.3,
                release = ms(1000),
            }
            v.osc:set { pitch = quant }
            v.mix:set {
                { input = out(v.osc, 'sine') },
                { input = out(v.osc, 'saw'), level = 0.4 },
                { input = out(v.osc, 'sub') },
            }
            v.amp:set {
                a = out(v.mix),
                b = out(v.adsr),
            }
        end)

        with(modules.delay, function(d)
            local voice = out(modules.voice.amp)

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
                gain           = out(d.gain, 'sine'),
                feedbackReturn = out(d.filter, 'lowpass'),
            }
            d.filter:set {
                input = out(d.delay, 'feedbackSend'),
                cutoff = hz(6000),
            }
        end)

        return out(modules.delay.delay)
    end

    return build, patch
end
