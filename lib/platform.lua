--- Maps mise's RUNTIME to SpecKit's goreleaser release assets.
--- Assets are named: specify_<version>_<os>_<arch>.<ext>
---   os   ∈ darwin | linux | windows
---   arch ∈ amd64 | arm64
---   ext  = zip on windows, tar.gz otherwise
--- The archive holds the `specify` binary at its root.

local M = {}

local function os_token()
    local t = RUNTIME.osType
    if t == "darwin" or t == "linux" or t == "windows" then
        return t
    end
    error("Unsupported operating system: " .. tostring(t))
end

local function arch_token()
    local a = RUNTIME.archType
    if a == "amd64" or a == "arm64" then
        return a
    end
    error("Unsupported architecture: " .. tostring(a))
end

--- The GitHub release download URL for a given version (no leading "v").
--- @param version string e.g. "0.1.0"
--- @return string url
function M.asset_url(version)
    local os = os_token()
    local arch = arch_token()
    local ext = (os == "windows") and "zip" or "tar.gz"
    return string.format(
        "https://github.com/markmals/speckit/releases/download/v%s/specify_%s_%s_%s.%s",
        version,
        version,
        os,
        arch,
        ext
    )
end

--- @return string "specify.exe" on Windows, "specify" otherwise
function M.binary_name()
    if RUNTIME.osType == "windows" then
        return "specify.exe"
    end
    return "specify"
end

return M
