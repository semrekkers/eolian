local pkg = {}

-- Uses the OSC mapping provided by TouchOSC's Simple layout

function buildVoice(idx, control, envelope, pitch)
    local modules = {
        osc  = synth.Osc(),
        adsr = synth.ADSR(),
        amp  = synth.BinaryMultiply(),
    }
    with(modules, function(t)
        t.osc:set { pitch = pitch }
        t.adsr:set {
            gate           = control:output('/2/push' .. tostring(idx+1)),
            attack         = envelope.attack:output(idx),
            decay          = envelope.decay:output(idx),
            sustain        = envelope.sustain:output(idx),
            release        = envelope.release:output(idx),
            disableSustain = envelope.disableSustain:output(idx),
        }
        t.amp:set {
            a = t.osc:output('saw'),
            b = t.adsr:output()
        }
    end)
    return {
        output = modules.amp:output()
    }
end

function pkg.build(self)
    local pressButtons = 16

    local modules = {
        control = synth.OSCServer{
            port = 8000,
            addresses = {
                -- Page 1
                { path = '/1/fader1', interp = 'ms', max = 5000, min = 50 },
                { path = '/1/fader2', interp = 'ms', max = 5000, min = 50 },
                { path = '/1/fader3' },
                { path = '/1/fader4', interp = 'ms', max = 5000, min = 50 },
                { path = '/1/fader5' },

                { path = '/1/toggle1' },
                { path = '/1/toggle2' },
                { path = '/1/toggle3' },
                { path = '/1/toggle4' },

                -- Page 2
                { path = '/2/push1', interp = 'gate' },
                { path = '/2/push2', interp = 'gate' },
                { path = '/2/push3', interp = 'gate' },
                { path = '/2/push4', interp = 'gate' },
                { path = '/2/push5', interp = 'gate' },
                { path = '/2/push6', interp = 'gate' },
                { path = '/2/push7', interp = 'gate' },
                { path = '/2/push8', interp = 'gate' },
                { path = '/2/push9', interp = 'gate' },
                { path = '/2/push10', interp = 'gate' },
                { path = '/2/push11', interp = 'gate' },
                { path = '/2/push12', interp = 'gate' },
                { path = '/2/push13', interp = 'gate' },
                { path = '/2/push14', interp = 'gate' },
                { path = '/2/push15', interp = 'gate' },
                { path = '/2/push16', interp = 'gate' },

                { path = '/2/toggle1' },
                { path = '/2/toggle2' },
                { path = '/2/toggle3' },
                { path = '/2/toggle4' },
            }
        },
        voice = {
            envelope = {
                attack         = synth.Multiple { size = pressButtons },
                decay          = synth.Multiple { size = pressButtons },
                sustain        = synth.Multiple { size = pressButtons },
                release        = synth.Multiple { size = pressButtons },
                disableSustain = synth.Multiple { size = pressButtons },
            },
            mix = synth.Mix { size = pressButtons }
        },
        mix = synth.Mix(),
    }

    local p  = theory.pitch('C2')
    local m2 = theory.minor(2)
    for i = 0, pressButtons-1, 1 do
        local voice = buildVoice(i, modules.control, modules.voice.envelope, p:value())
        modules.voice.mix:scope(i):set { input = voice.output }
        p = p:transpose(m2)
    end

    return {
        modules = modules,
        output = function()
            return modules.mix:output()
        end
    }
end

function pkg.patch(self, modules)
    with(modules.voice.envelope, function(t)
        t.attack:set         { input = modules.control:output('/1/fader1') }
        t.decay:set          { input = modules.control:output('/1/fader2') }
        t.sustain:set        { input = modules.control:output('/1/fader3') }
        t.release:set        { input = modules.control:output('/1/fader4') }
        t.disableSustain:set { input = modules.control:output('/1/toggle3') }
    end)
    modules.mix:set { master = 1 }
    modules.mix:scope(0):set {
        input = modules.voice.mix:output(),
        level = modules.control:output('/1/fader5')
    }
end

return pkg
