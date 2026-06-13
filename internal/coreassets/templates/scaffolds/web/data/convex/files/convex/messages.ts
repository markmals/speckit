import { query } from "./_generated/server.js";

export const list = query({
    args: {},
    handler: async ctx => await ctx.db.query("messages").collect(),
});
