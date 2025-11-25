/**
 * Stripe Elements theme configuration matching Oui design system
 * Reads CSS variables from the Oui theme to style Stripe Payment Elements
 */

/**
 * Get a CSS variable value from the document
 */
function getCSSVariable(variable: string): string {
  if (typeof window === 'undefined') return '';
  return getComputedStyle(document.documentElement)
    .getPropertyValue(variable)
    .trim();
}

/**
 * Convert hex color to RGB values
 */
function hexToRgb(hex: string): { r: number; g: number; b: number } | null {
  // Remove # if present
  hex = hex.replace('#', '');
  
  // Handle 3-digit hex
  if (hex.length === 3) {
    hex = hex.split('').map(char => char + char).join('');
  }
  
  if (hex.length !== 6) return null;
  
  const r = parseInt(hex.substring(0, 2), 16);
  const g = parseInt(hex.substring(2, 4), 16);
  const b = parseInt(hex.substring(4, 6), 16);
  
  return { r, g, b };
}

/**
 * Convert hex color to rgba string
 */
function hexToRgba(hex: string, alpha: number = 1): string {
  const rgb = hexToRgb(hex);
  if (!rgb) return `rgba(0, 0, 0, ${alpha})`;
  return `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, ${alpha})`;
}

/**
 * Get Stripe Elements appearance configuration matching Oui theme
 */
export function getStripeAppearance(): any {
  // Get Oui theme colors
  const surfaceBase = getCSSVariable('--oui-surface-base') || '#171521';
  const surfaceRaised = getCSSVariable('--oui-surface-raised') || '#1d1a29';
  const textPrimary = getCSSVariable('--oui-text-primary') || '#f5f3ff';
  const textSecondary = getCSSVariable('--oui-text-secondary') || '#c7b8ff';
  const textTertiary = getCSSVariable('--oui-text-tertiary') || '#9e8cff';
  const borderDefault = getCSSVariable('--oui-border-default') || '#3a2f5c';
  const borderStrong = getCSSVariable('--oui-border-strong') || '#5b4a8f';
  const accentPrimary = getCSSVariable('--oui-accent-primary') || '#8b5cf6';
  const accentDanger = getCSSVariable('--oui-accent-danger') || '#f43f5e';
  const interactiveHover = getCSSVariable('--oui-interactive-hover') || '#241d47';
  const interactiveFocus = getCSSVariable('--oui-interactive-focus') || '#d946ef';

  return {
    theme: 'night' as const, // Use night theme as base for dark mode
    variables: {
      colorPrimary: accentPrimary,
      colorBackground: surfaceBase,
      colorText: textPrimary,
      colorTextSecondary: textSecondary,
      colorTextPlaceholder: textTertiary,
      colorDanger: accentDanger,
      fontFamily: 'system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
      fontSizeBase: '16px',
      spacingUnit: '4px',
      borderRadius: '6px',
    },
    rules: {
      '.Input': {
        backgroundColor: surfaceBase,
        borderColor: borderDefault,
        borderWidth: '1px',
        borderRadius: '6px',
        color: textPrimary,
        fontSize: '16px',
        padding: '12px',
        transition: 'border-color 0.2s ease, box-shadow 0.2s ease',
        fontFamily: 'system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
        fontSmoothing: 'antialiased',
      },
      '.Input:hover': {
        borderColor: borderStrong,
      },
      '.Input:focus': {
        borderColor: accentPrimary,
        boxShadow: `0 0 0 2px ${hexToRgba(accentPrimary, 0.2)}`,
      },
      '.Input::placeholder': {
        color: textSecondary,
      },
      '.Input:disabled': {
        backgroundColor: getCSSVariable('--oui-interactive-disabled') || '#3f3a66',
        color: getCSSVariable('--oui-text-disabled') || '#6b6396',
        cursor: 'not-allowed',
        opacity: '0.6',
      },
      '.Input--invalid': {
        borderColor: accentDanger,
        color: accentDanger,
      },
      '.Input--invalid:focus': {
        borderColor: accentDanger,
        boxShadow: `0 0 0 2px ${hexToRgba(accentDanger, 0.2)}`,
      },
      '.Label': {
        color: textPrimary,
        fontSize: '14px',
        fontWeight: '500',
        marginBottom: '8px',
        fontFamily: 'system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
      },
      '.Tab': {
        backgroundColor: 'transparent',
        borderColor: borderDefault,
        color: textSecondary,
        borderRadius: '6px',
        padding: '8px 12px',
        transition: 'all 0.2s ease',
      },
      '.Tab:hover': {
        backgroundColor: interactiveHover,
        color: textPrimary,
      },
      '.Tab--selected': {
        backgroundColor: surfaceRaised,
        borderColor: accentPrimary,
        color: accentPrimary,
      },
      '.TabLabel': {
        color: 'inherit',
      },
      '.TabIcon': {
        color: 'inherit',
      },
      '.Block': {
        backgroundColor: surfaceRaised,
        borderColor: borderDefault,
        borderRadius: '6px',
      },
      '.Divider': {
        borderColor: borderDefault,
      },
    },
  };
}

