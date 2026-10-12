---
name: Monochrome Engineering
colors:
  surface: '#131313'
  surface-dim: '#131313'
  surface-bright: '#3a3939'
  surface-container-lowest: '#0e0e0e'
  surface-container-low: '#1c1b1b'
  surface-container: '#201f1f'
  surface-container-high: '#2a2a2a'
  surface-container-highest: '#353534'
  on-surface: '#e5e2e1'
  on-surface-variant: '#c1c6d6'
  inverse-surface: '#e5e2e1'
  inverse-on-surface: '#313030'
  outline: '#8b909f'
  outline-variant: '#414754'
  surface-tint: '#acc7ff'
  primary: '#acc7ff'
  on-primary: '#002f68'
  primary-container: '#498fff'
  on-primary-container: '#00285b'
  inverse-primary: '#005bbf'
  secondary: '#4edea3'
  on-secondary: '#003824'
  secondary-container: '#00a572'
  on-secondary-container: '#00311f'
  tertiary: '#ffb68b'
  on-tertiary: '#522300'
  tertiary-container: '#e47014'
  on-tertiary-container: '#481e00'
  error: '#ffb4ab'
  on-error: '#690005'
  error-container: '#93000a'
  on-error-container: '#ffdad6'
  primary-fixed: '#d7e2ff'
  primary-fixed-dim: '#acc7ff'
  on-primary-fixed: '#001a40'
  on-primary-fixed-variant: '#004492'
  secondary-fixed: '#6ffbbe'
  secondary-fixed-dim: '#4edea3'
  on-secondary-fixed: '#002113'
  on-secondary-fixed-variant: '#005236'
  tertiary-fixed: '#ffdbc8'
  tertiary-fixed-dim: '#ffb68b'
  on-tertiary-fixed: '#321300'
  on-tertiary-fixed-variant: '#753400'
  background: '#131313'
  on-background: '#e5e2e1'
  surface-variant: '#353534'
typography:
  headline-xl:
    fontFamily: Geist
    fontSize: 48px
    fontWeight: '600'
    lineHeight: 56px
    letterSpacing: -0.03em
  headline-xl-mobile:
    fontFamily: Geist
    fontSize: 32px
    fontWeight: '600'
    lineHeight: 40px
    letterSpacing: -0.02em
  headline-lg:
    fontFamily: Geist
    fontSize: 32px
    fontWeight: '500'
    lineHeight: 40px
    letterSpacing: -0.02em
  headline-lg-mobile:
    fontFamily: Geist
    fontSize: 24px
    fontWeight: '500'
    lineHeight: 32px
    letterSpacing: -0.015em
  headline-md:
    fontFamily: Geist
    fontSize: 20px
    fontWeight: '500'
    lineHeight: 28px
    letterSpacing: -0.01em
  body-lg:
    fontFamily: Geist
    fontSize: 16px
    fontWeight: '400'
    lineHeight: 26px
    letterSpacing: -0.005em
  body-md:
    fontFamily: Geist
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 22px
    letterSpacing: 0em
  body-sm:
    fontFamily: Geist
    fontSize: 12px
    fontWeight: '400'
    lineHeight: 18px
    letterSpacing: 0em
  label-mono:
    fontFamily: JetBrains Mono
    fontSize: 12px
    fontWeight: '400'
    lineHeight: 16px
    letterSpacing: 0.02em
  label-mono-sm:
    fontFamily: JetBrains Mono
    fontSize: 11px
    fontWeight: '400'
    lineHeight: 14px
    letterSpacing: 0.03em
rounded:
  sm: 0.125rem
  DEFAULT: 0.25rem
  md: 0.375rem
  lg: 0.5rem
  xl: 0.75rem
  full: 9999px
spacing:
  gutter: 1.5rem
  gutter-sm: 1rem
  margin: 2rem
  margin-sm: 1rem
  space-xs: 0.25rem
  space-sm: 0.5rem
  space-md: 1rem
  space-lg: 1.5rem
  space-xl: 2.5rem
---

## Brand & Style

This design system embodies a calm, hyper-focused utilitarian minimalism tailored for developers, systems engineers, and technical founders. It strips away ornamental noise—eschewing ambient blurs, gradient fills, and decorative border transitions—in favor of raw informational density, structural clarity, and razor-sharp typographic hierarchy.

The aesthetic communicates quiet competence, operational trust, and precision. Surfaces feel dense and matte, anchoring user focus entirely on content, data flows, and active tasks. Primary accents are applied with surgical discipline: never for visual embellishment, only to communicate primary interactive intent, focus states, and programmatic status.

## Colors

