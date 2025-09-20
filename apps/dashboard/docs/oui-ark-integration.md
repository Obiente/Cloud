# OUI + Ark UI Integration Summary

## Final Architecture Decision

After discussion and iteration, we've settled on a clean, minimal approach:

### 1. **Composition Strategy**
- **Minimal Compositions**: Only truly reusable base patterns
- **Direct Utility Application**: Sizes, variants, and states applied directly in Vue components
- **No Over-abstraction**: Avoid unnecessary composition layers

### 2. **Component Categories**

#### **Simple Components (OUI Styled)**
Use standard HTML elements with OUI utility classes:
- `Button.vue` - Standard button with OUI theming
- `Card.vue` - Layout container with OUI styling
- `CardHeader.vue`, `CardBody.vue`, `CardFooter.vue` - Card parts

#### **Complex Components (Ark UI + OUI)**
Use Ark UI for behavior with OUI styling:
- `Select.vue` - Dropdown selection with complex keyboard navigation
- `Tooltip.vue` - Contextual overlays with positioning
- `Dialog.vue` - Modal overlays with focus management
- `Toast.vue` - Notification system

### 3. **CSS Organization**

```
apps/dashboard/assets/css/tailwind/compositions/
├── button.css     # Only .btn-base
├── card.css       # Only .card-base + part layouts + status styles
└── (future components as needed)
```

**Composition Rules:**
- Only create compositions for patterns used 3+ times across different components
- Keep compositions as minimal base patterns
- Apply variants/sizes/states directly in Vue components

### 4. **Naming Conventions**

#### **OUI Utility Classes:**
```css
/* Correct */
.px-4
.text-lg  
.rounded-lg
.shadow-md

/* Incorrect */
.rounded-lg
.px-4
```

#### **OUI Color Variables:**
```css
/* Surfaces */
bg-surface-base, bg-surface-raised, bg-surface-overlay

/* Text */
text-primary, text-secondary, text-muted

/* Borders */
border-default, border-muted, border-strong

/* Status Colors */
bg-success, text-success, bg-danger, text-danger, etc.

/* Accents */
bg-accent-primary, text-accent-primary, bg-accent-secondary
```

### 5. **Implementation Examples**

#### **Simple Component (Card.vue):**
```vue
<template>
  <div
    :class="[
      // Base composition (reusable)
      'card-base',
      // Variants applied directly
      {
        'shadow-md bg-surface-raised border-muted': variant === 'raised',
        'shadow-lg bg-surface-overlay border-strong': variant === 'overlay',
        'bg-transparent border-default': variant === 'outline',
        'hover:-translate-y-0.5 hover:shadow-lg transition-all': hoverable,
      }
    ]"
  >
    <slot />
  </div>
</template>
```

#### **Complex Component (Select.vue):**
```vue
<template>
  <Select.Root :collection="collection" v-model="modelValue">
    <Select.Trigger
      class="
        btn-base
        w-full justify-between
        bg-surface-base border border-default text-primary
        hover:border-strong focus:border-primary focus:focus-ring
        px-3 py-2 text-sm
      "
    >
      <!-- Ark UI provides behavior, OUI provides styling -->
    </Select.Trigger>
  </Select.Root>
</template>
```

### 6. **Benefits of This Approach**

1. **Clear Separation**: Simple vs complex component needs
2. **Minimal Abstractions**: Only compose when truly reusable
3. **Maintainable**: Direct utility application is easy to understand and modify
4. **Consistent**: OUI theming throughout
5. **Performant**: No unnecessary CSS compilation or complexity

### 7. **Migration Status**

**✅ Completed:**
- Card components (Card, CardHeader, CardBody, CardFooter)
- Button component with direct utility application
- Select component using Ark UI + OUI styling
- Dashboard page updated to use OUI theming
- CSS compositions minimized to essentials

**📋 Next Steps:**
- Create additional Ark UI components as needed (Tooltip, Dialog, Toast)
- Migrate other pages to use OUI components
- Add proper TypeScript interfaces for component props
- Set up component documentation/storybook

This approach gives us the best of both worlds: simple, maintainable components with OUI theming for basic needs, and powerful Ark UI behavior for complex interactions.