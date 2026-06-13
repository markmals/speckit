import { loadStripe } from "@stripe/stripe-js";

/**
 * Client-side Stripe.js loader. Set VITE_STRIPE_PUBLISHABLE_KEY. Await it before
 * calling Stripe.js, e.g. `(await stripePromise)?.redirectToCheckout({ … })`.
 */
export const stripePromise = loadStripe(import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY as string);
