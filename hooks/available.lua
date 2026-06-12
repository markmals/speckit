--- Lists installable SpecKit versions from GitHub releases.
--- Docs: https://mise.jdx.dev/tool-plugin-development.html#available-hook
--- @param ctx AvailableCtx
--- @return AvailableVersion[]
function PLUGIN:Available(ctx) -- luacheck: ignore 212
    local http = require("http")
    local json = require("json")

    local resp, err = http.get({
        url = "https://api.github.com/repos/markmals/speckit/releases?per_page=100",
        headers = {
            ["Accept"] = "application/vnd.github+json",
            ["User-Agent"] = "mise-specify",
        },
    })
    if err ~= nil then
        error("Failed to fetch SpecKit releases: " .. err)
    end
    if resp.status_code ~= 200 then
        error("GitHub API returned status " .. resp.status_code .. " listing SpecKit releases")
    end

    local releases = json.decode(resp.body)
    local result = {}
    for _, rel in ipairs(releases) do
        if not rel.draft then
            local version = (rel.tag_name or ""):gsub("^v", "")
            if version ~= "" then
                table.insert(result, { version = version })
            end
        end
    end
    -- GitHub returns releases newest-first, which is the order mise expects.
    return result
end
