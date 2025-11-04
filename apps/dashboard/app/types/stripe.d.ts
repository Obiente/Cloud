// Stripe.js type declarations
declare global {
  interface Window {
    Stripe?: (publishableKey: string) => any;
  }
}

export {};

