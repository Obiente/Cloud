/**
 * Utility functions for applying OUI theme to Monaco Editor
 * 
 * This module is client-only and should only be imported dynamically
 */

// Track if theme has been registered to avoid multiple registrations
let themeRegistered = false;

// Store colors cache to avoid repeated DOM queries
let colorsCache: Record<string, string> | null = null;

export function getOUIColors(): Record<string, string> {
  // Return cached colors if available
  if (colorsCache) {
    return colorsCache;
  }

  // Only run on client side - double check
  if (typeof window === "undefined" || typeof document === "undefined" || typeof getComputedStyle === "undefined") {
    return {};
  }

  try {
    const root = document.documentElement;
    // Ensure root is actually an Element before calling getComputedStyle
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
      editorBackground: getStyle("--oui-surface-base"),
      editorForeground: getStyle("--oui-text-primary"),
      inputBackground: getStyle("--oui-surface-base"),
      inputForeground: getStyle("--oui-text-primary"),
      inputBorder: getStyle("--oui-border-default"),
      dropdownBackground: getStyle("--oui-surface-overlay"),
      dropdownForeground: getStyle("--oui-text-primary"),
      dropdownBorder: getStyle("--oui-border-default"),
      listActiveSelectionBackground: getStyle("--oui-interactive-hover"),
      listActiveSelectionForeground: getStyle("--oui-text-primary"),
      listHoverBackground: getStyle("--oui-interactive-hover"),
      listHoverForeground: getStyle("--oui-text-primary"),
      buttonBackground: getStyle("--oui-surface-raised"),
      buttonForeground: getStyle("--oui-text-primary"),
      buttonHoverBackground: getStyle("--oui-interactive-hover"),
      scrollbarSliderBackground: getStyle("--oui-surface-muted"),
      scrollbarSliderHoverBackground: getStyle("--oui-border-strong"),
      scrollbarSliderActiveBackground: getStyle("--oui-border-strong"),
      selectionBackground: getStyle("--oui-accent-primary") + "40",
      selectionForeground: getStyle("--oui-text-primary"),
      lineHighlightBackground: getStyle("--oui-interactive-hover"),
      lineNumberActiveForeground: getStyle("--oui-text-secondary"),
      lineNumberForeground: getStyle("--oui-text-tertiary"),
      widgetShadow: "0 0 0 0",
      editorWidgetBackground: getStyle("--oui-surface-overlay"),
      editorWidgetForeground: getStyle("--oui-text-primary"),
      editorWidgetBorder: getStyle("--oui-border-default"),
      focusBorder: getStyle("--oui-interactive-focus"),
      borderDefault: getStyle("--oui-border-default"),
      accentPrimary: getStyle("--oui-accent-primary"),
      accentSecondary: getStyle("--oui-accent-secondary"),
      accentSuccess: getStyle("--oui-accent-success"),
      accentWarning: getStyle("--oui-accent-warning"),
      accentDanger: getStyle("--oui-accent-danger"),
      accentInfo: getStyle("--oui-accent-info"),
    };
    
    // Cache the colors
    colorsCache = result;
    return result;
  } catch (err) {
    // If anything fails, return empty object
    return {};
  }
}

export function registerOUITheme(monaco: any) {
  // Only register theme once globally to avoid issues
  if (themeRegistered) {
    return;
  }
  
  // Only run on client side
  if (typeof window === "undefined" || typeof document === "undefined") {
    return;
  }

  const colors = getOUIColors();
  
  // If we couldn't get colors, skip registration
  if (!colors || Object.keys(colors).length === 0) {
    return;
  }

  monaco.editor.defineTheme("oui-dark", {
    base: "vs-dark",
    inherit: true,
    rules: [
      // General tokens
      { token: "", foreground: colors.foreground, background: colors.background },
      // Comments
      { token: "comment", foreground: colors.lineNumberForeground, fontStyle: "italic" },
      // Strings
      { token: "string", foreground: colors.accentSuccess || colors.foreground },
      // Keywords
      { token: "keyword", foreground: colors.accentPrimary || colors.foreground, fontStyle: "bold" },
      // Numbers
      { token: "number", foreground: colors.accentWarning || colors.foreground },
      // Operators
      { token: "operator", foreground: colors.accentSecondary || colors.foreground },
      // Variables
      { token: "variable", foreground: colors.accentInfo || colors.foreground },
      // Functions
      { token: "function", foreground: colors.accentPrimary || colors.foreground },
    ],
    colors: {
      "editor.background": colors.editorBackground,
      "editor.foreground": colors.editorForeground,
      "editor.lineHighlightBackground": colors.lineHighlightBackground,
      "editor.selectionBackground": colors.selectionBackground,
      "editor.selectionHighlightBackground": colors.selectionBackground,
      "editorCursor.foreground": colors.accentPrimary || colors.foreground,
      "editorWhitespace.foreground": colors.lineNumberForeground,
      "editorIndentGuide.activeBackground": colors.lineNumberForeground,
      "editorIndentGuide.background": colors.lineNumberForeground,
      "editorLineNumber.foreground": colors.lineNumberForeground,
      "editorLineNumber.activeForeground": colors.lineNumberActiveForeground,
      "editorRuler.foreground": colors.borderDefault || colors.lineNumberForeground,
      "editorSuggestWidget.background": colors.editorWidgetBackground,
      "editorSuggestWidget.foreground": colors.editorWidgetForeground,
      "editorSuggestWidget.border": colors.editorWidgetBorder,
      "editorSuggestWidget.selectedBackground": colors.listActiveSelectionBackground,
      "editorHoverWidget.background": colors.editorWidgetBackground,
      "editorHoverWidget.foreground": colors.editorWidgetForeground,
      "editorHoverWidget.border": colors.editorWidgetBorder,
      "editorWidget.background": colors.editorWidgetBackground,
      "editorWidget.foreground": colors.editorWidgetForeground,
      "editorWidget.border": colors.editorWidgetBorder,
      "editorWidget.shadow": colors.widgetShadow,
      "input.background": colors.inputBackground,
      "input.foreground": colors.inputForeground,
      "input.border": colors.inputBorder,
      "inputOption.activeBorder": colors.focusBorder,
      "dropdown.background": colors.dropdownBackground,
      "dropdown.foreground": colors.dropdownForeground,
      "dropdown.border": colors.dropdownBorder,
      "list.activeSelectionBackground": colors.listActiveSelectionBackground,
      "list.activeSelectionForeground": colors.listActiveSelectionForeground,
      "list.hoverBackground": colors.listHoverBackground,
      "list.hoverForeground": colors.listHoverForeground,
      "button.background": colors.buttonBackground,
      "button.foreground": colors.buttonForeground,
      "button.hoverBackground": colors.buttonHoverBackground,
      "scrollbarSlider.background": colors.scrollbarSliderBackground,
      "scrollbarSlider.hoverBackground": colors.scrollbarSliderHoverBackground,
      "scrollbarSlider.activeBackground": colors.scrollbarSliderActiveBackground,
      "focusBorder": colors.focusBorder,
    },
  });
  
  themeRegistered = true;
}

