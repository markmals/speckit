import Stripe from "stripe";

let stripe = new Stripe(process.env.STRIPE_SECRET_KEY ?? "");

/**
 * Server-only: creates a Stripe Checkout session (use `.url` to redirect the
 * browser). Set STRIPE_SECRET_KEY in the server environment. Call this from a
 * TanStack Start server function or route action — never from client code.
 */
export function createCheckoutSession(priceId: string, origin: string) {
    return stripe.checkout.sessions.create({
        mode: "subscription",
        line_items: [{ price: priceId, quantity: 1 }],
        success_url: `${origin}/?checkout=success`,
        cancel_url: `${origin}/?checkout=cancel`,
    });
}
