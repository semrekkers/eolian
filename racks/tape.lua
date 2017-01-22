local interpolated = function(module, ranges)
    local synth = require('eolian.synth')
    local proxy = require('eolian.synth.proxy')

    local proxied = {}

    if type(module) == 'function' then
        module = module()
    end

    for k, range in pairs(ranges) do
        proxied[k] = synth.Interpolate(range)
        module:set { [k] = proxied[k]:output() }
    end

    return {
        ns = function(_, prefix)
            return interpolated(module:ns(prefix), ranges)
        end,
        set = function(_, inputs)
            for k, v in pairs(inputs) do
                if type(v) == 'table' then
                    local prefix = k
                    for k,v in pairs(v) do
                        local full = prefix .. '/' .. k
                        if proxied[full] ~= nil then
                            proxied[full]:set { input = v }
                        else
                            module:set { [full] = v }
                        end
                    end
                else
                    if proxied[k] ~= nil then
                        proxied[k]:set { input = v }
                    else
                        module:set { [k] = v }
                    end
                end
            end
        end,
        output = proxy.outputs(module),
    }
end

return function(env)
    local synth = require('eolian.synth')

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
                adsr = interpolated(synth.ADSR(), {
                    attack  = { min = ms(10), max = ms(1000) },
                    decay   = { min = ms(10), max = ms(1000) },
                    release = { min = ms(10), max = ms(1000) },
                    sustain = { min = 0, max = 1 },
                }),
                osc  = synth.Oscillator(),
                mix  = synth.Mix(),
                amp  = synth.Multiply(),
            },
            tape = interpolated(synth.Tape(), {
                record   = { min = -1, max = 1 },
                splice   = { min = -1, max = 1 },
                unsplice = { min = -1, max = 1 },
                speed    = { min = -1, max = 1 },
                bias     = { min = -1, max = 1 },
                organize = { min = 0, max = 1 },
            }),
            delay = interpolated(synth.FilteredFBComb(), {
                cutoff   = { min = hz(50), max = hz(3000) },
                duration = { min = ms(10), max = ms(1000) },
                gain     = { max = 1 },
            }),
            filter = synth.LPFilter(),
        }
    end

    local function patch(modules)
        local channel = modules.midi:ns('cc/1')
        local cc      = function(n) return channel:output(n) end

        local clock = with(modules.clock, function(c)
            c.osc:set      { pitch = hz(5) }
            c.multiple:set { input = c.osc:output('pulse') }
            return c.multiple
        end)

        with(modules.random, function(r)
            r.trigger:set {
                input   = clock:output(0),
                divisor = 16,
            }
            r.series:set {
                clock   = clock:output(1),
                trigger = r.trigger:output(),
                size    = 8,
            }
            r.quant:set { input = r.series:output('values') }

            -- Cmin Penatonic
            r.quant:set {
                { pitch = pitch('C3') },
                { pitch = pitch('Eb3') },
                { pitch = pitch('F3') },
                { pitch = pitch('G3') },
                { pitch = pitch('Bb3') },
                { pitch = pitch('C4') },
                { pitch = pitch('Eb4') },
                { pitch = pitch('F4') },
                { pitch = pitch('G4') },
                { pitch = pitch('Bb4') },
            }
        end)

        local voice = with(modules.voice, function(v)
            v.adsr:set { 
                gate    = modules.random.series:output('gate'),
                attack  = cc(45),
                decay   = cc(46),
                sustain = cc(47),
                release = cc(48),
            }
            v.osc:set  { pitch = modules.random.quant:output(), }
            v.mix:set {
                { input = v.osc:output('sine') },
                { input = v.osc:output('saw'), level = 0.1 },
            }
            v.amp:set { a = v.mix:output(), b = v.adsr:output() }
            return v.amp
        end)

        modules.delay:set {
            input    = voice:output(),
            gain     = cc(25),
            cutoff   = cc(26),
            duration = cc(27),
        }

        modules.tape:set { 
            input    = modules.delay:output() ,
            record   = cc(9),
            splice   = cc(10),
            unsplice = cc(11),
            speed    = cc(21),
            bias     = cc(22),
            organize = cc(23),
        }
        modules.filter:set { input = modules.tape:output(), cutoff = hz(3000) }

        return modules.filter:output()
    end

    return build, patch
end
