# Design System — Derived from the Approved Stitch Overview

Source: `docs/design-references/overview/` (`code.html`, `DESIGN.md`,
`screen.png`). Stitch export name: "Monochrome Engineering".

Where the sources disagree, the section notes it. `code.html` (the actual
exported markup) takes precedence over `DESIGN.md` prose for Overview styling;
the screenshot is partially corrupted (overlapping text layers) and is usable
for structure and color impression only, not for exact measurements.

## 1. Confirmed tokens (from `code.html` Tailwind config)

| Role            | Value     | Usage in source                              |
|-----------------|-----------|----------------------------------------------|
| `background`    | `#080808` | Page canvas, sidebar logo glyph text         |
| `surface`       | `#121212` | Sidebar, header, cards, table, filter input  |
| `surface-raised`| `#161616` | Workspace switcher, `#142` chip, hover fills |
| `border`        | `#222222` | All 1px hairline borders and dividers        |
| `accent`        | `#2f81f7` | Active nav text, primary button fill, links  |
| `accent-hover`  | `#266ed0` | Primary button hover                         |
| `foreground`    | `#f0f0f0` | Primary text, logo glyph fill                |
| `muted`         | `#888888` | Secondary text, icons, placeholders          |
| `subtle`        | `#555555` | Tertiary dividers (`•`, `/`), icon hover edge|

Active nav background is a one-off `#1a2333` (dark blue-tinted surface, accent
text) — not a named token.

Status badge colors are inline one-offs in `code.html` (hue / 10% bg / 20%
border): amber `#e0a84e` (`NEEDS_INFORMATION`), green `#4edea3` (`READY`), red
`#e06c75` (`JOB_FAILED`).

### Discrepancies vs `DESIGN.md` (unresolved)

- `DESIGN.md` frontmatter defines a second palette (`surface-container*`,
  `primary: #acc7ff`, `secondary: #4edea3`, `tertiary`, `error`, `*-fixed`
  ramps). `code.html` uses none of it except the status greens. For Overview
  work, use the `code.html` table above.
- Prose says status green is `#10b981`; `code.html` badges use `#4edea3`.
  `code.html` wins for badges; `#10b981` is not confirmed anywhere in markup.
- Prose mentions secondary divider `#1a1a1a` and hover bg `#181818`; markup
  uses `border` (`#222222`) dividers and `surface-raised` (`#161616`) hovers.
  `#1a1a1a` / `#181818` are unconfirmed.
- Prose canvas mentions `#0c0c0c` structural zones; markup canvas is flat
  `#080808`. `#0c0c0c` is unconfirmed for Overview.

## 2. Typography and font loading

- Sans: `Geist` (weights 300–600 loaded), fallback `-apple-system,
  BlinkMacSystemFont, sans-serif`. All narrative and UI text.
- Mono: `JetBrains Mono` (weights 400–500 loaded), fallback `monospace`. All
  programmatic content: workspace name, breadcrumb, repo names, issue numbers,
  counts, table cells, chips, filter input, `⌘K` hint.
- Icons: `Material Symbols Outlined` icon font (variable `FILL@100..700`).
  External Google Fonts dependency — see §8.
- Confirmed scale (from markup + `DESIGN.md` type ramp, which agree):
  - Page title `20px` / semibold / tight tracking (`headline-md`).
  - Section headings `14px` / medium.
  - Body/descriptions `12–13px`, secondary text `13px` muted.
  - Table headers `11px` mono muted; table body `13px` (mono `12px` for repo
    cells); badges/chips mono `10–11px`; footer meta mono `11px`.
- Titles truncate (`truncate`, `line-clamp-2` on card descriptions,
  `max-w-xs` on table titles).

## 3. Spacing and layout

- App frame: fixed sidebar `w-60` (`240px`); content offset `pl-60`; fixed top
  header `h-13` (`52px`, non-standard value); content `pt-13`.
- Content column: `max-w-6xl`, `px-8`, `py-8`; page sections stacked with
  `gap-8`; section internals `gap-3`.
- Sidebar internals: `p-4` header block, `px-3 py-4` nav, `p-3` user footer.
- Cards: `p-4`; card grid `gap-3`; table cells `py-3 px-4` (`py-2.5` header).
- `DESIGN.md` spacing scale (`gutter 1.5rem`, `margin 2rem/1rem`, `space-*`)
  is not referenced by class name in the markup; confirmed values are the ones
  above. The `2rem → 1rem below 768px` margin rule is prose-only and
  unconfirmed (markup uses fixed `px-8`).

## 4. Surfaces, borders, radii, shadows

- Depth comes only from stepped tonal fills + 1px hairline borders. **No
  shadows, gradients, blurs (except the header `backdrop-blur-md`), or
  decorative transitions anywhere.** Card-nesting rule: never place a card
  inside a card; group with whitespace, horizontal rules, or mono tags.
