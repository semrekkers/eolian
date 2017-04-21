local string = require('eolian.string')

return function(rack, name, fn, opts)
    local parts = string.split(name, '.')
    local path  = rack

    for i,v in ipairs(parts) do
        if path[v] ~= nil then
            if i == #parts then
                break
            end
            if type(v) == 'table' then
                path = path[v]
            end
        else
            if i == #parts then
                path[v] = fn(opts or {})
            else
                path[v] = {}
            end
            path = path[v]
        end
    end
end
