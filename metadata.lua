-- metadata.lua — mise tool-plugin metadata for `specify` (SpecKit's CLI).
-- Referenced from a project's mise.toml as:
--   [plugins]
--   specify = "https://github.com/markmals/speckit"
--   [tools]
--   specify = "latest"
-- Docs: https://mise.jdx.dev/tool-plugin-development.html#metadata-lua

PLUGIN = { -- luacheck: ignore
    name = "specify",
    version = "1.0.0",
    description = "SpecKit — spec-driven development engine for native multiplatform apps",
    author = "markmals",
    updateUrl = "https://github.com/markmals/speckit",
    minRuntimeVersion = "0.2.0",
}
