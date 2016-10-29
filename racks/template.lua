local pkg = {}

-- More information about the structure of this file: https://github.com/brettbuddin/eolian/wiki/Rack-Files

function pkg.build(self)
    local modules = {
        -- Rack composition
    }
    return {
        modules = modules,
        output = function()
            return 0
        end
    }
end

function pkg.patch(self, modules)
    -- Patch configuration
end

return pkg
