# UI-CONTEXT.md — Horus Frontend Reference

Authoritative visual and UX reference for any agent working on the Horus frontend. Read this before creating or modifying any UI. When a rule here conflicts with a default from a library or template (including shadcn examples), this document wins.

---

## 1. What Horus is

Horus is an **operational uptime-monitoring dashboard**. It is a tool, not a marketing site.

The UI must be: focused, restrained, technical, trustworthy, dense but readable, **calm when healthy, urgent only when something is actually wrong.**

**Reference products** (directional only, never copy literally):

| Product | Borrow |
|---|---|
| Better Stack | Information hierarchy, incident/status presentation |
| Linear | Spacing, typography, density, restraint, interaction quality |
| Uptime Kuma | Simplicity, obvious monitoring workflows |

**Tiebreaker:** when choosing between *more impressive* and *more understandable*, choose more understandable.

---

## 2. Core principles (priority order)

1. **Healthy is quiet. Problems are loud.** Neutral colors dominate; status color becomes prominent only when meaningful.
2. **Scan speed over decoration.** A user must spot an outage within seconds.
3. **Never rely on color alone.** Every status has a text label (and a shape where useful).
4. **Only expose what the backend supports.** No placeholder features, nav items, or fields.
5. **Density without cramping.** Compact tables/lists; related data close, sections clearly separated.
6. **Server state is the source of truth.** Keep UI state local and minimal.

---

## 3. Quick rules (cheat sheet)

**Do**
- Dark neutral surfaces, subtle borders, strong text contrast
- Small radii, minimal shadows, no/low gradients
- Tables or dense lists for monitors, checks, incidents
- Status as `dot + label` (`● HEALTHY`, `● DOWN`, `○ DISABLED`)
- Tabular numbers for all metrics and timestamps
- Skeletons for known layouts; inline errors with Retry
- Confirmation dialog for delete only
- shadcn/Radix primitives for accessibility

**Don't**
- Card grids for everything, giant KPI numbers, oversized headings
- Full-row / full-card status background colors
- Glassmorphism, glow, decorative gradients, heavy shadows
- Decorative charts, pie charts for binary data, 3D, many series
- Marketing copy, hero sections, excessive icons, big empty whitespace
- Bouncing/pulsing status icons, animated backgrounds, page transitions
- Toast-only validation or toast-only page-critical errors
- Settings / Teams / Billing / Users nav (not implemented)
- A global state library or a custom design-system package (v1)

---

## 4. Visual system

### 4.1 Theme
Dark operational interface by default. Neutral grays for surfaces; one restrained accent.

### 4.2 Suggested tokens
These are starting defaults. Adjust values if needed, but keep the *roles* and *restraint*. Define them as CSS variables / Tailwind theme tokens, and reference tokens, never raw hex, in components.

| Role | Intent |
|---|---|
| `background` | App background, darkest neutral |
| `surface` | Panels, tables, sidebar (one step above background) |
| `surface-raised` | Dialogs, dropdowns (one more step) |
| `border` | Subtle 1px dividers and outlines |
| `foreground` | Primary text, high contrast |
| `muted-foreground` | Secondary metadata, helper text |
| `accent` | Links, selected nav, primary actions, focus rings |
| `status-down` | Red, failure states only |
| `status-healthy` | Green, confirmed healthy only, used sparingly |
| `status-warning` | Amber, **reserved**, do not use unless a degraded/warning state exists |
| `status-disabled` | Muted neutral |

Shape and effects: small radii (e.g. 4–6px), 1px borders, shadows only on floating layers (dialogs, menus), no gradients, no glow.

### 4.3 Color semantics

| Color | Use for | Never use for |
|---|---|---|
| **Neutral** | Surfaces, nav, text, borders, inactive controls, **healthy rows** | — |
| **Red** | Monitor DOWN, active incident, failed check, destructive actions | Decoration, emphasis, branding |
| **Amber** | Warning/degraded **only if Horus gains that concept** | Inventing warning states |
| **Green** | Small dot + `HEALTHY` label for confirmed healthy | Large cards, row backgrounds, buttons |
| **Accent** | Links, selected nav, primary buttons, focus states | Anything that competes with status colors |

The accent color must remain visually distinct from, and quieter than, status colors.

### 4.4 Typography
Compact, highly readable, clear hierarchy through **weight, color, and modest size steps**, not large size jumps.

Hierarchy: `Page title` → `Section title` → `Primary data` → `Body` → `Secondary metadata` → `Muted helper text`.

