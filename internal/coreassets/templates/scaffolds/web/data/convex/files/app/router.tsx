import { QueryClientProvider } from "@tanstack/react-query";
import { createRouter } from "@tanstack/react-router";
import { setupRouterSsrQueryIntegration } from "@tanstack/react-router-ssr-query";
import { ConvexProvider } from "convex/react";

import { convex, queryClient } from "#/data/convex.ts";

import { routeTree } from "./routes.gen.ts";

export function getRouter() {
    let router = createRouter({
        routeTree,
        context: { queryClient },
        defaultPreload: "intent",
        scrollRestoration: true,
        Wrap: ({ children }) => (
            <ConvexProvider client={convex}>
                <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
            </ConvexProvider>
        ),
    });
    setupRouterSsrQueryIntegration({ router, queryClient });
    return router;
}

declare module "@tanstack/react-router" {
    interface Register {
        router: ReturnType<typeof getRouter>;
    }
}
