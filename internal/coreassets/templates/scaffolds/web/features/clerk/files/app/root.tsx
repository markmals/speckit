import type { QueryClient } from "@tanstack/react-query";

import { ClerkProvider } from "@clerk/tanstack-react-start";
import { createRootRouteWithContext, HeadContent, Outlet, Scripts } from "@tanstack/react-router";

import appCss from "#/styles/tailwind.css?url";

export const Route = createRootRouteWithContext<{ queryClient: QueryClient }>()({
    head: () => ({
        meta: [
            { charSet: "utf-8" },
            { content: "width=device-width, initial-scale=1", name: "viewport" },
            { title: "web" },
        ],
        links: [{ href: appCss, rel: "stylesheet" }],
    }),
    component: RootComponent,
});

function RootComponent() {
    return (
        <RootDocument>
            <Outlet />
        </RootDocument>
    );
}

function RootDocument({ children }: { children: React.ReactNode }) {
    return (
        <ClerkProvider>
            <html lang="en">
                <head>
                    <HeadContent />
                </head>
                <body>
                    {children}
                    <Scripts />
                </body>
            </html>
        </ClerkProvider>
    );
}
