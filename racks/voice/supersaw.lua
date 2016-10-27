return function(count)
    local pitch  = synth.Multiple { size = count }
    local spread = synth.Multiple { size = count }

    local mix = synth.Mix { size = count }
    for i = 0, count-1, 1 do
        -- Establish scaling factor for this oscillator
        local scale = synth.Direct()
        scale:set { input = i }

        -- Create detune amount
        local detune = synth.BinaryMultiply()
        detune:set {
            a = spread:output(tostring(i)),
            b = scale:output()
        }

        -- Mix the oscillator
        local osc = synth.Osc()
        osc:set {
            pitch = pitch:output(tostring(i)),
            detune = detune:output(),
        }
        mix:scope(i):set { input = osc:output('saw') }
    end

    return { 
        set = function(_, inputs)
            if inputs.pitch ~= nil then
                pitch:set { input = inputs.pitch }
            end
            if inputs.spread ~= nil then
                spread:set { input = inputs.spread }
            end
        end,
        output = mix:output()
    }
end
