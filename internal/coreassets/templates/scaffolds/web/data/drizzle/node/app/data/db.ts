import { drizzle } from "drizzle-orm/node-sqlite";

import * as schema from "#db/schema.ts";

// Node's built-in node:sqlite — a local file in dev (override with DATABASE_URL).
const DATABASE_URL = process.env.DATABASE_URL ?? "./.data/app.db";

export const db = drizzle(DATABASE_URL, { schema });
