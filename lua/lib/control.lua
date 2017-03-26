return function(m, options, defaultInput)
    options = options or {}
    defaultInput = defaultInput or 'input'

    if type(m) == 'function' then
        m = m()
    end

    local synth  = require('eolian.synth')
    local proxy  = require('eolian.synth.proxy')
    local eolianString = require('eolian.string')

    local controls = {}
    for name,_ in pairs(m.inputs()) do
        if options[name] ~= nil then
            controls[name] = synth.Control(options[name])
            set(m, name, out(controls[name]))
        end
    end

    return {
        id = function()
            return string.format("Controlled[%s]", m.id())
        end,
        members = function()
            local m = { m.id() }
            for _,c in pairs(controls) do
                table.insert(m, c.id())
            end
            return m
        end,
        inputs = function()
            local t = {}
            for k,v in pairs(m.inputs()) do
                if controls[k] ~= nil then
                    t[k] = controls[k].inputs()[defaultInput]
                else
                    t[k] = v
                end
            end
            return t
        end,
        outputs = m.outputs,
        set = function(_, arg1, arg2)
            local apply = function(k,v)
                local segs = eolianString.split(k, '/')
                if #segs == 2 and controls[segs[1]] ~= nil then
                    set(controls[segs[1]], segs[2], v)
                elseif controls[k] ~= nil then
                    set(controls[k], defaultInput, v)
                else
                    set(m, k, v)
                end
            end

            if type(arg1) == 'table' then
                for k,v in pairs(arg1) do
                    apply(k,v)
                end
            else
                apply(arg1, arg2)
            end
        end,
        out = proxy.outputs(m),
        close = m.close,
        reset = function()
            for k,v in pairs(m.inputs()) do
                if controls[k] ~= nil then
                    controls[k].reset()
                else
                    m.resetOnly({k})
                end
            end
        end
    }
end
