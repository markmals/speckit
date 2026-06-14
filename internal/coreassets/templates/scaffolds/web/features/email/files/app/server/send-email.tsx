import { Resend } from "resend";

import { WelcomeEmail } from "@/app/emails/welcome.tsx";

let resend = new Resend(process.env.RESEND_API_KEY);

/**
 * Server-only: renders the React Email template and sends it via Resend. Set
 * RESEND_API_KEY in the server environment. Call this from a TanStack Start
 * server function or a route action — never import it into client code.
 */
export function sendWelcomeEmail(to: string, name: string) {
    return resend.emails.send({
        from: "Acme <onboarding@resend.dev>",
        to: [to],
        subject: "Welcome!",
        react: <WelcomeEmail loginUrl="https://example.com/login" name={name} />,
    });
}
