---
target: external/react/src/
total_score: 27
max_score: 40
na_heuristics: 
p0_count: 2
p1_count: 1
p2_count: 1
p3_count: 1
timestamp: 2026-07-29T22-22-00Z
slug: external-react-src
---
# Khayal PWA Design Critique

**Target:** external/react/src/
**Heuristic Score:** 27/40 (Acceptable)
**Date:** 2026-07-30

---

| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | 4 | No spinner on send button during capture wait |
| 2 | Match Between System and Real World | 3 | "hybrid/keyword/semantic" are opaque jargon |
| 3 | User Control and Freedom | 3 | 3.5s auto-dismiss is uncontrollable; no undo on discard |
| 4 | Consistency and Standards | 3 | TextCapture uses forbidden weight 300; timeAgo duplicated 5x; .mc class collision |
| 5 | Error Prevention | 2 | No confirmation before discard; no URL validation |
| 6 | Recognition Rather Than Recall | 2 | Search modes never explained; "queue" concept never introduced |
| 7 | Flexibility and Efficiency of Use | 3 | Cmd+Enter; no keyboard tab switching; no search-as-you-type |
| 8 | Aesthetic and Minimalist Design | 3 | Capture tab information-dense; onboarding uses foreign shadcn wrappers |
| 9 | Help Users Recognize, Diagnose, and Recover from Errors | 4 | Parsed error codes; expanded error views; retry/discard everywhere |
| 10 | Help and Documentation | 0 | No tooltips, contextual help, onboarding tour, mode explanations, or FAQ |

---

## Design Specificity

The thermal palette (near-void to Bone with a single Flame Gold accent) is distinctive. The shape pulse hierarchy, streak arc SVG, hourly bar charts, spark lines, and gradient-top-border hero lines are signature touches. But 30-40% of the interface is category-interchangeable: three-tab bottom nav, chip/pill filters, shadcn/ui primitives, the onboarding token form.

CLI detector found 2 issues — a false positive on a markdown blockquote and a negligible layout-transition on bar charts. Manual review found 50+ instances across 10 categories that the detector missed entirely.

Browser visualization was not possible (no running dev server).

---

## What's Working

1. **The capture result tile system** — Four distinct variants (success/queued/offline/error) with semantic colors, icon containers, drain-bar animation timing, and context-appropriate actions. The parseError function that splits code · message is excellent.

2. **The send button as a physical object** — The 50px gold circle with gradient, glow, scale press-down, and disabled opacity. The only element with a shadow — a deliberate, earned exception.

3. **The search-to-capture bridge** — When search returns zero results, a gold-tinted suggestion appears: "capture a note about this" with a + icon. Cross-tab thinking most apps miss.

## Priority Issues

### P0 — Zero accessibility infrastructure
No ARIA attributes anywhere in the components/ directory. 12+ unlabeled icon-only interactive elements. 4 unlabeled form controls. 5+ inputs with outline-none and no alternative focus indicator. Zero prefers-reduced-motion support despite Framer Motion transitions and CSS animations. 5+ locations communicate status through color alone. Touch targets as small as 8x8px. Placeholder text likely fails WCAG AA contrast.
- Fix: Add aria-label to every icon-only element. Add role/aria-selected/tabIndex to custom interactive elements. Replace outline-none with focus-visible. Wrap animations in prefers-reduced-motion: no-preference. Add aria-live to capture tiles. Increase touch targets to 44px.
- Suggested command: $impeccable audit external/react/src/

### P0 — Onboarding is cold and transactional
Glass card with bare password field and "connect" button. No atmosphere, no value proposition, no token guidance. Shadcn wrappers look foreign. type="password" triggers iOS password manager. The "Lamp Room" creative north star is absent from the first thing every user sees.
- Fix: Atmospheric entry screen with Bricolage Grotesque heading, value proposition, auto-discovered server URL, CSS-class-driven design. Use type="text" on token field.
- Suggested command: $impeccable onboard external/react/src/components/Onboarding.tsx

### P1 — Capture tab overloaded
Stats dashboard (~220px bento grid) gates the compose area. On iPhone SE, send button falls below fold. After the 5th daily capture, streak stats are noise. Queue empty state renders a void with a tiny refresh button — no "queue is empty" reassurance.
- Fix: Collapse stats into compact horizontal strip with pulldown disclosure. Compose area at 70%+ viewport. Add queue empty state matching search idle pattern.
- Suggested command: $impeccable distill external/react/src/components/capture/CaptureView.tsx

### P2 — Hardcoded colors bypass the design token system
30+ CSS custom properties declared but 45+ inline style instances use raw hex/rgba values across 11 files. The two sources of truth diverge: CSS says --gold: #c9933a while BottomNav.tsx hardcodes color: "#C9933A". Unmaintainable.
- Fix: Replace all inline color styles with Tailwind classes or CSS variable references. Extract duplicated timeAgo to utils/time.ts. Fix .mc class collision.
- Suggested command: $impeccable extract external/react/src/

### P3 — Search modes undocumented; camera button misleads
"hybrid/keyword/semantic" with zero explanation. Filter chips without result counts. "open camera" button calls fileRef.click() — same as drop zone. No capture="environment". OR divider is functionally meaningless.
- Fix: Add mode descriptions beneath chips. Add result counts to filter chips. Implement real camera access or rename button to "choose from library".
- Suggested command: $impeccable clarify external/react/src/components/search/SearchView.tsx

---

## Persona Red Flags

### Daily Thinker (10-20 captures/day)
Stats gatekeep the compose area every single open. Auto-dismiss races rapid capture. TextCapture uses forbidden Bricolage 300 weight — should be IBM Plex Mono 400.

### Vault Explorer (searches frequently)
13+ visible actions in search idle. Search state destroyed on note close — returns to blank search, not previous results. No result counts on filter chips.

### Accessibility-Dependent User (screen reader, keyboard-only)
Cannot identify any icon-only element. Lost in stats grid — raw numbers with no context. Queue processing invisible — no aria-live region for job completion.

---

## Minor Observations

- timeAgo duplicated in 5 files — extract to utils/time.ts
- .mc class collision — mode chips and model-row children share class name
- UrlCapture's "add a note" field silently discards input (no ref, value never read)
- @keyframes indeterminate referenced but not defined in index.css
- Search scores at 2 decimal places are noise for fuzzy semantic results
- scrollbar-width: none removes desktop scroll affordance
- Sequential retryAll has no per-job feedback — 20+ failed jobs = 10+ second blackout
- btn-gradient:hover shadow on onboarding button violates shadow exception rule

## Questions to Consider

1. What if the capture tab were just a compose area — full-screen input with the send button at center-bottom, stats on a pulldown sheet or 4th tab?
2. What if search showed mode-badged result count previews as you type, so users commit to the mode that looks best?
3. What if the onboarding screen asked "Where is your server?" — auto-discovers URL, pings health, guides token entry only if needed?
