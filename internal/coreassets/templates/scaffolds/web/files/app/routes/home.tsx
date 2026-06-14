import { IconRocket } from "@tabler/icons-react";
import { createFileRoute } from "@tanstack/react-router";
import { motion } from "motion/react";

import { greeting } from "@/app/lib/greeting.ts";
import { Button } from "@/components/ui/button";

export const Route = createFileRoute("/")({
    component: Home,
});

function Home() {
    return (
        <motion.main
            animate={{ opacity: 1, y: 0 }}
            className="mx-auto flex max-w-2xl flex-col items-center gap-6 p-12"
            initial={{ opacity: 0, y: 8 }}
        >
            <h1 className="text-3xl font-bold">{greeting("world")}</h1>
            <p className="text-neutral-600">A TanStack Start app scaffolded by SpecKit.</p>
            <Button onPress={() => globalThis.alert("Hello from React Aria!")}>
                <IconRocket aria-hidden className="size-4" /> Get started
            </Button>
        </motion.main>
    );
}
