/**
 * Composes query parameters from an object into a query string.
 * Filters out undefined, null, and empty string values.
 * 
 * @param params - Object with query parameter key-value pairs
 * @returns Query string (without leading '?') or empty string if no valid params
 * 
 * @example
 * composeQueryParams({ status: '1', environment: '2' })
 * // Returns: 'status=1&environment=2'
 * 
 * @example
 * composeQueryParams({ status: '1', search: undefined, filter: '' })
 * // Returns: 'status=1'
 */
export function composeQueryParams(
  params: Record<string, string | number | undefined | null>
): string {
  const searchParams = new URLSearchParams();
  
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== '') {
      searchParams.append(key, String(value));
    }
  }
  
  return searchParams.toString();
}

/**
 * Composes a full URL with query parameters.
 * 
 * @param path - The base path (e.g., '/deployments')
 * @param params - Object with query parameter key-value pairs
 * @returns Full URL with query string
 * 
 * @example
 * composeQueryUrl('/deployments', { status: '1', environment: '2' })
 * // Returns: '/deployments?status=1&environment=2'
 */
export function composeQueryUrl(
  path: string,
  params: Record<string, string | number | undefined | null>
): string {
  const queryString = composeQueryParams(params);
  return queryString ? `${path}?${queryString}` : path;
}

