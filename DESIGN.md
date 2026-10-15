# Karvan design system

Read this before building or changing any page. If something here is wrong, fix this file first, then the code.

## 1. Identity

Karvan moves people's users from one panel to another. A caravan (کاروان) crosses the desert from one caravanserai to the next and arrives with everything it set out with. The panel should feel the same: calm, careful, and sure of the road.

- **Mark:** an eight-pointed Persian star (khatam, from girih tilework) in firouzeh turquoise, cut by a road, with saffron dashes along the road. Source of truth: `scripts/brand/logo.py`. Every logo, favicon and the 3D logo is generated from its polygons.
- **3D mark (sign-in only):** the same polygons extruded as a glazed tile, with the road as a darker bed and the dashes as a gold inlay.
- **Name:** "Karvan" in English, «کاروان» in Persian. Set in Vazirmatn ExtraBold. Never stretched, outlined or given effects.
- **Mood:** glazed tile and night travel. Light theme: kahgel plaster walls in daylight. Dark theme: the desert at night, lamp-warm text on cool dark.
- **Calmer than a cloud console.** Karvan does fewer things. Fewer colors on screen, more space, one clear action per area.

### Where boldness goes
Only one place per screen gets the memorable treatment:
- Login: the 3D star.
- Every other page: the content itself. Chrome stays quiet.

## 2. Color

Colors exist only as tokens in `web/src/styles/tokens.css`. Components use Kumo semantic classes (`bg-kumo-base`, `text-kumo-subtle`, ...) or the Karvan brand classes (`text-karvan-firouzeh`, `fill-karvan-saffron`). **No color literal anywhere else** (the floor check fails the build).

### Brand
| Token | Light | Dark | Use |
|---|---|---|---|
| firouzeh | `oklch(46% 0.085 196)` | `oklch(76% 0.1 188)` | the mark, links, focus rings, selected state |
| brand (button fill) | `oklch(46% 0.085 196)` | `oklch(49% 0.088 194)` | primary buttons only; white text on both |
| saffron | `oklch(68% 0.15 62)` | `oklch(80% 0.14 75)` | the road: dashes, active-route marks, progress. Never body text |

### Neutrals
Light neutrals carry a trace of sand (hue 75 to 80, chroma 0.002 to 0.012). They stay close to white, so the page never turns cream. Dark neutrals lean cool (hue 220, chroma about 0.013), like night sky over sand. Dark text stays warm (hue 85).

| Role | Light | Dark |
|---|---|---|
| canvas (page) | 97.6% | 15.5% |
| base (panels, inputs) | 99.4% | 19% |
| recessed | 95.4% | 14% |
| hairline | 91.5% | 26% |
| text default | 25.5% | 93% |
| text subtle | 50% | 72% |

### Status
| Meaning | Name | Solid light / dark |
|---|---|---|
| success | pistachio | `oklch(58% 0.12 150)` / `oklch(72% 0.13 150)` |
| warning | saffron | `oklch(74% 0.15 70)` / `oklch(72% 0.15 68)` |
| danger | pomegranate (anar) | `oklch(57% 0.19 25)` / `oklch(64% 0.19 25)` |
| info | lapis (lajvard) | `oklch(55% 0.12 250)` / `oklch(70% 0.11 245)` |

Status is never shown by color alone. It always comes with an icon and a word.

### Contrast
Body text is at least 4.5:1 on its surface in both themes, and large text and icons at least 3:1. Check new pairs with a contrast tool before adding a token.

## 3. Type

- **One family: Vazirmatn (variable)** for Persian and English. Its Latin is drawn to sit with its Persian, so a Persian sentence with a username in it reads as one voice.
- **JetBrains Mono** only for machine values: UUIDs, tokens, ports, paths, IPs, commands.

| Step | Size / line-height (en) | fa | Weight | Use |
|---|---|---|---|---|
| display | 28 / 36 | 28 / 40 | 800 | login wordmark only |
| title | 22 / 30 | 22 / 34 | 700 | page title (one `h1` per page) |
| heading | 16 / 24 | 16 / 26 | 650 | section titles (`h2`) |
| body | 14 / 22 | 15 / 26 | 400 | default |
| small | 13 / 20 | 13 / 22 | 400 | helper text, meta |
| tiny | 12 / 16 | 12 / 18 | 500 | badges, counters |

- Persian runs one step looser in line-height; Vazirmatn's tall forms need it.
- Sentence case everywhere. **No all-caps labels**, no letter-spaced eyebrows.
- Numbers in tables and counters use tabular figures.
- Line length stays under 75 characters for paragraphs.

## 4. Space, radius, elevation

