--- Puts the installed `specify` binary on PATH. The archive extracts the binary
--- at the install root, so PATH is the install path itself.
--- Docs: https://mise.jdx.dev/tool-plugin-development.html#envkeys-hook
--- @param ctx EnvKeysCtx
--- @return EnvKey[]
function PLUGIN:EnvKeys(ctx)
    return {
        { key = "PATH", value = ctx.path },
    }
end
