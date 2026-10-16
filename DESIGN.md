# Karvan design system

Read this before building or changing any page. If something here is wrong, fix this file first, then the code.

## 1. Identity

Karvan moves users from one panel to another, and their configs keep working. The panel should look like the tool it sits next to: a clean operator console in the style of the Cloudflare dashboard, built on Cloudflare's Kumo components.

- **Mark ("Merge"):** two overlapping rounded squares in white on a Kumo-blue rounded tile, with the overlap filled. One panel, another panel, and what they share. Source of truth: `web/src/components/brand/Mark.tsx` (viewBox 64, tile radius 14, stroke 3.2). `web/public/favicon.svg` and `docs/brand/karvan-mark.svg` are static copies with the blue written as `#056dff`.
- **Name:** "Karvan" in English, "کاروان" in Persian, set in Vazirmatn semibold next to the mark (`Wordmark.tsx`). Never stretched, outlined or given effects.
- **Mood:** calm, flat and exact. White and grey surfaces, hairlines, one blue.

### Panel logos
The sign-in page and the overview show the Marzban and PasarGuard logos to say which panels Karvan works with. Both projects are AGPL-3.0, like Karvan; the logos are used only to name them, never to suggest they endorse Karvan.
- Marzban: its logo path, inline in `components/panels/PanelLogo.tsx`, drawn in `currentColor`.
- PasarGuard: its `logo-dark.png`, resized to 128 px (`components/panels/pasarguard.png`). Always shown on a white tile so it reads in both themes.

### Where boldness goes
Only one place per screen gets the memorable treatment:
- Sign-in: the blue side with the dotted globe.
- Every other page: the content itself. Chrome stays quiet.

## 2. Color

Karvan uses Kumo's palette as it ships: neutral greys for surfaces and text, Kumo blue as the brand, Kumo's status colors. Components use Kumo semantic classes (`bg-kumo-base`, `bg-kumo-tint`, `text-kumo-subtle`, `border-kumo-hairline`, `bg-kumo-brand`, ...).

`web/src/styles/tokens.css` holds the only color Karvan adds:

| Token | Value | Use |
|---|---|---|
| `on-brand` | white | text, icons and marks drawn on `bg-kumo-brand` |

**No color literal anywhere else** in `web/src` (the floor check fails the build). If a new color is truly needed, add a token there and a row here.

- `text-kumo-brand` is Cloudflare orange in Kumo. For blue text use `text-(--color-kumo-brand)`.
- Status is never shown by color alone. It always comes with an icon and a word.
- **Contrast:** body text at least 4.5:1 on its surface in both themes; large text and icons at least 3:1. Check new pairs before adding a token.

## 3. Type

- **One family: Vazirmatn (variable)** for Persian and English. Its Latin is drawn to sit with its Persian, so a Persian sentence with a username in it reads as one voice.
- **JetBrains Mono** only for machine values: UUIDs, tokens, ports, paths, IPs, commands.

| Step | Size / line-height (en) | fa | Weight | Use |
|---|---|---|---|---|
| display | 28 / 36 | 28 / 40 | 700 | sign-in tagline only |
| title | 22 / 30 | 22 / 34 | 600 | page title (one `h1` per page) |
| heading | 16 / 24 | 16 / 26 | 600 | card and section titles (`h2`) |
| body | 14 / 22 | 15 / 26 | 400 | default |
| small | 13 / 20 | 13 / 22 | 400 | helper text, meta, table cells |
| tiny | 12 / 16 | 12 / 18 | 500 | badges, counters |

- Sizes live in `web/src/styles/type.css`. Persian runs one step looser in line-height; Vazirmatn's tall forms need it.
- Sentence case everywhere. **No all-caps labels**, no letter-spaced eyebrows.
- Numbers in tables and counters use tabular figures.
- Line length stays under 75 characters for paragraphs.

## 4. Space, radius, elevation