- No oversized headings.
- Use `font-variant-numeric: tabular-nums` (Tailwind: `tabular-nums`) on every metric, latency, percentage, and timestamp.
- Monospace is acceptable for URLs, error strings, and HTTP codes if it improves scanning; don't use it for everything.

### 4.5 Data formatting

Keep operational values consistent everywhere.

| Value | Format | Example |
|---|---|---|
| Uptime | Percentage, consistent decimals | `99.98%` |
| Latency | Integer + unit, space before unit | `218 ms` |
| HTTP status | Code only in tables; `HTTP 503` in prose/incidents | `503` / `HTTP 503` |
| Duration | Short, largest sensible unit(s) | `12 min`, `2 h 5 min` |
| Recent time | Relative, with absolute on hover/tooltip | `20s ago` |
| Missing value | Em dash, never blank or `null` | `—` |

### 4.6 Spacing and density
Medium-to-high density. Use a consistent spacing scale. Rule of thumb: **related → tight; different sections → clearly separated.** Several monitors should be visible without scrolling.

### 4.7 Animation
Minimal. Allowed: subtle dialog/dropdown transitions, small loading indicators. Everything else is out. Monitoring data should feel stable, not animated.

---

## 5. Status vocabulary

Use exactly these labels and indicators. Keep them short and consistent across every screen.

| State | Indicator | Label | Text style |
|---|---|---|---|
| Healthy | `●` small green dot | `HEALTHY` | Neutral text |
| Down | `●` red dot | `DOWN` | Normal weight, red indicator |
| Disabled | `○` hollow muted dot | `DISABLED` | Muted text, row subdued |

- Disabled ≠ failure. Never style it as an error.
- Do not add states (e.g. "Degraded", "Pending") that the backend does not provide.
- Provide an accessible name for the indicator (visible label or `aria-label`).

---

## 6. Layout and navigation

- Persistent **left sidebar** on desktop.
- Nav, in this order, and **only this**:

```text
Horus
  Overview
  Monitors
  Incidents
```

- Do not add nav items until the feature exists.
- Keep hierarchy flat (no deep nav trees in v1).
- Monitor detail must have a clear route back to the monitor list (breadcrumb or back link).
- Selected nav item uses the accent color.

### Responsive behavior
Desktop is primary, but tablet and mobile must be usable.
- Collapse the sidebar on small screens.
- Stack metrics vertically.
- Tables scroll horizontally **or** switch to compact rows.
- Do not force every desktop column onto mobile.
- Always preserve status visibility and primary actions.

---

## 7. Screen specifications

### 7.1 Overview / Dashboard
Must answer immediately: **Is anything down? How many monitors are healthy? Are there active incidents? What changed recently?**

Priority order (top to bottom):
1. Active problems (down monitors, active incidents)
2. Overall monitor status (e.g. healthy / down / disabled counts)
3. Useful summary metrics
4. Recent activity / history

Do **not** open with decorative analytics. When nothing is wrong, the active-problems area should be quiet and clearly confirm that.

### 7.2 Monitor list
- Default presentation: **dense table or list**. Never a grid of large cards.
- Rows are clickable (navigate to detail) where appropriate.
- Columns:

```text
Status | Monitor name | URL | Latest status | Latency | Uptime | Last check | Actions
```

Example:

```text
● DOWN     API Production   api.example.com   503   812 ms   94.2%   20s ago
● HEALTHY  Website          example.com       200   181 ms   99.9%   15s ago
○ DISABLED Staging          staging.ex.com    —     —        —       —
```

- Healthy rows: neutral. Down rows: restrained red indicator on the status cell, not a full-row fill.
- Actions column: lightweight (row dropdown or icon buttons with tooltips), not visually dominant.

### 7.3 Monitor detail
Current condition first. Top-to-bottom:

```text
1. Monitor name + current status
2. URL
3. Key metrics (3–4 max): Uptime · Average latency · Latest HTTP status
4. Response-time / history visualization
5. Current incident (if any)
6. Recent checks
7. Incident history
```

Do not overload the top with many cards. Include a back route to the monitor list and the lightweight enable/disable/delete actions.

### 7.4 Incidents
- **Active** incidents: strong visual priority. Show: `monitor`, `failure type`, `HTTP status / error`, `started time`, `current duration`.
- **Resolved** incidents: visually quieter. Never as urgent as active ones.
- History is chronological, **newest first**, easy to scan.

### 7.5 Checks (recent checks list)
Compact table. Fields:

```text
Status | HTTP code | Latency | Failure type | Checked at
```

- Successful checks: neutral.
- Failed checks: restrained red indicator only. Never paint the whole row red.

