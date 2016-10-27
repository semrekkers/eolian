local pkg = {}

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