- **Spacing scale (px):** 4, 8, 12, 16, 20, 24, 32, 40, 56. Nothing off-scale.
- **Layout:** sidebar 256 wide, top bar 56 high, content max 1120 with 24 gutters (16 under 640).
- **Radius has hierarchy:** 6 for small controls (badges, chips); 8 for buttons and inputs; 12 for panels; 16 for dialogs and the login card. Pills only for status dots.
- **Elevation:** surfaces are flat and separated by a 1px hairline plus a background step. Shadows only on things that float: menus, popovers, dialogs, toasts. No soft shadows under panels.

## 5. Motion

- State changes: 150 ms, ease-out. Drawers and dialogs: 220 ms. Nothing bounces.
- Motion answers an action (open, expand, confirm), or shows real progress. No entrance animations on page load. The login 3D star is the single exception: a slow sway (14 s per swing) that leans a little toward the pointer, and a 700 ms fade from the flat mark.
- `prefers-reduced-motion: reduce`: no transforms or autoplay. Fades shorten to 1 ms. The 3D star is replaced by the flat mark.
- Weak devices (2 GB memory or less, no WebGL, slow first frames) get the flat mark too.

## 6. Icons

- Phosphor, `regular` weight, 20 px in navigation and buttons, 16 px inline. `bold` marks the current nav item.
- Icons never stand in for words in primary actions. Icon-only buttons have an `aria-label` and a tooltip.
- No emoji in the UI.

## 7. RTL and languages

- `<html lang dir>` is set before first paint (`public/boot.js`). Persian is the default.
- Use logical properties only: `ms-* me-* ps-* pe-* start-* end-* border-s border-e rounded-s rounded-e text-start`. Physical `left/right` classes are a bug.
- Icons that point a direction (arrows, chevrons, "back") flip in RTL with `rtl:-scale-x-100`. The logo never flips.
- Machine values (UUID, IP, port, path, username) are wrapped in `<bdi dir="ltr">` and stay in Latin digits.
- Persian text uses Persian digits (۰–۹); English uses Latin digits. `src/lib/format` does this; never format numbers by hand.
- Dates: Jalali calendar and Asia/Tehran by default, Gregorian in English unless the operator chooses otherwise. Times say which zone they are in when it matters ("at 18:40 Tehran time").
- Kumo components that use physical classes (Sidebar, InputGroup, Combobox, Dropdown, Toast) get checked in RTL; any broken one gets a wrapper in `src/components/` instead of being used raw.

## 8. Components and patterns

- **Kumo first** for leaf components: Button, Input, Field, Select, Switch, Dialog, DropdownMenu, Tooltip, Badge, Toast, CommandPalette, Table. Import each from `@cloudflare/kumo/components/<name>`, and add its chunk name to the `@source` line in `src/styles/app.css`, or its styles are missing.
- Kumo's Button already pulls in its tooltip code; Tooltip itself stays out of the sign-in page to keep it light.
- **Own shell:** sidebar, top bar, page header and panel switcher are Karvan components with logical properties.
- **Primary action:** one per area, `variant="primary"`. Everything else is secondary or ghost.
- **Loading:** skeletons shaped like the content, never spinners for content. Spinners only inside a button that is working.
- **Empty states:** say what is missing, why, and what to do. Always one action.
- **Errors:** say what happened and how to fix it, in plain words. Never show raw server errors. Errors do not apologize.
- **Dangerous actions:** need the name typed to confirm (delete, restore, turning Safe Mode off).
- **Status badges:** icon + word + color, from the status table above.
- **Touch:** every target is at least 44 × 44 px on touch screens.
- **Not yet built:** navigation items for later pages stay visible but inert with a "soon" tag, so the structure can be judged early.

## 9. Writing

- Plain, calm and direct, like a careful colleague. Active voice. Name things the way the operator thinks about them ("users", "panels"), not the way the code does.
- A button says what happens: "Sign in", not "Submit". The same action keeps the same name in the toast that confirms it.
- Persian copy is written in Persian, not translated word for word. Use the half-space (ZWNJ) correctly («می‌شود», «کاربران‌ها» is wrong, «کاربران» is right).
- No exclamation marks, no filler ("Please", «لطفاً») unless leaving it out would sound rude.

## 10. Do and don't

| Do | Don't |
|---|---|
| Hairline borders and background steps for structure | Identical soft-shadow cards in a grid |
| One bold thing per screen | Gradients, glows and glass as decoration |
| Firouzeh and saffron, used sparingly | Default purple/blue, or Cloudflare orange as the primary |
| Real content and real data in every state | Lorem ipsum, fake numbers, emoji |
| Sentence case, quiet labels | All-caps eyebrows, "01 / 02 / 03" markers on things that are not steps, "→" stuck on links |
| Logical CSS properties | `left`, `right`, `ml-*`, `pr-*` |
| Test at 360 and 1440 px, fa and en, dark and light, keyboard only | Shipping a page seen in one language and one theme |
