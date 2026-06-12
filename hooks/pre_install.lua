--- Returns the download URL for a specific SpecKit version. mise downloads and
--- extracts the archive automatically; the `specify` binary sits at its root.
--- Docs: https://mise.jdx.dev/tool-plugin-development.html#preinstall-hook
--- @param ctx PreInstallCtx
--- @return PreInstallResult
function PLUGIN:PreInstall(ctx)
    local platform = require("platform")
    return {
        version = ctx.version,
        url = platform.asset_url(ctx.version),
    }
end