- **Spacing scale (px):** 4, 8, 12, 16, 20, 24, 32, 40, 56. Nothing off-scale, except 2 px optical nudges and 2 or 6 px padding inside small inline elements (badges, keyboard keys).
- **Layout:** sidebar 256 wide, top bar and sidebar header 56 high, content gutters 32 (16 under 640). Pages with side information use a main column plus a 352 aside, split by a hairline.
- **Radius:** 6 for badges and nav rows; 8 for buttons, inputs and cards; 12 for dialogs and the command palette. Pills only for "Soon" tags and status dots.
- **Elevation:** surfaces are flat, separated by a 1 px hairline. Shadows only on things that float: menus, popovers, dialogs, the mobile sheet, toasts. Cards have no shadow.

## 5. Motion

- State changes: 150 ms, ease-out. Drawers and dialogs: 220 ms. Nothing bounces.
- Motion answers an action (open, expand, confirm) or shows real progress. No entrance animations on page load.
- The one exception is the globe on the sign-in page: it turns slowly (about one turn a minute) and acts out what Karvan does (section 8, Sign-in). It draws at about 30 frames a second and stops when it is off screen.
- `prefers-reduced-motion: reduce`: no transforms or autoplay; transitions shorten to 1 ms. The globe shows one still frame.

## 6. Icons

- Phosphor, `regular` weight: 18 px in navigation, 20 px in buttons, 16 px inline.
- Icons never stand in for words in primary actions. Icon-only buttons have an `aria-label` and a title or tooltip.
- No emoji in the UI.

## 7. RTL and languages

- `<html lang dir>` is set before first paint (`public/boot.js`). Persian is the default.
- Use logical properties only: `ms-* me-* ps-* pe-* start-* end-* border-s border-e rounded-s rounded-e text-start`. Physical `left/right` classes are a bug. In an RTL flex column `self-start` is already the right edge; do not add `rtl:` overrides to "fix" it.
- Icons that point a direction (arrows, chevrons, sign out) flip in RTL with `rtl:-scale-x-100`. The logo, the panel logos and the globe never flip.
- Machine values (UUID, IP, port, path, username, commands) are wrapped in `<bdi dir="ltr">` or `dir="ltr"` and stay in Latin digits.
- Persian text uses Persian digits (۰ to ۹); English uses Latin digits. `src/lib/format` does this; never format numbers by hand.
- Dates: Jalali calendar and Asia/Tehran by default, Gregorian in English unless the operator chooses otherwise. Times say which zone they are in when it matters ("It is 18:40 in Tehran").
- Kumo components that use physical classes get checked in RTL; any broken one gets a wrapper or a workaround in Karvan code instead of being used raw. Known cases: Kumo's Sidebar (not used), the DropdownMenu item `icon` slot (put the icon in the children with `gap-2` instead).

## 8. Components and patterns

- **Kumo first** for leaf components: Button, Input, Field, Select, Switch, Dialog, DropdownMenu, Tooltip, Badge, Toast, CommandPalette, Table. Import each from `@cloudflare/kumo/components/<name>`, and add its chunk name to the `@source` line in `src/styles/app.css`, or its styles are missing.
- The sign-in page stays light: no Tooltip, Dropdown or CommandPalette there. `LanguageSwitch` (one button naming the other language) and `ThemeToggle` (one button) exist for this reason.

### Shell
Karvan's own components, with logical properties:
- **Sidebar:** wordmark header with a bottom hairline; panel switcher row; "Search... Ctrl K" button that opens the command palette; grouped navigation with small group labels, 36 px rows, the current page on `bg-kumo-tint`.
- **Top bar:** on phones the menu button, wordmark and search; on all sizes the language switch (from 640 up), theme toggle and account menu at the end.
- **Mobile:** the sidebar opens as a sheet from the start edge, with focus inside it and Escape to close.
- **Command palette:** every page and preference is reachable from Ctrl K.

### Page
- **Page header:** title, one line of context under it, actions at the end, bottom hairline across the full width.
- **Cards:** `bg-kumo-base`, hairline border, radius 8, title row at the top, no shadow, no divider under the title unless the card holds a table.
- **Tables:** header row on a hairline, rows about 48 px high, a footer that says how much is shown ("Showing the latest 5 events"). Long lists live on their own page, never on the overview. On phones, secondary columns fold under the first cell instead of scrolling sideways.
- **Aside:** short status blocks (Safe Mode, two-step sign-in, account), each a heading, a status badge and one sentence, divided by hairlines.

