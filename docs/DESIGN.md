---
name: Khayal
description: A private treasury of thought — local-first second brain with a flame-gold thermal palette on near-void dark.
colors:
  flame-gold: "#c9933a"
  pale-flame: "#e8b86d"
  deep-flame: "#a67830"
  flame-glow: "rgba(201,147,58,0.06)"
  flame-glow-strong: "rgba(201,147,58,0.12)"
  flame-glow-medium: "rgba(201,147,58,0.4)"
  green-signal: "#3ddc84"
  amber-pulse: "#ffb340"
  red-warning: "#ff4d4d"
  near-void: "#070707"
  ink: "#0d0d0d"
  slate: "#141414"
  charcoal: "#1c1c1c"
  gone: "#f5f5f5"
  ash: "rgba(245,245,245,0.5)"
  smoke: "rgba(245,245,245,0.2)"
  hairline: "rgba(255,255,255,0.05)"
  outline: "rgba(255,255,255,0.09)"
typography:
  display:
    fontFamily: "Bricolage Grotesque, sans-serif"
    fontWeight: 800
    fontSize: "clamp(1.375rem, 5vw, 2rem)"
    lineHeight: 1.2
    letterSpacing: "-0.03em"
  body:
    fontFamily: "IBM Plex Mono, ui-monospace, monospace"
    fontWeight: 400
    fontSize: "16px"
    lineHeight: 1.6
  label:
    fontFamily: "IBM Plex Mono, ui-monospace, monospace"
    fontWeight: 700
    fontSize: "9px"
    lineHeight: 1
    letterSpacing: "0.05em"
rounded:
  pill: "100px"
  lg: "20px"
  md: "16px"
  sm: "12px"
  xs: "8px"
spacing:
  tiny: "4px"
  compact: "8px"
  standard: "12px"
  comfortable: "16px"
  frame: "20px"
components:
  button-primary:
    backgroundColor: "linear-gradient(135deg, {colors.flame-gold} 0%, {colors.deep-flame} 100%)"
    textColor: "#000000"
    rounded: "50%"
    size: "50px"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.ash}"
    rounded: "{rounded.sm}"
    padding: "8px"
  button-icon:
    backgroundColor: "{colors.slate}"
    textColor: "{colors.smoke}"
    rounded: "7px"
    size: "24px"
  chip-selected:
    backgroundColor: "{colors.flame-gold}"
    textColor: "#000000"
    rounded: "{rounded.pill}"
    padding: "4px 10px"
  chip-unselected:
    backgroundColor: "transparent"
    textColor: "{colors.smoke}"
    rounded: "{rounded.pill}"
    padding: "4px 10px"
  input-search:
    backgroundColor: "{colors.slate}"
    textColor: "{colors.bone}"
    rounded: "{rounded.sm}"
    padding: "11px 14px"
  card-standard:
    backgroundColor: "{colors.slate}"
    textColor: "{colors.bone}"
    rounded: "{rounded.md}"
    padding: "15px"
  nav-tab-active:
    textColor: "{colors.flame-gold}"
  nav-tab-inactive:
    textColor: "{colors.smoke}"
  badge-type-text:
    backgroundColor: "rgba(61,220,132,0.08)"
    textColor: "{colors.green-signal}"
    rounded: "{rounded.pill}"
    padding: "2px 6px"
---

# Design System: Khayal

## Overview

**Creative North Star: "The Lamp Room"**

A quiet chamber lit by a single warm flame. Every surface is precise and considered — never decorative. Gold is illumination, not ornament. Light reveals; it never blinds. The interface recedes so thought can advance.

Khayal's design language is a dark thermal palette anchored by Flame Gold (`#C9933A`), a warm, aged gold that glows rather than shouts. Near-void backgrounds (`#070707`) give depth without weight. The single accent color appears on at most 10% of any screen — its rarity is what gives it authority. When it appears, it marks the most important action, state, or result on the page.

The system is anchored in two typefaces: **Bricolage Grotesque** carries display hierarchy with weight and letter-spacing precision — it never shouts, even at 32px. **IBM Plex Mono** handles body, UI, labels, and metadata with the clarity of a tool built for daily use. Monospace here is honest: it is the font of data, code, measurements, and controls — never a costume for "technical."

Surfaces are flat at rest. Depth is conveyed through tonal layering (four background stops from `#070707` to `#1C1C1C`) rather than shadows. The only shadow in the system is the gold glow on the primary send button — a deliberate exception that makes the most important action feel consequential. Motion is minimal and authored: 150-200ms transitions on hover states, a gentle `scale(0.98)` press-down on tappable elements, and synchronized drain-bar animations on status tiles.

