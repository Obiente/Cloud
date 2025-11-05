/**
 * Utility functions for applying OUI theme to highlight.js
 * 
 * This module is client-only and should only be imported dynamically
 */

// Store colors cache to avoid repeated DOM queries
let colorsCache: Record<string, string> | null = null;

export function getOUIHighlightColors(): Record<string, string> {
  // Return cached colors if available
  if (colorsCache) {
    return colorsCache;
  }

  // Only run on client side
  if (typeof window === "undefined" || typeof document === "undefined" || typeof getComputedStyle === "undefined") {
    return {};
  }

  try {
    const root = document.documentElement;
    if (!root || !(root instanceof Element)) {
      return {};
    }
    
    const getStyle = (prop: string): string => {
      try {
        const style = getComputedStyle(root);
        return style.getPropertyValue(prop).trim();
      } catch {
        return "";
      }
    };

    const result: Record<string, string> = {
      background: getStyle("--oui-surface-base"),
      foreground: getStyle("--oui-text-primary"),
      textSecondary: getStyle("--oui-text-secondary"),
      textTertiary: getStyle("--oui-text-tertiary"),
      accentPrimary: getStyle("--oui-accent-primary"),
      accentSecondary: getStyle("--oui-accent-secondary"),
      accentSuccess: getStyle("--oui-accent-success"),
      accentWarning: getStyle("--oui-accent-warning"),
      accentDanger: getStyle("--oui-accent-danger"),
      accentInfo: getStyle("--oui-accent-info"),
      borderDefault: getStyle("--oui-border-default"),
    };
    
    // Cache the colors
    colorsCache = result;
    return result;
  } catch (err) {
    return {};
  }
}

export function applyOUIThemeToHighlightJS() {
  // Only run on client side
  if (typeof window === "undefined" || typeof document === "undefined") {
    return;
  }

  const colors = getOUIHighlightColors();
  
  // If we couldn't get colors, skip
  if (!colors || Object.keys(colors).length === 0) {
    return;
  }

  // Create or update style tag with OUI-themed highlight.js styles
  const styleId = "highlight-js-oui-theme";
  let styleElement = document.getElementById(styleId) as HTMLStyleElement | null;
  
  if (!styleElement) {
    styleElement = document.createElement("style");
    styleElement.id = styleId;
    document.head.appendChild(styleElement);
  }

  // Generate CSS using OUI colors
  styleElement.textContent = `
    .hljs {
      display: block;
      overflow-x: auto;
      padding: 0;
      background: ${colors.background || "#121022"};
      color: ${colors.foreground || "#f5f3ff"};
    }

    .hljs-comment,
    .hljs-quote {
      color: ${colors.textTertiary || "#9e8cff"};
      font-style: italic;
    }

    .hljs-keyword,
    .hljs-selector-tag,
    .hljs-subst {
      color: ${colors.accentPrimary || "#a855f7"};
      font-weight: bold;
    }

    .hljs-number,
    .hljs-literal,
    .hljs-variable,
    .hljs-template-variable,
    .hljs-tag .hljs-attr {
      color: ${colors.accentWarning || "#f59e0b"};
    }

    .hljs-string,
    .hljs-doctag {
      color: ${colors.accentSuccess || "#22c55e"};
    }

    .hljs-title,
    .hljs-section,
    .hljs-selector-id {
      color: ${colors.accentSecondary || "#22d3ee"};
      font-weight: bold;
    }

    .hljs-type,
    .hljs-class .hljs-title {
      color: ${colors.accentInfo || "#60a5fa"};
    }

    .hljs-tag,
    .hljs-name,
    .hljs-attribute {
      color: ${colors.accentPrimary || "#a855f7"};
      font-weight: normal;
    }

    .hljs-regexp,
    .hljs-link {
      color: ${colors.accentSuccess || "#22c55e"};
    }

    .hljs-symbol,
    .hljs-bullet {
      color: ${colors.accentDanger || "#f43f5e"};
    }

    .hljs-built_in,
    .hljs-builtin-name {
      color: ${colors.accentSecondary || "#22d3ee"};
    }

    .hljs-meta {
      color: ${colors.textTertiary || "#9e8cff"};
    }

    .hljs-deletion {
      background: ${colors.accentDanger || "#f43f5e"}40;
    }

    .hljs-addition {
      background: ${colors.accentSuccess || "#22c55e"}40;
    }

    .hljs-emphasis {
      font-style: italic;
    }

    .hljs-strong {
      font-weight: bold;
    }
  `;
  
  // Clear cache so colors refresh on next call
  colorsCache = null;
}

