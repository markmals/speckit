import { ConvexQueryClient } from "@convex-dev/react-query";
import { QueryClient } from "@tanstack/react-query";
import { ConvexReactClient } from "convex/react";

// VITE_CONVEX_URL is written to .env.local by `convex dev` (an anonymous local
// deployment in dev). It's how the browser client reaches your Convex backend.
const CONVEX_URL = import.meta.env.VITE_CONVEX_URL as string;

export const convex = new ConvexReactClient(CONVEX_URL);
export const convexQueryClient = new ConvexQueryClient(convex);
export const queryClient = new QueryClient({
    defaultOptions: {
        queries: {
            queryFn: convexQueryClient.queryFn(),
            queryKeyHashFn: convexQueryClient.hashFn(),
        },
    },
});

convexQueryClient.connect(queryClient);
