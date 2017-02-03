return function(env)
    local synth       = require('eolian.synth')
    local theory      = require('eolian.theory')
    local interpolate = require('eolian.synth.interpolate')

    local function build()
        return {
            midi = synth.MIDIController { device = 'Launch Control' },
            clock = {
                osc      = synth.Oscillator(),
                multiple = synth.Multiple(),
            },
            random = {
                trigger = synth.ClockDivide(),
                series  = synth.RandomSeries(),
                quant   = synth.Quantize(),
            },
            voice = {
                adsr = interpolate(synth.ADSR(), {
                    attack  = { min = ms(10), max = ms(1000) },
                    decay   = { min = ms(10), max = ms(1000) },
                    release = { min = ms(10), max = ms(1000) },
                    sustain = { max = 1 },
                }),
                osc = synth.Oscillator(),
                mix = synth.Mix(),
                amp = synth.Multiply(),
            },
            tape = interpolate(synth.Tape(), {
                record   = { min = -1, max = 1 },
                splice   = { min = -1, max = 1 },
                unsplice = { min = -1, max = 1 },
                speed    = { min = -1, max = 1 },
                bias     = { min = -1, max = 1 },
                reset    = { min = -1, max = 1 },
                organize = { max = 1 },
            }),
            delay = interpolate(synth.FilteredFBComb(), {
                cutoff   = { min = hz(50), max = hz(3000) },
                duration = { min = ms(10), max = ms(1000) },
                gain     = { max = 1 },
            }),
            filter = interpolate(synth.Filter(), {
                cutoff    = { min = hz(1000), max = hz(5000) },
                resonance = { min = 1, max = 50 },
            }),
        }
    end

    local function patch(modules)
        local channel = modules.midi:ns('cc/1')
        local cc      = function(n) return out(channel, n) end

        local clock = with(modules.clock, function(c)
            set(c.osc, { pitch = hz(9) })
            set(c.multiple, { input = out(c.osc, 'pulse') })
            return c.multiple
        end)

        with(modules.random, function(r)
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

        local voice = with(modules.voice, function(v)
            local series = modules.random.series
            set(v.adsr, { 
                gate    = out(series, 'gate'),
                attack  = cc(45),
                decay   = cc(46),
                sustain = cc(47),
                release = cc(48),
            })
            set(v.osc, { pitch = out(modules.random.quant) })
            set(v.mix, {
                { input = out(v.osc, 'sine') },
                { input = out(v.osc, 'saw'), level = 0.2 },
            })
            set(v.amp, { a = out(v.mix), b = out(v.adsr) })
            return v.amp
        end)

        set(modules.delay, {
            input    = out(voice),
            gain     = cc(25),
            cutoff   = cc(26),
            duration = cc(27),
        })

        set(modules.tape, { 
            input    = out(modules.delay),
            record   = cc(9),
            splice   = cc(10),
            unsplice = cc(11),
            reset    = cc(12),
            speed    = cc(21),
            bias     = cc(22),
            organize = cc(23),
        })
        set(modules.filter, { input = out(modules.tape), cutoff = cc(41), resonance = cc(42) })

        return out(modules.filter, 'lowpass')
    end

    return build, patch
end