- Borders: `1px solid #222222` on sidebar edge, header, cards, table wrapper,
  inputs, chips. Inner dividers (`border-t`, `divide-y`) also `#222222`.
- Radii: default `rounded` (`4px`) on cards, buttons, inputs, chips, table
  wrapper (`overflow-hidden`); small chips/pills `rounded` to `rounded-full`
  only for the `20px`-tall avatar and the `#142` breadcrumb chip. No
  large-radius surfaces.
- Hover: surfaces shift to `surface-raised` (or `surface-raised/40–50` at
  reduced opacity for table rows/headers); card and switcher borders shift to
  `subtle` (`#555555`); text hovers go muted → foreground. Focus (filter
  input): border → `subtle`, no glow ring. Buttons: accent → `accent-hover`;
  ghost/secondary → border `#444444` (prose-only, unconfirmed in markup).

## 5. Components and states

- **Sidebar**: logo glyph (white rounded square, `adjust` icon) + `Issue
  Refiner` 14px; workspace switcher button (raised fill, `domain` icon, mono
  org name, `unfold_more` icon); nav items 13px with 17px icons — active item
  tinted bg `#1a2333` + accent text, inactive muted → foreground/raised on
  hover; footer with 28px initials avatar, 13px name, 11px mono role, icon-only
  sign-out button.
- **Top header**: 12px mono breadcrumb (muted links, `subtle` separators,
  current-issue chip); right side has search + notifications 28px icon buttons
  and a 24px circular avatar. All header content in the export is static mock
  (specific repo, `#142`); no behavior is specified.
- **Page head**: 20px title + 13px muted subtitle; `Filter repositories...`
  input (`h-8 w-56`, mono 12px, search icon, `⌘K` kbd chip, `id=
  "quickFilterInput"` implying client-side filtering — no JS shipped) and
  `Import repository` primary button (`h-8`, accent fill, `add` icon). Import
  opens nothing in the export; the destination/flow is unknown.
- **Repo cards**: mono 13px name + 10px mono Public/Private chip; 12px muted
  two-line description; footer mono 11px (`N open issues • ago`) + white
  `Open` text button (hover → accent).
- **Investigations table**: raised/50 mono 11px muted header (`Repository &
  Issue`, `Title`, `Status`, `Model`, right-aligned `Actions`); rows hover
  raised/40; status badges mono 11px tinted as in §1; `View result` 12px muted
  link → foreground on hover.
- **Counts** next to section headings are 12px mono muted (`3`, `4`); their
  data source (total vs capped list length) is unspecified.

## 6. Responsive rules (as evidenced)

- Repo grid: 1 column → `md:` 3 columns.
- Page-head row stacks vertically below `sm` (`flex-col sm:flex-row`).
- Table has `min-w-[760px]` inside an `overflow-hidden` wrapper (horizontal
  scroll implied on narrow screens, not explicitly handled).
- No other breakpoints exist in the markup. `DESIGN.md`'s 12-column/1280px
  grid and `768px` margin rule are prose-only and unconfirmed for this screen.

## 7. Asset reuse guidance

- Reuse markup structure and class patterns from `code.html` directly when
  rebuilding a screen; it is the authoritative reference.
- Static mock copy (repo names, descriptions, issue counts, times, breadcrumb,
  user identity) is placeholder — replace with real API data, never copy mock
  values into the app.
- `screen.png` is currently corrupted (overlapping layers); usable for color
  impression only until re-exported. Request a clean export if exact visual
  comparison is needed.

## 8. Unresolved questions

- Icon strategy: the export depends on the `Material Symbols Outlined` webfont
  (external Google Fonts request). Whether to ship that font, self-host it, or
  replace icons with inline SVG is undecided — do not silently add the external
  font link without a decision.
- `Manage integrations →` destination, `Open` button destination, `View
  result` destination, notification/search behavior, `⌘K` shortcut behavior,
  import flow, and count semantics are all unspecified by the export.
- Selection color (`accent/20`) and hidden scrollbars / `overscroll-behavior:
  none` are confirmed in markup but have no product rationale; keep them only
  if harmless.

## 9. Visual anti-patterns (explicitly out of system)

- Gradient fills, ambient/colored shadows, glow rings, blur surfaces (outside
  the header treatment), decorative border transitions.
- Pill-shaped buttons or cards; colors outside the §1 tokens (except the three
  confirmed status hues, used only for status badges).
- Generic dashboard cards, telemetry widgets, invented metrics, or icon
  treatments not present in the export.
- Light-mode styling: the system is dark-only; no light variant is specified.
- Copying Stitch mock data (names, counts, times) as if it were real content.
