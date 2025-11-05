/**
 * Content Security Policy middleware
 * Dynamically sets CSP headers based on runtime configuration
 * Reads API host from NUXT_PUBLIC_API_HOST environment variable
 */
export default defineEventHandler((event) => {
  const config = useRuntimeConfig(event);
  const apiHost = config.public.apiHost;
  const requestHost = config.public.requestHost;
  
  // Parse API host URL to extract protocol and host
  const allowedHosts = new Set<string>();
  
  const addHost = (host: string) => {
    try {
      const url = new URL(host);
      allowedHosts.add(`${url.protocol}//${url.host}`);
    } catch (e) {
      // If URL parsing fails, use as-is (might be a relative path)
      console.warn('[CSP] Failed to parse host URL:', host);
      allowedHosts.add(host);
    }
  };
  
  // Add API host
  addHost(apiHost);
  
  // Add request host if it's different from API host
  if (requestHost && requestHost !== apiHost) {
    addHost(requestHost);
  }
  
  // Build connect-src directive with allowed hosts
  const connectSrcParts = [
    "'self'",
    ...Array.from(allowedHosts),
    // Stripe endpoints
    'https://api.stripe.com',
    'https://m.stripe.network',
  ];
  
  // Build CSP directives
  const cspDirectives = [
    "default-src 'self'",
    "script-src 'self' 'unsafe-inline' 'unsafe-eval' https://js.stripe.com https://m.stripe.network",
    "style-src 'self' 'unsafe-inline' 'unsafe-hashes' https://m.stripe.network",
    "img-src 'self' data: https:",
    "font-src 'self' data:",
    `connect-src ${connectSrcParts.join(' ')}`,
    "frame-src https://js.stripe.com https://hooks.stripe.com",
    "frame-ancestors 'self'",
  ];
  
  // Set CSP header
  event.node.res.setHeader('Content-Security-Policy', cspDirectives.join('; '));
});

