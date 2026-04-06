# Obiente Cloud — Design Language

This document defines the visual language, UX patterns, and design philosophy for the Obiente Cloud dashboard. It serves as the source of truth for ongoing UI work and ensures consistency across all views.

---

## 1. Philosophy

**Dense but breathable.** The dashboard is a professional tool — not a marketing page. Every pixel earns its place. We favor information density over whitespace-padding, but use deliberate visual hierarchy so the eye knows exactly where to land.

**Dark-first, accent-driven.** The UI lives on deep, near-black surfaces. Color is used surgically: accent purple for primary elements, semantic colors (green/yellow/red/blue) for status and data types. Gratuitous color is noise.

**Structure over decoration.** No gradients, no drop shadows, no rounded-everything softness. Visual hierarchy comes from border weight, font weight, spacing, and icon placement — not from heavy-handed styling.

**Monospace where it matters.** Technical values (IPs, ports, domains, container IDs, connection strings, file paths) always render in monospace. It signals "this is a copyable/actionable value" without needing a label.

---

## 2. Color System

### Theme: Dark Purple (default)

| Token | Hex | Usage |
|---|---|---|
| `--oui-background` | `#0b0a10` | Page background |
| `--oui-surface-base` | `#13111c` | Card backgrounds, sidebar |
| `--oui-surface-raised` | `#191724` | Elevated surfaces |
| `--oui-surface-overlay` | `#201e2d` | Modals, dropdowns |
| `--oui-surface-muted` | `#272339` | Subtle backgrounds, icon containers |
| `--oui-accent-primary` | `#8b5cf6` | Primary actions, brand accent (violet) |
| `--oui-accent-info` | `#6366f1` | Information, secondary data (indigo) |
| `--oui-accent-success` | `#22c55e` | Running, healthy, connected |
| `--oui-accent-warning` | `#f59e0b` | Degraded, building, attention needed |
| `--oui-accent-danger` | `#f43f5e` | Errors, stopped, destructive actions |

### Color Assignment by Data Type

Colors are not random — each metric/resource type consistently maps to a color across all views:

- **CPU** → `accent-primary` (violet) — the "brain" of compute
- **Memory** → `accent-info` (indigo/blue) — pools/stacks
- **Bandwidth / Network** → `accent-success` (green) — data flowing
- **Storage / Disk** → `accent-secondary` or `warning` — capacity
- **Cost / Billing** → `accent-success` (green) — money

This means a CPU icon is always violet, a memory icon is always blue, network is always green — across LiveMetrics, UsageStatistics, CostBreakdown, and overview cards. The user builds subconscious association.

---

## 3. Typography

- **Primary text**: `text-primary` (`#eae7f3`) — headings, values, prominent data
- **Secondary text**: `text-secondary` (`#958db8`) — body content, descriptions
- **Tertiary text**: `text-tertiary` (`#6a6189`) — labels, captions, metadata

### Size Scale

| Size | Usage |
|---|---|
| `2xl` | Hero numbers (total cost, big KPIs) |
| `xl` | Resource card values (CPU cores, memory) |
| `lg` | Metric values in grids |
| `sm` | Section headings, row values, body text |
| `xs` | Labels, captions, badges, metadata |

### Weight

- `semibold` — headings, values, anything the eye should find first
- `medium` — secondary emphasis, row data
- `regular` (default) — body text, descriptions

### Monospace Rule

Any value that a user might copy, type into a terminal, or reference technically gets `font-mono`:
- IP addresses, domains, ports (`:8080`)
- Container IDs (`a1b2c3d4e5f6`)
- Connection strings (`postgresql://...`)
- Environment variable keys (`DATABASE_URL`)
- Git branches, repo paths

---

## 4. Layout Patterns

### Quick Info Bar

The first thing inside any resource detail tab. A horizontal `OuiCard variant="outline"` containing:
- Left: Icon in a `h-8 w-8 rounded-lg bg-surface-muted` container + primary identifier (domain, IP, host:port) in monospace + subtitle line with type/region/ancillary info
- Right: Cluster of `OuiBadge variant="secondary" size="xs"` showing key specs (vCPU count, memory, cost)

This pattern replaces the old "Details" card with its boring key-value `divide-y` list. The user sees the most important info at a glance without scrolling.

