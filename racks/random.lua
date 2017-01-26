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
            set(c.osc, { pitch = hz(5) })
            set(c.multiple, { input = c.osc:output('pulse') })
        end)

        with(modules.random, function(r)
            local clock = modules.clock.multiple

            set(r.trigger, {
                input   = clock:output(0),
                divisor = 16,
            })
            set(r.series, {
                clock   = clock:output(1),
                trigger = r.trigger:output(),
                size    = 8,
            })
            set(r.quant, { input = r.series:output('values') })

            local scale = theory.scale('C3', 'minorPentatonic', 2)
            for i,p in ipairs(scale) do
                set(r.quant, i-1 .. '/pitch', p)
            end
        end)

        with(modules.voice, function(v)
            local gate  = modules.random.series:output('gate')
            local quant = modules.random.quant:output()

            set(v.adsr, {
                gate    = gate,
                attack  = ms(30),
                decay   = ms(50),
                sustain = 0.3,
                release = ms(1000),
            })
            set(v.osc, { pitch = quant })
            set(v.mix, {
                { input = v.osc:output('sine') },
                { input = v.osc:output('saw'), level = 0.4 },
                { input = v.osc:output('sub') },
            })
            set(v.amp, {
                a = v.mix:output(),
                b = v.adsr:output(),
            })
        end)

        with(modules.delay, function(d)
            local voice = modules.voice.amp:output()

            set(d.cutoff, {
                pitch = hz(0.1),
                amp   = 0.1,
            })
            set(d.gain, {
                pitch  = hz(0.2),
                amp    = 0.2,
                offset = 0.7
            })
            set(d.delay, {
                input          = voice,
                gain           = d.gain:output('sine'),
                feedbackReturn = d.filter:output(),
            })
            set(d.filter, {
                input = d.delay:output('feedbackSend'),
                cutoff = hz(6000),
            })
        end)

        return modules.delay.delay:output()
    end

    return build, patch
end
