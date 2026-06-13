import { index, rootRoute } from "@tanstack/virtual-file-routes";

export const routes = rootRoute("root.tsx", [index("routes/home.tsx")]);
