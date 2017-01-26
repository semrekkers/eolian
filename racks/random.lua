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
            set(c.multiple, { input = out(c.osc, 'pulse') })
        end)

        with(modules.random, function(r)
            local clock = modules.clock.multiple

            set(r.trigger, {
                input   = out(clock, 0),
                divisor = 16,
            })
            set(r.series, {
                clock   = out(clock, 1),
                trigger = out(r.trigger),
                size    = 8,
            })
            set(r.quant, { input = out(r.series, 'values') })

            local scale = theory.scale('C3', 'minorPentatonic', 2)
            for i,p in ipairs(scale) do
                set(r.quant, i-1 .. '/pitch', p)
            end
        end)

        with(modules.voice, function(v)
            local gate  = out(modules.random.series, 'gate')
            local quant = out(modules.random.quant)

            set(v.adsr, {
                gate    = gate,
                attack  = ms(30),
                decay   = ms(50),
                sustain = 0.3,
                release = ms(1000),
            })
            set(v.osc, { pitch = quant })
            set(v.mix, {
                { input = out(v.osc, 'sine') },
                { input = out(v.osc, 'saw'), level = 0.4 },
                { input = out(v.osc, 'sub') },
            })
            set(v.amp, {
                a = out(v.mix),
                b = out(v.adsr),
            })
        end)

        with(modules.delay, function(d)
            local voice = out(modules.voice.amp)

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
                gain           = out(d.gain, 'sine'),
                feedbackReturn = out(d.filter),
            })
            set(d.filter, {
                input = out(d.delay, 'feedbackSend'),
                cutoff = hz(6000),
            })
        end)

        return out(modules.delay.delay)
    end

    return build, patch
end
