import { cloudflare } from "@cloudflare/vite-plugin";
import babel from "@rolldown/plugin-babel";
import tailwindcss from "@tailwindcss/vite";
import { tanstackStart } from "@tanstack/react-start/plugin/vite";
import react, { reactCompilerPreset } from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import devtoolsJson from "vite-plugin-devtools-json";

export default defineConfig({
    server: { port: 1612 },
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