### Resource Grid (2-col details)

Below the Quick Info Bar, a `OuiGrid :cols="{ sm: 1, lg: 2 }" gap="sm"` containing two `OuiCard variant="outline"` cards:
- Each card has an icon + heading row, then a `grid grid-cols-2 gap-3` of label/value pairs
- Labels are `size="xs" color="tertiary"`, values are `size="sm" weight="medium"`
- No `divide-y` separators — the grid whitespace provides enough visual separation

### Metric Cards (4-col)

For live metrics and resource stats: `OuiGrid :cols="{ sm: 2, md: 4 }" gap="sm"` of `OuiCard variant="outline"` cards. Each card:
- Icon + label row (icon color matches the data-type color mapping)
- Large value (`size="xl"` or `size="lg"`)
- Optional: mini progress bar, subtitle

### Connection Hub

For copyable connection info (SSH, database, game server):
- Grouped inside a `OuiCard variant="outline" status="success"` (green left border = "this is a live connection")
- Each copyable field in its own `rounded-lg border border-border-default` container
- Copy button is `opacity-0 group-hover:opacity-100` — appears on hover, not always visible
- Monospace values, break-all for long strings

### Timeline (Build History)

Vertical timeline for sequential events:
- Absolute vertical line on the left (`left-[15px] w-px bg-border-default`)
- Colored dot per entry (green=success, red=failed, amber=building with inner pulse animation)
- Compact card to the right with key data in a single inline row
- Error messages inline as colored divs, not nested cards

### Unified List (Env Vars, Routing Rules)

Single `OuiCard variant="outline"` with `divide-y` rows inside, instead of individual cards per item:
- Hover-reveal action buttons per row (`opacity-0 group-hover:opacity-100`)
- Footer row with count + primary action button
- Segmented toggle for view modes (e.g., List vs. raw `.env`)

---

## 5. Card Usage

### Variants

| Variant | When |
|---|---|
| `outline` | Default for all content cards. Thin border, transparent background. |
| `default` | Outer wrapper only (e.g., tab content wrapper). Rarely used inside content. |
| `raised` | Almost never. Reserved for special elevated elements. |

### Status Prop

Cards accept `status="success|warning|danger|info"` which adds a 4px colored left border:
- `success` — live/connected things (connection hubs, running services)
- `warning` — sleeping databases, degraded state, building
- `danger` — error states, failed builds
- `info` — informational callouts, connection instructions

### Anti-Pattern: Card-in-Card

**Never nest a Card inside another Card's body.** This was the single biggest visual problem in the old UI. Instead:
- Use `div` with `rounded-lg border border-border-default` for sub-groupings inside a card
- Use `divide-y` for list rows
- Use `grid` with `gap-px bg-border-default` + `bg-surface-base` cells for dense grid layouts (the "gap-px trick")

---

## 6. Icon Language

All icons are Heroicons v2 (outline variant, 24px). Sized contextually:

| Context | Size |
|---|---|
| Card heading icon | `h-3.5 w-3.5` |
| Quick Info Bar icon (in container) | `h-4 w-4` inside `h-8 w-8` rounded container |
| Inline action (copy, delete) | `h-3.5 w-3.5` |
| Empty state icon | `h-6 w-6` inside `h-12 w-12` rounded container |
| Button icon | `h-3.5 w-3.5` |

### Icon + Heading Pattern

Section headings inside cards always pair an icon with the heading text:
```
<OuiFlex align="center" gap="xs">
  <SomeIcon class="h-3.5 w-3.5 text-accent-primary" />
  <OuiText size="sm" weight="semibold">Section Title</OuiText>
</OuiFlex>
```

The icon color should match the card's purpose (green for connection, purple for compute, blue for data, etc.)

---

## 7. Interactive Patterns

### Hover-Reveal Actions

Action buttons that are always-visible create visual clutter. For non-primary actions (copy, delete, expand):
- Wrap the row in a `group` class
- Action buttons get `opacity-0 group-hover:opacity-100 transition-opacity`
- The user discovers actions by mousing over the specific row

### Copy Buttons

Replace `OuiButton variant="ghost"` with raw `<button>` elements for copy actions:
- `class="p-1 rounded text-tertiary hover:text-primary"` + hover-reveal
- Just the `ClipboardIcon`, no text label needed
- Toast notification on success

