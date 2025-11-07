import { ref, computed, watch, onMounted, onBeforeUnmount } from "vue";

/**
 * Tailwind breakpoint values (in pixels)
 * These match Tailwind's default breakpoints
 */
const TAILWIND_BREAKPOINTS = {
  sm: 640,   // 40rem
  md: 768,   // 48rem
  lg: 1024,  // 64rem
  xl: 1280,  // 80rem
  "2xl": 1536, // 96rem
} as const;

export type TailwindBreakpoint = keyof typeof TAILWIND_BREAKPOINTS;

export interface UseSidebarOptions {
  /**
   * Tailwind breakpoint at which sidebar becomes desktop (default: 'xl' / 1280px)
   * Must match the breakpoint used in Tailwind classes (e.g., xl:hidden, xl:block)
   */
  desktopBreakpoint?: TailwindBreakpoint;
  
  /**
   * Whether to lock body scroll when sidebar is open
   */
  lockBodyScroll?: boolean;
  
  /**
   * Whether to trap focus within sidebar when open
   */
  trapFocus?: boolean;
}

const DEFAULT_OPTIONS: Required<UseSidebarOptions> = {
  desktopBreakpoint: "xl",
  lockBodyScroll: true,
  trapFocus: true,
};

/**
 * Composable for managing sidebar state with robust mobile handling
 * Uses Tailwind breakpoints for consistency with CSS classes
 */
export function useSidebar(options: UseSidebarOptions = {}) {
  const opts = { ...DEFAULT_OPTIONS, ...options };
  
  const isOpen = ref(false);
  const isDesktop = ref(false);
  const sidebarId = "mobile-primary-navigation";
  
  // Get pixel value for the breakpoint
  const breakpointPx = TAILWIND_BREAKPOINTS[opts.desktopBreakpoint];
  
  // Check if we're on desktop
  const checkBreakpoint = () => {
    if (import.meta.client) {
      const matches = window.matchMedia(`(min-width: ${breakpointPx}px)`).matches;
      isDesktop.value = matches;
      
      // Auto-close sidebar when switching to desktop
      if (matches && isOpen.value) {
        isOpen.value = false;
      }
    }
  };
  
  // Lock/unlock body scroll
  const lockBodyScroll = () => {
    if (!import.meta.client || !opts.lockBodyScroll) return;
    
    if (isOpen.value && !isDesktop.value) {
      // Store original overflow
      const originalOverflow = document.body.style.overflow;
      document.body.style.overflow = "hidden";
      
      return () => {
        document.body.style.overflow = originalOverflow;
      };
    }
  };
  
  // Focus management
  let previousActiveElement: HTMLElement | null = null;
  let focusableElements: HTMLElement[] = [];
  
  const trapFocus = () => {
    if (!import.meta.client || !opts.trapFocus || isDesktop.value) return;
    
    const sidebar = document.getElementById(sidebarId);
    if (!sidebar) return;
    
    // Store previous focus
    previousActiveElement = document.activeElement as HTMLElement;
    
    // Get all focusable elements
    focusableElements = Array.from(
      sidebar.querySelectorAll<HTMLElement>(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      )
    ).filter(el => !el.hasAttribute('disabled') && !el.hasAttribute('aria-hidden'));
    
    // Focus first element
    if (focusableElements.length > 0) {
      focusableElements[0]?.focus();
    }
    
    // Handle keyboard navigation
    const handleKeydown = (e: KeyboardEvent) => {
      if (!isOpen.value || isDesktop.value) return;
      
      if (e.key === "Tab") {
        const currentIndex = focusableElements.indexOf(e.target as HTMLElement);
        
        if (e.shiftKey) {
          // Shift + Tab: go to previous
          if (currentIndex <= 0) {
            e.preventDefault();
            focusableElements[focusableElements.length - 1]?.focus();
          }
        } else {
          // Tab: go to next
          if (currentIndex >= focusableElements.length - 1) {
            e.preventDefault();
            focusableElements[0]?.focus();
          }
        }
      }
    };
    
    document.addEventListener("keydown", handleKeydown);
    
    return () => {
      document.removeEventListener("keydown", handleKeydown);
      // Restore focus
      previousActiveElement?.focus();
    };
  };
  
  // Open sidebar
  const open = () => {
    isOpen.value = true;
  };
  
  // Close sidebar
  const close = () => {
    isOpen.value = false;
  };
  
  // Toggle sidebar
  const toggle = () => {
    isOpen.value = !isOpen.value;
  };
  
  // Handle escape key
  const handleEscape = (e: KeyboardEvent) => {
    if (e.key === "Escape" && isOpen.value && !isDesktop.value) {
      close();
    }
  };
  
  // Watch for sidebar state changes
  let unlockScroll: (() => void) | null = null;
  let untrapFocus: (() => void) | null = null;
  
  watch(isOpen, (open) => {
    if (import.meta.client) {
      // Cleanup previous locks
      unlockScroll?.();
      untrapFocus?.();
      unlockScroll = null;
      untrapFocus = null;
      
      if (open && !isDesktop.value) {
        const scrollUnlock = lockBodyScroll();
        const focusUntrap = trapFocus();
        unlockScroll = scrollUnlock || null;
        untrapFocus = focusUntrap || null;
      }
    }
  }, { immediate: true });
  
  // Setup on mount
  onMounted(() => {
    if (import.meta.client) {
      checkBreakpoint();
      
      // Listen for resize events using Tailwind breakpoint
      const mediaQuery = window.matchMedia(`(min-width: ${breakpointPx}px)`);
      const handleChange = () => {
        checkBreakpoint();
      };
      
      // Modern browsers
      if (mediaQuery.addEventListener) {
        mediaQuery.addEventListener("change", handleChange);
      } else {
        // Fallback for older browsers
        mediaQuery.addListener(handleChange);
      }
      
      // Listen for escape key
      window.addEventListener("keydown", handleEscape);
      
      // Cleanup function
      return () => {
        if (mediaQuery.removeEventListener) {
          mediaQuery.removeEventListener("change", handleChange);
        } else {
          mediaQuery.removeListener(handleChange);
        }
        window.removeEventListener("keydown", handleEscape);
        unlockScroll?.();
        untrapFocus?.();
      };
    }
  });
  
  onBeforeUnmount(() => {
    unlockScroll?.();
    untrapFocus?.();
  });
  
  return {
    isOpen: computed(() => isOpen.value),
    isDesktop: computed(() => isDesktop.value),
    sidebarId,
    open,
    close,
    toggle,
  };
}

