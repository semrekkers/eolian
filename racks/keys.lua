local pkg = {}

function voice(midi, idx)
    local pitch = synth.Multiple()
    local high = {
        osc = synth.Osc(),
    }
    local low = {
        pitch = synth.BinaryMultiply(),
        osc   = synth.Osc(),
    }
    local mix = synth.Mix()

    local adsr  = synth.ADSR()
    local mult  = synth.BinaryMultiply()

    pitch:set { input = midi:scope(idx):output('pitch') }

    high.osc:set  { pitch = pitch:output(0) }
    low.pitch:set { a = pitch:output(1), b = 0.25 }
    low.osc:set   { pitch = low.pitch:output() }

    mix:scope(0):set { input = high.osc:output('pulse') }
    mix:scope(1):set { input = low.osc:output('saw') }

    adsr:set  {
        gate    = midi:scope(idx):output('gate'),
        attack  = ms(50),
        decay   = ms(50),
        sustain = 0.5,
        release = ms(1000),
    }
    mult:set { a = mix:output(), b = adsr:output() }

    return { 
        output = function() 
            return mult:output()
        end
    }
end

function pkg.build(self)
    local midi = synth.MIDIController { device  = 2 }
    local modules = {
        voices = {
            voice(midi, 0),
            voice(midi, 1),
            voice(midi, 2),
            voice(midi, 3),
            voice(midi, 4),
            voice(midi, 5),
        },
        midi     = midi,
        mix      = synth.Mix { size = 6 },
        filter   = synth.LPFilter(),
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
        m.mix:scope(0):set { input = m.voices[1]:output() }
        m.mix:scope(1):set { input = m.voices[2]:output() }
        m.mix:scope(2):set { input = m.voices[3]:output() }
        m.mix:scope(3):set { input = m.voices[4]:output() }
        m.mix:scope(4):set { input = m.voices[5]:output() }
        m.mix:scope(5):set { input = m.voices[6]:output() }

        m.filter:set   { input = m.mix:output(), cutoff = hz(2000) }
        m.delay:set    { input = m.filter:output(), gain = 0.3 }
        m.compress:set { input = m.delay:output() }
        m.clip:set     { input = m.compress:output(), max = 3 }
    end)
end

return pkg