### 7.6 Charts
Every chart must answer a monitoring question. Appropriate: recent response time, recent success/failure history, uptime/check-outcome trend.
- Prefer simple line or bar charts.
- Readable at a glance, minimal legends, few series.
- Forbidden: 3D, pie charts for trivial binary data, dozens of series, decorative charts.

### 7.7 Create monitor form
Include **only implemented fields**:

```text
Name · URL · Interval · Timeout · Expected status
```

- Explicit labels and helper text where needed.
- Don't expose backend terminology unnecessarily (use user-facing language).
- **Inline validation** next to fields. Never hide validation failures only in a toast.
- Submit button shows a loading state; on success, navigate/refetch.

---

## 8. Actions and feedback

| Action | Prominence | Confirmation | Feedback |
|---|---|---|---|
| Create | Primary button | No | Button loading → navigate/refetch |
| Enable / Disable | Lightweight | **No** (unless a clear reason exists) | Immediate pending state → row updates |
| Delete | Low prominence, destructive styling in dialog | **Yes**, confirmation dialog | Confirm → remove row / refetch |
| Navigation | — | Never | — |

Users must never have to guess whether an action succeeded. Mutations always show pending and result states.

### Dialogs and sheets
- Dialogs: destructive confirmation and small focused forms.
- Sheets/drawers: only when they genuinely improve the workflow.
- Prefer full pages for substantial information. Do not put every action in a modal.

---

## 9. States

Every data-driven screen and component must handle all three.

| State | Requirement |
|---|---|
| **Loading** | Use skeletons where the layout is known. No full-page centered spinners. Small actions get local indicators. Layout stays stable during refetch. |
| **Empty** | Explain what's missing **and** what to do next, with an action. Never a bare "No data". |
| **Error** | Show near the affected content, with a Retry action where practical. Don't expose backend/internal error details. Don't rely only on toasts for page-critical failures. |

Reference copy:

```text
No monitors yet
Create your first monitor to start checking uptime.
[ Add monitor ]
```

```text
Could not load monitors.
[ Retry ]
```

---

## 10. Accessibility (minimum bar)

- All controls keyboard accessible, with **visible focus states**
- Sufficient color contrast (check muted text on dark surfaces)
- Semantic HTML (real `table`, `button`, `nav`, headings in order)
- Labels on all form inputs
- Status never conveyed by color alone
- Buttons (including icon-only) have accessible names
- Dialogs trap focus, handled via the component library
- Prefer shadcn/Radix primitives over custom implementations

---

## 11. Tech and architecture conventions

### Components
- Use **shadcn/ui** as a toolkit, not as the final design. Adjust composition and density for Horus.
- Primitives in use: `Button`, `Dialog`, `DropdownMenu`, `Table`, `Input`, `Select`, `Badge`, `Tooltip`, `Skeleton`, `Alert`.
- Avoid unnecessary component abstraction. No custom design-system package in v1.

### State management

| Kind | Tool |
|---|---|
| Server state: `monitors`, `checks`, `summary`, `incidents` | **TanStack Query** |
| Local UI state: dialogs, selected rows, form state, temporary filters | React local state |
| Global state library | **Not allowed** unless a concrete need appears |

---

## 12. v1 scope

In scope:

```text
Overview/dashboard · Monitor list · Create monitor · Monitor detail
Enable / Disable / Delete monitor · Check history · Summary / uptime
Current incident · Incident history · Loading / Empty / Error states
```

Out of scope: anything the backend does not support. If a feature isn't listed above or in the API, **do not build it, stub it, or add nav for it.**

---

## 13. Review checklist

Before marking any screen complete, verify every item:

- [ ] Can the user tell the system's state immediately?
- [ ] Are failures more visible than healthy states?
- [ ] Is healthy information visually quiet?
- [ ] Is the page easy to scan (dense, aligned, tabular numbers)?
- [ ] Are actions obvious but not visually dominant?
- [ ] Are destructive actions protected by a confirmation dialog?
- [ ] Are loading, empty, and error states all handled?
- [ ] Does it work without relying on color alone?
- [ ] Is unnecessary decoration removed?
- [ ] Does it expose only implemented functionality?
- [ ] Is it keyboard-usable with visible focus?
- [ ] Is it usable on a narrow viewport?

---

## 14. Anti-patterns

```text
card grid for every piece of information     giant KPI numbers
rainbow status colors                        heavy gradients
glassmorphism                                oversized rounded corners
marketing copy                               huge empty whitespace
excessive icons                              meaningless charts
hidden primary actions                       status communicated only by color
```

---

## Guiding rule

> When choosing between **more impressive** and **more understandable**, choose the more understandable design. Horus is an operational tool first.