**Key Characteristics:**
- Single-accent thermal palette: one gold on layered near-void darks
- Monospace as honest material, not aesthetic signifier
- Flat-by-default surfaces with tonal depth, not shadow
- Deliberate rarity of the primary color — restraint is the luxury
- Tactile, deliberate interactions: every press feels consequential
- Rounded form language with a pulse: fully round pills, 16px card corners, 50% circle on the send button

## Colors

A thermal palette built from a single warm accent laid over layered darks. Semantic greens, ambers, and reds provide functional signals without competing with the gold.

### Primary
- **Flame Gold** (`#C9933A`): The single accent. Used on the send button, active navigation tabs, selected chips, group labels, the streak arc, and the hero result horizontal accent line. Appears on ≤10% of any screen. Its HSL equivalent is `36 56% 51%`.
- **Pale Flame** (`#E8B86D`): The accent's lighter sister. Used for keyword search highlights, URL display text, today's dot in the week bar, and hero result ghost numbers. Carries warmth without commanding attention.
- **Deep Flame** (`#A67830`): The darker end of the gold gradient. Used only in the send button gradient (`linear-gradient(135deg, #c9933a, #a67830)`) and nowhere else. Creates the thermal transition from hot to warm.

### Neutral
- **Near-Void** (`#070707`): The deepest background. Page-level surface. Behind everything.
- **Ink** (`#0D0D0D`): First elevation step. Recent search items, compact result cards, queue item backgrounds.
- **Slate** (`#141414`): Primary surface. Cards, inputs, tiles, bento grid cells, compose area, hero cards. The workhorse background.
- **Charcoal** (`#1C1C1C`): Elevated hover and active surface. Rarely used at rest.
- **Bone** (`#F5F5F5`): Primary text on all dark surfaces. White-adjacent but slightly warm to harmonize with the gold.
- **Ash** (`rgba(245,245,245,0.5)`): Secondary text. Metadata, descriptions, hover-state color for inactive elements.
- **Smoke** (`rgba(245,245,245,0.2)`): Subtle text. Labels, disabled states, placeholder text, inactive icons.
- **Hairline** (`rgba(255,255,255,0.05)`): Default border. The thinnest possible visible separation.
- **Outline** (`rgba(255,255,255,0.09)`): Stronger border. Used on search bars, URL input rows, hover borders, and header icon buttons.

### Semantic
- **Green Signal** (`#3DDC84`): Success, done, online indicator. Used on success tiles, done checkmarks, online dot, delta badges, and result timing. Has a subtle glow (`box-shadow: 0 0 8px #3ddc84`) on the online indicator.
- **Amber Pulse** (`#FFB340`): Processing and queued states. Used on processing hero cards, queued tiles, active step dots, queue dot indicators. Pulsing animation on processing indicators.
- **Red Warning** (`#FF4D4D`): Errors and failures. Used on error tiles, failed job cards, error codes, error text. Appears with a reduced-opacity background (`rgba(255,77,77,0.04)`).

### Named Rules
**The One Flame Rule.** Flame Gold is used on ≤10% of any given screen. Its rarity is the point. When it appears, it marks the single most important action or state. Never use it decoratively.

**The Thermal Depth Rule.** Lightness alone carries elevation. A surface at `#070707` is background; `#0D0D0D` is first elevation; `#141414` is card level; `#1C1C1C` is hover/active. No shadow participates in this hierarchy.

**The Ghost Border Rule.** All default borders use `rgba(255,255,255,0.05)`. Stronger emphasis bumps to `0.09`. Gold borders enter only for active/focus states and never exceed `rgba(201,147,58,0.3)`.

## Typography

**Display Font:** Bricolage Grotesque (with sans-serif fallback)
**Body Font:** IBM Plex Mono (with ui-monospace, monospace fallback)
**Label/Mono Font:** IBM Plex Mono (same stack as body)

**Character:** A disciplined pairing — Bricolage Grotesque provides warmth and personality at display sizes with controlled letter-spacing; IBM Plex Mono brings the precision and clarity of a tool that is used dozens of times daily. The monospace is functional material, not aesthetic posture.

