import { cloudflare } from "@cloudflare/vite-plugin";
import babel from "@rolldown/plugin-babel";
import tailwindcss from "@tailwindcss/vite";
import { tanstackStart } from "@tanstack/react-start/plugin/vite";
import react, { reactCompilerPreset } from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import devtoolsJson from "vite-plugin-devtools-json";

export default defineConfig({
    server: { port: 1612 },
    // Resolve the `@/*` aliases from tsconfig (Vite 8 native) so rac-ui's
    // shadcn-native imports (`@/components/ui/*`, `@/lib/cva`) and the app's
    // `@/app/*` imports both resolve without a manual alias map.
    resolve: { tsconfigPaths: true },
    plugins: [
        devtoolsJson(),
        cloudflare({ viteEnvironment: { name: "ssr" } }),
        tanstackStart({
            srcDirectory: "app",
            router: {
                routesDirectory: ".",
                virtualRouteConfig: "app/routes.ts",
                generatedRouteTree: "routes.gen.ts",
            },
        }),
        react(),
        babel({ presets: [reactCompilerPreset()] }),
        tailwindcss(),
    ],
});