### Sign-in
- Two halves on large screens. Start side on `bg-kumo-base`: wordmark at the top, then a centered form (title, username, password, one blue button) and the recovery hint with the `karvan reset -password` command.
- End side on `bg-kumo-brand`: language and theme buttons at the top, the "Marzban ⇄ PasarGuard" line, and one short line, then a spinning dotted globe.
- The globe acts out four scenes in turn (`globe-scenes.ts`): a move and a merge in each direction. Two panels appear on land, each as its logo on a white tile with its name under it; users travel as dots along a dashed arc. In a move the sending panel fades out once the last user arrives; in a merge both stay.
- The globe is drawn on a canvas from a land mask made by `scripts/brand/globe.mjs` (`globe-land.ts`, about 2 KB gzip), so no map code ships. It is a lazy chunk with its own budget, loaded after the form.
- Under 1024 px only the form side shows; the language and theme buttons move into its top bar.

### Behavior
- **Primary action:** one per area, `variant="primary"`. Everything else is secondary or ghost.
- **Loading:** skeletons shaped like the content, never spinners for content. Spinners only inside a button that is working.
- **Empty states:** an icon, what is missing, why, and one action.
- **Errors:** say what happened and how to fix it, in plain words. Never show raw server errors. Errors do not apologize.
- **Dangerous actions:** need the name typed to confirm (delete, restore, turning Safe Mode off).
- **Status badges:** icon + word + Kumo status color.
- **Touch:** every target is at least 44 × 44 px on touch screens.
- **Not yet built:** navigation items for later pages stay visible but inert with a "Soon" tag, so the structure can be judged early.

## 9. Writing

The voice is a friend who built this in their spare time: casual and warm, not robotic, formal or salesy. Nobody reading it should think a machine wrote it.

- **Short.** Say it once. No extra explanation, no "coming in the next version" notes, nothing that repeats what the screen already shows. If a line can go, it goes.
- **No hype.** No big promises, no "seamless", no "without any hassle".
- A button says what happens: "Sign in", not "Submit". The same action keeps the same name in the toast that confirms it.
- Name things the way the operator does ("users", "panels"), not the way the code does.
- **Persian** is casual spoken Persian, written in Persian, not translated word for word. Use "تو", not "شما": "رمزت یادت رفته؟", "دوباره امتحان کن". Everyday loan words are fine (بکاپ، تم، لود، رفرش).
  - Never use the half-space (U+200C). "می" and "نمی" join the verb: "میرن", "میشه", "نمیشه". Plural "ها" joins after a letter that does not connect (کاربرها) and takes a space after one that does (پنل ها).
  - A spaced "ها" can wrap onto the next line in a heading; reword it if it does.
  - No hamza on he (U+0654): "نسخه بعدی". No guillemet quotes, no Persian semicolon, no ellipsis character (use three dots), no em or en dash.
- **English:** plain, short, sentence case. Straight quotes and apostrophes only, no em or en dashes, three dots instead of the ellipsis character.
- No exclamation marks, no filler ("Please") unless leaving it out would sound rude.
- `scripts/check-floor.sh` fails on any of the banned characters in tracked files.

## 10. Do and don't

| Do | Don't |
|---|---|
| Hairline borders for structure, flat cards | Soft shadows under cards, gradients, glows and glass as decoration |
| One bold thing per screen | Decoration competing with content |
| Kumo blue for the brand and primary actions | Cloudflare orange anywhere, or extra brand colors |
| Real content and real data in every state | Lorem ipsum, fake numbers, emoji |
| Sentence case, quiet labels | All-caps eyebrows, "01 / 02 / 03" markers on things that are not steps |
| Logical CSS properties | `left`, `right`, `ml-*`, `pr-*` |
| Test at 360 and 1440 px, fa and en, dark and light, keyboard only | Shipping a page seen in one language and one theme |