The palette is engineered around deep, near-black values to minimize visual strain and create a flat, matte canvas:
- **Canvas / Root Background:** `#080808` (deep black) with subtle structural zones stepping to `#0c0c0c`.
- **Containers & Surfaces:** `#121212` for primary functional cards, panels, and sidebars; `#161616` reserved for hover elevations and active input backdrops.
- **Borders & Dividers:** Rigid, single-pixel hairline borders in `#222222`. Secondary subtle dividers use `#1a1a1a`.
- **Text & Foreground:** Crisp primary text at `#f0f0f0`, muted secondary metadata at `#888888`, and disabled/tertiary placeholders at `#555555`.
- **Interactive Accent (Primary Blue):** `#2f81f7` used strictly for primary call-to-actions, active navigation highlights, selected radio/checkbox fills, and inline links.
- **Status / Indicator Green:** `#10b981` utilized strictly as an inline mono indicator or tag for live health, benefits badges, and active services.

## Typography

Typography prioritizes functional legibility and density. `Geist` drives all narrative, marketing, and operational content with a neutral, Swiss-inspired modernist neutrality. Tight letter tracking on headlines creates a compact, monolithic visual rhythm.

`JetBrains Mono` handles all programmatic identifiers, runtime logs, metric readouts, category chips, and key-value pair tags. It is deliberately constrained to small scales (`11px` to `12px`) to serve as a precise secondary layer.

## Layout & Spacing

The layout is built upon an uncompromising 12-column grid constrained to a maximum width of `1280px` for desktop views, transitioning to a unified single or dual-column vertical flow on mobile viewports.

Spacing relies on strict structural modules rather than organic gaps. Section transitions rely on rhythmic vertical padding (`space-xl` to double `space-xl`) rather than divider graphics. Margins conform to strict boundaries: `2rem` on wide displays, scaling gracefully down to `1rem` below `768px`. Content grids favor symmetrical split layouts—such as a side-by-side split between interactive feature stacks and high-contrast photographic/code viewports.

## Elevation & Depth

This system avoids ambient shadows, colored glows, and blur filters. Depth is established purely through low-contrast outlines and stepped tonal fills:

- **Base Layer:** `#080808` canvas.
- **Structural Containers:** `#121212` with a mandatory `1px solid #222222` outline.
- **Interactive State / Hover:** Fills transition to `#161616`, while the hairline border shifts to `#2f81f7` or `#333333`.
- **Card Nesting Rule:** Strictly zero nesting of card containers inside other cards. Visual hierarchies inside a surface must be grouped using whitespace, single horizontal lines (`#222222`), or mono category tags.

## Shapes

Corner radii adhere strictly to a compact, structural radius (`0.25rem` / `4px` base up to `0.5rem` / `8px` maximum).

- **Standard Inputs, Badges, and Rows:** `4px` border-radius to preserve an architectural, engineered appearance.
- **Primary Hero CTAs & Modal Dialogs:** `6px` to `8px` maximum.
- **Pills:** Never used for full cards or interactive buttons. Fully rounded pill treatments are reserved exclusively for miniature status indicators (`rounded-full` with height `20px` or less).

## Components

### Buttons
- **Primary Action:** Solid crisp white `#f0f0f0` fill with `#080808` bold text, or `#2f81f7` fill with white text for explicit system triggers. Height: `36px` to `40px`, padding: `0 16px`, `4px` border radius. No drop shadow. Hover state reduces opacity to `90%`.
- **Secondary / Ghost:** `#121212` background, `1px solid #222222` border, `#f0f0f0` text. Hover shifts border to `#444444` and background to `#181818`.

### Stacked List Selectors / Accordion Items
- Structural panels stacked vertically with `space-sm` gap.
- Background `#121212`, hairline border `#222222`, rounded `4px`.
- Active items display a vertical indicator bar on the left edge (`2px` wide, `#10b981` or `#2f81f7`), crisp `#f0f0f0` title, and an unrolled `#888888` description block below. Inactive items display as single-line collapsed bars with `#888888` text that turns `#f0f0f0` on hover.

### Chips & Status Badges
- Constructed with `label-mono` typography.
- Background `#121212`, `1px solid #222222`, padding `2px 8px`, rounded `4px`.
- Status variants feature a leading `6px` circular dot colored `#10b981` (active) or `#888888` (idle).

### Form Controls & Inputs
- **Text Inputs:** Height `38px`, background `#0c0c0c`, `1px solid #222222`, text `#f0f0f0`, placeholder `#555555`. Focus state converts border to `#2f81f7` without glow rings.
- **Checkboxes & Radios:** `16px x 16px`, background `#121212`, border `#333333`. Checked state fills `#2f81f7` with a white checkmark icon or center dot.

### Code / Terminal Panels
- Surface `#0c0c0c` surrounded by `1px solid #1a1a1a`. Padding: `16px`. Font: `JetBrains Mono` at `13px` with `#888888` line numbers and `#f0f0f0` text.