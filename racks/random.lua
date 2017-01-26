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
                filter = synth.LPFilter(),
                delay  = synth.FBLoopComb(),
            },
        }
    end

    local function patch(modules)
        with(modules.clock, function(c)
            c.osc:set      { pitch = hz(5) }
            c.multiple:set { input = c.osc:output('pulse') }
        end)

        with(modules.random, function(r)
            r.trigger:set {
                input   = modules.clock.multiple:output(0),
                divisor = 16,
            }
            r.series:set {
                clock   = modules.clock.multiple:output(1),
                trigger = r.trigger:output(),
                size    = 8,
            }
            r.quant:set { input = r.series:output('values') }

            local scale = theory.scale('C3', 'minorPentatonic', 2)
            for i,p in ipairs(scale) do
                r.quant:set { [i-1 .. '/pitch'] = p }
            end
        end)

        with(modules.voice, function(v)
            v.adsr:set {
                gate    = modules.random.series:output('gate'),
                attack  = ms(30),
                decay   = ms(50),
                sustain = 0.3,
                release = ms(1000),
            }
            v.osc:set {
                pitch = modules.random.quant:output(),
            }
            v.mix:set {
                { input = v.osc:output('sine') },
                { input = v.osc:output('saw'), level = 0.4 },
                { input = v.osc:output('sub') },
            }
            v.amp:set {
                a = v.mix:output(),
                b = v.adsr:output(),
            }
        end)

        with(modules.delay, function(d)
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
                input          = modules.voice.amp:output(),
                gain           = d.gain:output('sine'),
                feedbackReturn = d.filter:output(),
            }
            d.filter:set {
                input = d.delay:output('feedbackSend'),
                cutoff = hz(6000),
            }
        end)

        return modules.delay.delay:output()
    end

    return build, patch
end