### Hierarchy
- **Display** (800, 22-32px, 1.2 line-height, -0.03em tracking): Greetings, stat big numbers, hero numbers. Appears only in capture view stats and the streak counter. Weight 800 at all sizes; never regular.
- **Title** (600-700, 13-14px, 1.3-1.35 line-height, -0.01em tracking): Search result titles, hero result titles, card headings. Bricolage Grotesque at weight 600-700.
- **Body** (400, 16px, 1.6 line-height): Primary input text. Search bar value, text capture textarea. IBM Plex Mono at 16px minimum to prevent iOS auto-zoom.
- **Body Small** (400-500, 11-12px, 1.3-1.6 line-height): Descriptions, excerpt text, button labels, suggestion chips. IBM Plex Mono or system font depending on context.
- **Label** (700, 9px, 1.0 line-height, 0.05em tracking, uppercase): Section headers, field labels, metadata, timestamps, type badges, chip text. IBM Plex Mono exclusively. The uppercase treatment is consistent: letter-spacing between 0.03em and 0.1em depending on emphasis.

### Named Rules
**The 16px Floor Rule.** All text inputs render at minimum 16px. This prevents iOS Safari from auto-zooming on focus. The visual density of the UI is achieved through labels and metadata at 9px — never by shrinking input text.

**The No Leading Zero Rule.** Numeric display values (streak count, note count, capture count) use letter-spacing of -0.04em to -0.06em with weight 800. No leading zeros, no monospace numerals — these are values, not data tables.

## Layout

The PWA is a single-page app occupying `100svh` (smallest viewport height) with three tabs switched via bottom navigation. No page routing, no sidebar, no horizontal scroll.

**Page structure:** Header (fixed top, 56px including safe area) → Content (flex-1, overflow-y auto) → Bottom Nav (fixed bottom, 60px + safe area). The header carries the brand mark, app name, version pill, a 24px logout icon button, and the online indicator. The bottom nav has three tabs — Capture, Search, Queue — with icons, 9px uppercase labels, and a sliding gold indicator bar.

**Grid system:** The capture view uses a 2-column bento grid (`grid-template-columns: 1fr 1fr`) with an 8px gap for stats tiles. The streak and today tiles are single-column; the vault tile spans both columns. Cards use `border-radius: 18px` with `padding: 15px`.

**Spacing rhythm:** Component gaps at 4-8px (chips, dots, meta items), card padding at 12-16px, page horizontal padding at 12-14px, and navigation at 10-20px. All spacing grows from the 4px base unit: 4, 8, 12, 16, 20.

**Responsive behavior:** The PWA is mobile-first and primarily consumed as a touch interface. Touch targets minimum 44px height. Safe area insets respected via `env(safe-area-inset-*)` on header, bottom nav, and standalone mode viewport. Desktop usage inherits the same layout — no breakpoint-based reflow.

## Elevation & Depth

**Flat-by-default with tonal layering.** No cards carry box-shadow at rest. The four-step background scale (`#070707` → `#0D0D0D` → `#141414` → `#1C1C1C`) is the sole depth mechanism. Lighter surfaces read as closer to the user.

### Shadow Vocabulary
The system has exactly one shadow, and it is reserved:
- **Send Button Glow** (`box-shadow: 0 4px 16px rgba(201,147,58,0.3)`): Applied to the primary send button. A deliberate exception that makes the most important action feel physically present. On hover, this intensifies to `rgba(201,147,58,0.2)` via `btn-gradient:hover`.

Glass effects (`backdrop-filter: blur(20px)`) are used on the bottom navigation bar for a single reason: the nav sits above scrollable content and needs to establish its fixed surface without a hard opaque cutoff.