### Status Dots

Inline status indicators use a `span` with:
- `h-1.5 w-1.5 rounded-full` (or `h-2 w-2` for more prominent)
- Color class matching status: `bg-success`, `bg-danger`, `bg-warning`
- For "building/pending" states: add inner pulse animation (`animate-ping` on an inner span)

### Empty States

Centered layout with:
- Icon in `h-12 w-12 rounded-xl bg-surface-muted` container
- Title in `size="sm" weight="semibold"`
- Description in `size="xs" color="tertiary"`
- Optional action button below

---

## 8. Spacing & Density

### Gap Scale

| Gap | px | Usage |
|---|---|---|
| `xs` | 4px | Between label and value, icon and text |
| `sm` | 8px | Between items in a tight list, grid cells, cards in a grid |
| `md` | 12px | Between card sections, between cards in a stack |
| `lg` | 16px | Between major page sections (used sparingly) |

### General Rules

- Card grid gaps: always `gap="sm"` (8px)
- Stack gaps between cards: `gap="md"` (12px) — not `lg`
- Padding inside cards: default `OuiCardBody` padding (don't override with `p-0` + manual padding)
- Remove unnecessary wrapper `OuiStack gap="lg"` → use `gap="md"` for tighter feel

---

## 9. Badges

- `size="xs"` — default for inline metadata badges
- `variant="secondary"` — neutral specs (vCPU count, memory size, region)
- `variant="primary"` — highlighted info (cost, plan tier)
- `variant="success|warning|danger"` — status badges with matching dot

Status badges always include a dot:
```
<OuiBadge :variant="statusVariant" size="xs">
  <span class="inline-flex h-1.5 w-1.5 rounded-full mr-1" :class="dotClass" />
  {{ label }}
</OuiBadge>
```

---

## 10. Data Visualization

### Mini Progress Bars

For usage metrics inside cards, use thin bars (`h-1 rounded-full`) with the data-type color at reduced opacity (`bg-accent-primary/60`). These are visual accents, not precise gauges.

### CPU Threshold Bar

CPU usage gets special treatment with color thresholds:
- < 70%: green (`bg-accent-primary` or `bg-success`)
- 70–90%: yellow (`bg-warning`)
- > 90%: red (`bg-danger`)

Plus a subtle bottom glow line on the card for extra emphasis.

### Stacked Bar Chart

CostBreakdown uses a horizontal stacked bar (`h-2 rounded-full flex`) where each segment is proportionally sized via `width: X%`. Below it, a legend grid with color dots + label + value.

---

## 11. What We Avoid

- **Card-in-card nesting** — the #1 visual anti-pattern
- **`OuiAlert` for non-error content** — use Card with `status` prop instead
- **`OuiCode` blocks for single values** — use monospace text with hover-copy
- **Excessive `divide-y` key-value lists** — use 2-col grids instead
- **Text-only buttons for copy actions** — icon-only with hover reveal
- **`gap="lg"` or `gap="xl"` between cards** — too airy, use `md` or `sm`
- **Labels that restate the obvious** — if a badge says "2 vCPU", no need for a "CPU Cores" label
- **Full-word action labels on every button** — icons-only for dense rows, labels for primary actions

---

## 12. Component Hierarchy (Detail Pages)

```
ResourceHeader (title, badges, action buttons)
└── ResourceTabs
    └── Tab: Overview
        ├── Quick Info Bar (card with icon + primary ID + badge cluster)
        ├── 2-col Grid (details card + connection/network card)
        ├── LiveMetrics (4-col metric cards — shared component)
        ├── UsageStatistics (4-col usage mini-cards with bars)
        └── CostBreakdown (total + stacked bar + legend)
    └── Tab: Builds
        └── Timeline (vertical line + colored dots + compact cards)
    └── Tab: Environment
        └── Unified List (single card, divide-y rows, segmented toggle)
    └── Tab: Services
        └── Service Cards (status-bordered, containers as rows)
    └── Tab: Settings
        └── Settings sections (form cards)
```

---

## Notes

This is a living document. After reviewing the current state, mark what works and what doesn't, and we'll refine from there.
