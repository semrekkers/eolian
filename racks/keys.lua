local pkg = {}

function pkg.build(self)
    local modules = {
        midi = synth.MIDIController {
            device = 2
        },

        lfo = {
            osc  = synth.Osc(),
            mult = synth.Multiple(),
        },

        voice = {
            adjust = synth.BinarySum(),
            mult   = synth.Multiple(),
            one = {
                osc = synth.Osc(),
            },
            two = {
                osc   = synth.Osc(),
                shift = synth.BinaryMultiply(),
            },
            three = {
                osc   = synth.Osc(),
                shift = synth.BinaryMultiply(),
            },
            mix    = synth.Mix(),
            filter = synth.LPFilter(),
        },

        envelope = {
            adsr   = synth.ADSR(),
            mult   = synth.Multiple(),
            interp = synth.Interpolate(),
        },
        amp      = synth.BinaryMultiply(),
        delay    = synth.FBComb(),
        compress = synth.Compress(),
        clip     = synth.Clip(),
    }
    return {
        modules = modules,
        output = function()
            return modules.clip:output()
        end
    }
end

function pkg.patch(self, modules)
    with(modules, function(m)
        with(m.lfo, function(l)
            l.osc:set  { pitch = hz(5), amp = 0.00001 }
            l.mult:set { input = l.osc:output('sine') }
        end)

        with(m.voice, function(v)
            v.adjust:set { a = m.midi:output('pitch'), b = m.lfo.mult:output(0) }
            v.mult:set   { input = v.adjust:output() }

            v.one.osc:set     { pitch = v.mult:output(0) }
            v.two.shift:set   { a = v.mult:output(1), b = 0.5 }
            v.two.osc:set     { pitch = v.two.shift:output() }
            v.three.shift:set { a = v.mult:output(2), b = 0.25 }
            v.three.osc:set   { pitch = v.three.shift:output() }

            v.mix:scope(0):set { input = v.one.osc:output('saw') }
            v.mix:scope(1):set { input = v.two.osc:output('saw') }
            v.mix:scope(2):set { input = v.three.osc:output('saw') }
            v.filter:set       {
                input     = v.mix:output(),
                cutoff    = m.envelope.interp:output(),
                resonance = 0.5
            }
        end)

        with(m.envelope, function(e)
            e.adsr:set {
                gate    = m.midi:output('gate'),
                attack  = ms(50),
                decay   = ms(50),
                sustain = 0.9,
                release = ms(1000),
            }
            e.mult:set   { input = e.adsr:output() }
            e.interp:set { input = e.mult:output(0), max = hz(5000), min = 0 }
        end)

        m.amp:set      { a = m.voice.filter:output(), b = m.envelope.mult:output(1) }
        m.delay:set    { input = m.amp:output(), gain = 0.3 }
        m.compress:set { input = m.delay:output() }
        m.clip:set     { input = m.compress:output(), max = 3 }
    end)
end

return pkg