### Named Rules
**The Flat-By-Default Rule.** Surfaces are flat at rest. Shadows appear only as a response to state (the send button's default glow, hover intensification). A card with both a border and a shadow is a ghost card — never use both.

**The No Ghost Card Rule.** Declare elevation once: a border or a shadow. Never a 1px border beneath a wide soft shadow. Cards that need lift use a lighter background from the tonal scale; cards that need containment use a hairline border.

## Shapes

**Rounded form language with a pulse.** The system bends toward softness but keeps a deliberate center. Fully round shapes (50% circles, `100px` pills) house the most interactive elements — the send button, chips, badges, version tags. Large-radius shapes (16-20px) contain content: cards, tiles, the compose area, hero cards. Standard-radius shapes (12-14px) define inputs and list items.

**The pulse hierarchy:**
- **50% circle:** Only the send button. It is the most pressable thing on screen. Its roundness makes it physically distinct from every other element.
- **100px pill:** Chips, badges, version tags, type badges. Small controls and metadata that need to feel like tokens rather than boxes.
- **18-20px:** Bento stats tiles, compose area, processing hero card. Content containers that carry significant information.
- **14-16px:** Status tiles, search result cards, hero result cards, queue hero cards. The standard content card.
- **12-13px:** Search bar, URL input row, recent search items, queue items. Interactive list elements.
- **7-10px:** Icons within components, small containers, note input. Supporting elements.
- **2px:** Scrollbar thumbs, hour bars, week dots. Tiny informational elements.

**Borders:** Always `rgba(255,255,255,0.05)` at rest; gold enters at `rgba(201,147,58,0.15-0.3)` for active, focus, or selected states. Gold borders are muted enough to feel integrated, not decorative.

## Components

### Buttons
- **Shape:** Primary send button is a 50px circle. Action buttons (retry, discard, sync) are fully rounded (8-10px) with 100% width on mobile. All touch targets ≥44px.
- **Primary (Send):** `linear-gradient(135deg, #c9933a, #a67830)` background, `#000` icon, `0 4px 16px rgba(201,147,58,0.3)` glow. On hover: glow intensifies. On active: `scale(0.95-0.98)`. Disabled: `opacity: 0.3`, `pointer-events: none`. The onboarding CTA ("enter the lamp room") uses the same gradient in a 12px-radius full-width rectangle.
- **Ghost / Secondary:** Transparent background, `rgba(255,255,255,0.05)` border, text at `rgba(245,245,245,0.5)`. On hover: `rgba(255,255,255,0.04)` background. Used for retry, discard, back navigation, and secondary actions.
- **Accent Action:** `rgba(201,147,58,0.08)` background, `rgba(201,147,58,0.2)` border, `#c9933a` text. Used for retry actions and sync buttons where the gold accent signals recoverability.
- **Icon Action:** 24×24px, 7px radius, `--s2` background, `--border2` border, icon at `--t3`. Hover lifts border and icon to `--gold`. Used for the header logout. Deliberately compact — it sits in the header metadata cluster, not the thumb zone.

### Chips / Pills
- **Style:** 100px border-radius, 4-10px padding, IBM Plex Mono at 9px weight 700, uppercase.
- **Selected:** `#c9933a` background, `#000` text, `#c9933a` border, shadow `0 3px 10px rgba(201,147,58,0.25)`.
- **Unselected:** Transparent background, `rgba(255,255,255,0.09)` border, `rgba(245,245,245,0.2)` text. On hover: border shifts to `rgba(201,147,58,0.3)`, text to `rgba(245,245,245,0.4)`.
- **Filter variant:** Selected uses `rgba(201,147,58,0.1)` background (not solid gold) with `rgba(201,147,58,0.25)` border and `#c9933a` text. More restrained than mode chips because filters are secondary controls.
- **Mode descriptor:** A 9px Smoke line beneath the search mode chips names the active mode in plain language — "combines keyword and meaning search", "matches exact words and phrases", "finds notes by meaning".

### Cards / Containers
- **Corner Style:** 16-18px. Bento tiles at 18px; result cards at 16px.
- **Background:** `#141414` with `rgba(255,255,255,0.05)` border.
- **Shadow Strategy:** Flat at rest (see Elevation). No shadows.
- **Hover:** Background lifts to `#1C1C1C` on result cards; border shifts to `rgba(255,255,255,0.09)`.
- **Hero Variant:** Has a 1px gradient top accent line (`linear-gradient(90deg, #c9933a, transparent)`) for high-score results, or amber variant (`#ffb340`) for processing heroes. This is the only colored top-border in the system.
- **Streak Variant:** Uses `linear-gradient(145deg, rgba(201,147,58,0.1), rgba(201,147,58,0.02))` background with `rgba(201,147,58,0.18)` border — the warmest card in the system.
- **Internal Padding:** 14-16px. Tighter on compact results (10-12px).

### Inputs / Fields
- **Style:** `#141414` background, `rgba(255,255,255,0.09)` border, 12-14px border-radius. Text at 16px `#f5f5f5`. Placeholder at `rgba(245,245,245,0.2)`.
- **Focus:** Border lifts to `rgba(201,147,58,0.3)`, with `box-shadow: 0 0 16px rgba(201,147,58,0.1)` and an inset `0 0 0 1px rgba(201,147,58,0.08)`.
- **Active:** The search bar `.active` state adds the gold border and inset ring before the user types — anticipation feedback.
- **Clear:** 18px circle, `rgba(255,255,255,0.07)` background, `rgba(245,245,245,0.2)` text. Appears when input has content.

### Navigation
- **Bottom Nav:** Fixed to viewport bottom. `rgba(7,7,7,0.92)` background with `backdrop-filter: blur(20px)`. Three equal-width tabs. Icons at 20px, `rgba(245,245,245,0.2)` stroke, 1.5px stroke-width. Labels at 9px IBM Plex Mono, uppercase, `rgba(245,245,245,0.2)`.
- **Active Tab:** Icon stroke becomes `#c9933a`. Label color becomes `#c9933a`. A 20px×2px rounded gold bar slides to the active tab via shared layout animation.
- **Inactive Tab:** All elements at `rgba(245,245,245,0.2)`.
- **Safe Area:** `padding-bottom: max(env(safe-area-inset-bottom), 16px)`.

### Capture Result Tiles
- **Shape:** 14px border-radius, 12-14px padding, flex row with icon + content.
- **Success:** `rgba(61,220,132,0.05)` background, `rgba(61,220,132,0.12)` border. Green checkmark icon container. Auto-dismiss with 3s drain bar animation.
- **Queued:** `rgba(255,179,64,0.04)` background, `rgba(255,179,64,0.12)` border. Spinning loader icon. 4s auto-dismiss. Step dots show processing progress.
- **Offline:** `rgba(201,147,58,0.04)` background, `rgba(201,147,58,0.1)` border. Zap icon. 3.5s auto-dismiss.
- **Error:** `rgba(255,77,77,0.04)` background, `rgba(255,77,77,0.15)` border. Alert icon. No auto-dismiss. Shows error code + hint + retry/discard actions.

### Status Indicators
- **Online Dot:** 7px circle, `#3ddc84`, `box-shadow: 0 0 8px #3ddc84`. In the header. Carries `role="status"` with a text label (`connected` / `degraded` / `offline`) so state never rides on color alone.
- **Version Pill:** 100px border-radius, `#141414` background, `rgba(255,255,255,0.09)` border, 9px IBM Plex Mono, `rgba(245,245,245,0.2)` text. In the header.
- **Update Icon:** `#3ddc84` arrow icon. Appears in header when a new version is available.

### Empty States
- **Queue empty:** Centered column — 40px icon container (10px radius, `rgba(255,255,255,0.03)` background, `rgba(255,255,255,0.05)` border) with the icon at 15% Bone; a one-line title at 50% Bone, weight 600; sub-copy at 10px IBM Plex Mono Smoke. No card, no shadow — quiet by design.
- **Search idle / no results:** Same grammar — muted icon, one-line title, then actionable next steps (recent searches, suggestions, alternate-mode retries, the capture-this bridge). An empty state always offers a next action.

## Do's and Don'ts

### Do:
- **Do** use Flame Gold (`#C9933A`) as the single accent on at most 10% of any screen. Its rarity is the point.
- **Do** use the four-step background scale (`#070707` → `#0D0D0D` → `#141414` → `#1C1C1C`) for all depth. Never add a shadow where a tonal step would suffice.
- **Do** use Bricolage Grotesque at weight 800 for all display and stat numbers. Weight 600-700 for titles. Never use weight 300 or 400 for display.
- **Do** use IBM Plex Mono for all labels, metadata, timestamps, code, and measurements. It is functional material, not decoration.
- **Do** maintain the 16px minimum on all text inputs — this prevents iOS auto-zoom.
- **Do** respect the shape pulse hierarchy: 50% for the send button only, 100px for pills/chips/badges, 16-20px for cards, 12-14px for inputs.
- **Do** use `backdrop-filter: blur()` only on the bottom navigation bar. Glass elsewhere feels decorative rather than functional.

### Don't:
- **Don't** add a second accent color. The system is one gold. Semantic colors (green, amber, red) are functional signals, not design accents.
- **Don't** use box-shadow on cards. The send button's glow is the single deliberate exception.
- **Don't** combine a 1px border with a box-shadow on the same element. Declare elevation once.
- **Don't** use monospace as an aesthetic signifier for "technical." IBM Plex Mono appears where it is honest: code, data, labels, measurements, and UI controls.
- **Don't** use gradient text. Emphasis comes from weight or size within the two-typeface system.
- **Don't** introduce horizontal scroll. The PWA is a single-column layout at all viewport widths.
- **Don't** shrink input text below 16px. The visual density of the UI is achieved through 9px labels and metadata — never by reducing input legibility.
