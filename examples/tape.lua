return function(_)
    local synth  = require('eolian.synth')
    local theory = require('eolian.theory')
    local ctrl   = require('eolian.synth.control')

    local func = require('eolian.func')
    local with, set, out = func.with,
                           func.set,
                           func.out

    local value = require('eolian.value')
    local hz, ms = value.hz, value.ms

    local function build()
        return {
            midi  = synth.MIDIController { device = 'Launch Control' },
            clock = synth.Clock(),
            random = {
                trigger = synth.ClockDivide(),
                series  = synth.RandomSeries(),
                quant   = synth.Quantize(),
            },
            voice = {
                adsr = ctrl(synth.ADSR(), {
                    attack  = { min = ms(10), max = ms(1000) },
                    decay   = { min = ms(10), max = ms(1000) },
                    release = { min = ms(10), max = ms(1000) },
                    sustain = { max = 1 },
                }),
                osc = synth.Oscillator(),
                mix = synth.Mix(),
                amp = synth.Multiply(),
            },
            tape = ctrl(synth.Tape(), {
                record   = { min = -1, max = 1 },
                splice   = { min = -1, max = 1 },
                unsplice = { min = -1, max = 1 },
                speed    = { min = -1, max = 1 },
                bias     = { min = -1, max = 1 },
                reset    = { min = -1, max = 1 },
                organize = { max = 1 },
            }),
            delay = ctrl(synth.FilteredFBDelay(), {
                cutoff   = { min = hz(50), max = hz(3000) },
                duration = { min = ms(10), max = ms(1000) },
                gain     = { max = 1 },
            }),
            filter = ctrl(synth.Filter(), {
                cutoff    = { min = hz(1000), max = hz(5000) },
                resonance = { min = 1, max = 50 },
            }),
        }
    end

    local function patch(m)
        local channel = m.midi:ns('1/cc')
        local cc      = function(n) return out(channel, n) end

        set(m.clock, { tempo = hz(9) })

        with(m.random, function(r)
            set(r.trigger, {
                input   = out(m.clock),
                divisor = 16,
            })
            set(r.series, {
                clock   = out(m.clock),
                trigger = out(r.trigger),
                size    = 8,
            })
            set(r.quant, { input = out(r.series, 'value') })

            local scale = theory.scale('C3', 'minorPentatonic', 2)
            for i,p in ipairs(scale:pitches()) do
                set(r.quant, i-1 .. '/pitch', p)
            end
        end)

        local voice = with(m.voice, function(v)
            local series = m.random.series
            set(v.adsr, {
                gate    = out(series, 'gate'),
                attack  = cc(45),
                decay   = cc(46),
                sustain = cc(47),
                release = cc(48),
            })
            set(v.osc, { pitch = out(m.random.quant), pulseWidth = cc(28) })
            set(v.mix, {
                { input = out(v.osc, 'sine') },
                { input = out(v.osc, 'pulse'), level = 0.4 },
            })
            set(v.amp, { a = out(v.mix), b = out(v.adsr) })
            return v.amp
        end)

        set(m.tape, {
            input    = out(voice),
            record   = cc(9),
            splice   = cc(10),
            unsplice = cc(11),
            reset    = cc(12),
            speed    = cc(21),
            bias     = cc(22),
            organize = cc(23),
            zoom     = cc(43),
            slide    = cc(44),
        })
        set(m.delay, {
            input    = out(m.tape),
            gain     = cc(25),
            cutoff   = cc(26),
            duration = cc(27),
        })
        set(m.filter, { input = out(m.delay), cutoff = cc(41), resonance = cc(42) })

        local sink = out(m.filter, 'lowpass')

        return sink, sink
    end

    return build, patch
end
