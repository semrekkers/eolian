return function(env)
    local synth = require 'eolian.synth'
    local polyphony = 4

    local function voice(midi, idx)
        local pitch = synth.Multiple()
        local high = {
            osc = synth.Osc(),
        }
        local low = {
            pitch = synth.Multiply(),
            osc   = synth.Osc(),
        }
        local mix  = synth.Mix()
        local adsr = synth.ADSR()
        local mult = synth.Multiply()

        pitch:set { input = midi:scope(idx):out('pitch') }

        high.osc:set  { pitch = pitch:out(0) }
        low.pitch:set { a = pitch:out(1), b = 0.5 }
        low.osc:set   { pitch = low.pitch:out() }

        mix:set {
            { input = high.osc:out('saw') },
            { input = low.osc:out('saw') },
        }

        adsr:set  {
            gate    = midi:scope(idx):out('gate'),
            attack  = ms(100),
            decay   = ms(50),
            sustain = 0.9,
            release = ms(3000),
        }
        mult:set { a = mix:out(), b = adsr:out() }

        return { 
            output = function() 
                return mult:out()
            end
        }
    end

    local function build()
        local midi = synth.MIDIController { 
            device    = "DEVICE_NAME",
            polyphony = polyphony,
        }

        local voices = {}
        for i = 0,polyphony-1 do
            table.insert(voices, i+1, voice(midi, i))
        end

        return {
            midi     = midi,
            voices   = voices,
            mix      = synth.Mix { size = polyphony },
            filter   = synth.Filter(),
            delay    = synth.FBComb(),
            compress = synth.Compress(),
            clip     = synth.Clip(),
        }
    end

    local function patch(modules)
        with(modules, function(m)
            for i = 0,polyphony-1 do
                m.mix:scope(i):set { input = m.voices[i+1]:out() }
            end

            m.filter:set   { input = m.mix:out(), cutoff = hz(5000) }
            m.delay:set    { input = m.filter:out('lowpass'), gain = 0.4 }
            m.compress:set { input = m.delay:out() }
            m.clip:set     { input = m.compress:out(), level = 3 }
        end)

        return modules.clip:out()
    end

    return build, patch
end