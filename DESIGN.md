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
| `chart-1` to `chart-5` | Kumo's chart palette: blue, amber, pink, purple, teal (light and dark values from `ChartPalette.categorical`) | chart series, in this order; the first series of every chart is blue |
| `chart-grid` | Kumo's neutral chart fill (light and dark) | grid lines and axis ticks |

**No color literal anywhere else** in `web/src` (the floor check fails the build). If a new color is truly needed, add a token there and a row here.
Kumo's sixth chart color is an orange; Karvan never uses it (no Cloudflare orange). A chart never needs more than five series; more become "Other".
The chart tokens sit in a `@theme static` block: charts pick a color by number at run time (`var(--color-chart-3)`), which Tailwind's class scan cannot see, so the variables are always emitted.

- `text-kumo-brand` is Cloudflare orange in Kumo. For blue text use `text-(--color-kumo-brand)`.
- Status is never shown by color alone. It always comes with an icon and a word.
- **Contrast:** body text at least 4.5:1 on its surface in both themes; large text and icons at least 3:1. Check new pairs before adding a token.

## 3. Type

- **One family: Vazirmatn (variable)** for Persian and English. Its Latin is drawn to sit with its Persian, so a Persian sentence with a username in it reads as one voice.
- **JetBrains Mono** only for machine values: UUIDs, tokens, ports, paths, IPs, commands. Drawn without ligatures, so `://` and `->` read exactly as typed.

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
- **Layout:** sidebar 256 wide (64 when collapsed), top bar and sidebar header 56 high, content gutters 32 (16 under 640). Pages with side information use a main column plus a 352 aside, split by a hairline. Content stops growing at 1400 wide and stays at the start edge, like the dashboard.
- **Radius:** 6 for badges and nav rows; 8 for buttons, inputs and cards; 12 for dialogs and the command palette. Pills only for "Soon" tags and status dots.
- **Elevation:** surfaces are flat, separated by a 1 px hairline. Shadows only on things that float: menus, popovers, dialogs, the mobile sheet, toasts. Cards have no shadow.

## 5. Motion

- State changes: 150 ms, ease-out. Drawers and dialogs: 220 ms. Nothing bounces.
- Hover colors change at once, never with a transition (Kumo's rule: fading hovers feel slow).
- Motion answers an action (open, expand, confirm) or shows real progress. No entrance animations on page load.
- Live charts redraw in place when a sample arrives; they do not slide or tween. The live migration view moves dots only in step with real batches, and while a backup restore runs, which has no batches (section 8, Live progress).
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
- Charts keep time running left to right in both languages, like a graph in a school book; only their labels are Persian. A move from one panel to another reads in the page direction ("Marzban, arrow, PasarGuard" runs right to left in Persian).
- Machine values (UUID, IP, port, path, username, commands) are wrapped in `<bdi dir="ltr">` or `dir="ltr"` and stay in Latin digits.
- A machine value inside a translated sentence (a file, a setting, a command) is marked with backticks in the message, and the sentence is drawn with `CodeText`: the value keeps left to right in mono at about 0.9 em. Left unmarked, `.env` after Persian words reads `env.`.
- Persian text uses Persian digits (۰ to ۹); English uses Latin digits. `src/lib/format` does this; never format numbers by hand.
- Dates: Jalali calendar and Asia/Tehran by default, Gregorian in English unless the operator chooses otherwise. Times say which zone they are in when it matters ("It is 18:40 in Tehran").
- Kumo components that use physical classes get checked in RTL; any broken one gets a wrapper or a workaround in Karvan code instead of being used raw. Known cases: Kumo's Sidebar (not used), the DropdownMenu item `icon` slot (put the icon in the children with `gap-2` instead), Kumo's Table (`text-left`) and InputGroup (`pl-*`, `rounded-l-*`), which Karvan replaces with its own table and search field (section 8, Tables).

## 8. Components and patterns

- **Kumo first** for leaf components: Button, Input, Field, Select, Switch, Dialog, DropdownMenu, Tooltip, Badge, Toast, CommandPalette. Import each from `@cloudflare/kumo/components/<name>`, and add its chunk name to the `@source` line in `src/styles/app.css`, or its styles are missing.
- The sign-in page stays light: no Tooltip, Dropdown or CommandPalette there. `LanguageSwitch` (one button naming the other language) and `ThemeToggle` (one button) exist for this reason.

### Shell
The Cloudflare dashboard's frame, built from Karvan's own components with logical properties (Kumo's Sidebar uses physical classes):
- **Sidebar header (56 high):** the mark, then the panel switcher where the dashboard has its account switcher: the current panel's logo and name, an up and down chevron. Its menu lists the panels with a status dot, then "All panels", "Scan this server" and "Manage panels".
- **Sidebar body:** a "Search... Ctrl K" field that opens the palette, then grouped rows: icon, label, a chevron for rows with sub-pages, 36 high (44 on touch), the current row on `bg-kumo-tint`. Group labels are small and quiet. No Settings or Security rows: they live in the account menu.
- **Sidebar footer:** one button that collapses the sidebar to icons with tooltips; remembered per browser.
- **Top bar (56 high):** at the end, the Safe Mode status (a small badge that opens Settings, like the dashboard's plan badge) and the account button. On phones: menu button, mark, search, account button.
- **Account menu:** like the dashboard's: who is signed in, Settings, Appearance (light, dark, same as system), Language (فارسی, English), Keyboard shortcuts, then Sign out. Language and theme have no buttons of their own in the app.
- **Mobile:** the sidebar opens as a sheet from the start edge, with focus inside it and Escape to close.
- **Footer:** one quiet row at the end of the content: version, source code, report a security issue.
- **Command palette:** every page, panel, migration, report, setting and preference is reachable from Ctrl K. `?` opens the shortcut sheet.

### Page
- **Page header:** a small label above the title when the page sits inside another (the dashboard's breadcrumb), the title, one line of context under it, actions at the end, a bottom hairline across the full width.
- **Cards:** `bg-kumo-base`, hairline ring, radius 8, title row at the top, no shadow, no divider under the title unless the card holds a table.
- **Layer cards** (analytics, grouped facts): Kumo's LayerCard, a tinted title row over a white body, one level only (never a LayerCard inside another).
- **Tables:** a search field across the top with filters and refresh next to it; the table in a ringed frame with a tinted header row; rows about 48 high; "..." row menus; a footer that says how much is shown ("Showing 1 to 20 of 340") with previous and next page buttons (`components/data/Pager`; Kumo's Pagination has physical classes and carets that do not turn in Persian). Long lists live on their own page, never on the overview. Under 640 rows become stacked cards with the main value first, like the dashboard's DNS records on a phone.
  - Built from `components/data`: `DataTable` (the columns, and the card one row becomes on a phone) and `SearchField` (`/` focuses it). Kumo's Table is not used: it sets `text-left` and stripes its rows.
  - The first column names the row with its logo and links to its page, underlined as in the dashboard. Status cells are an icon in the status color and a word in the default color, like the dashboard's "Active".
  - A short list the server does not page ends with "Showing 2 of 2" and no pager.
- **Detail pages:** title with status badges, underline tabs, content cards, an aside for facts and quick actions.
- **Settings pages:** one page with underline tabs, one route per tab. Each setting is a row: title and one line of description at the start, the control at the end, rows split by hairlines.
- **Aside:** short status blocks, each a heading, a status badge and one sentence, divided by hairlines.
- **Facts** (`FactList`): label at the start, value at the end, rows split by hairlines. A value too long for the rest of its row (a path on a phone) moves under its label.
- **Panel names:** always with their logo (`PanelLabel`). A move between panels is shown as the source logo, an arrow and the destination logo (`MoveLabel`), in lists, the wizard, the live view, reports, old links and backups.

### Overview
Laid out like the dashboard's account home, top to bottom:
- **Header:** the title, the date line ("Wednesday, October 21, 2026, 16:39 in Tehran", in the operator's zone and calendar) and "New migration" at the end.
- **Search:** one wide field across the content that opens the palette; it says what the palette finds and shows its shortcut.
- **Three columns:** Panels, Migrations and Recents. Each has a quiet title (a link with a chevron when its page exists) and at most a handful of short rows split by hairlines, each row a link with a chevron at the end. A column with nothing in it says so in one line on a tinted box. Columns stack on phones.
  - Panels: logo, name and its status. Migrations: the move (`MoveLabel`), the admin, the state, then how many users moved and when.
  - Recents: the last pages this browser opened, newest first, at most 8, named in the current language as "section / page" ("Panels / logo name"). Kept in localStorage as paths inside the hidden path, never the hidden path itself.
- **Summary:** a heading, then three cards that differ in shape: users moved (the number, a meter of moved and skipped, the jobs by outcome), Safe Mode (status badge and its facts) and Old links (the count and the way in).
- **Analytics:** a heading with "All analytics", refresh and the range (Live, 24 hours, 7 days, 30 days) at the end. Then this server's tiles, and for each panel database its name with its logo, a row of tiles (size; connections and queries per second for a database server, the write-ahead log and used traffic for SQLite, which counts neither; online now) and two layer cards: users by status and traffic per hour. The section loads after the rest of the page.
- **No panels yet:** one empty state with both logos and "Scan this server" takes the place of the columns and the summary.

### Analytics
- Laid out like the dashboard's analytics: header with refresh and a range button (Live, 1 hour, 24 hours, 7 days, 30 days) joined in one group, underline tabs (Server, then one per panel with its logo), a row of stat tiles, then layer cards in two columns (one on phones).
- The range lives in the address (`?range=24h`) and carries across tabs, so two tabs compare the same time. Polls run only while the tab is visible and do not count as activity; pressing refresh does.
- **Stat tile:** label, the value in title size with its unit smaller, one quiet line under it (wraps to two lines on phones, never cut), a sparkline along the bottom edge (a meter for a disk). Tiles in one row share a height. A sparkline without a fixed scale spans its own lowest to highest value, so a slow climb (users, size) still shows its shape.
- **Time series:** thin 1.5 px lines, a faint area under the first series, dashed grid lines in `chart-grid`, four or five round ticks, axis labels in `small` and `text-kumo-subtle`. Labels are HTML over the SVG, so Persian digits and words read in the page direction while time runs left to right. Hovering, touching or moving with the arrow keys (the chart is a slider) shows a crosshair and a tooltip with the time and every series. Missing samples leave a gap, never a line to zero; a lone sample is a dot.
- **Bars** (traffic and new users): per hour up to a day, per day beyond, the series stacked; a quiet hour is a real zero.
- **Legend:** above the chart, each series with its color and number: the latest value when live; over a range the average, the peak, and for network, disk and traffic the total moved.
- **Bar list** (users by status, top admins, largest tables): label, a thin meter, the number; the longest bar fills the column. On phones the meter moves under the label so the words keep the row.
- **Numbers:** every chart shows its current or total value in text next to it; screen readers get a table of the values (at most 12 rows spread over the range). A chart is never the only place a number lives.
- **States:** a skeleton shaped like the chart while loading; "No data in this range" with the range named when a series is empty; a database Karvan cannot reach shows the panel's finding instead of an empty chart.

### Wizard
- One route per step, each a lazy page in its own folder under `features/wizard`: Connect at `/migrations/new`, then `/migrations/:id/summary`, `compare`, `changes`, `preview` and `start`. Nothing on either panel changes before Start, and no step says otherwise.
- **Header:** "Migrations" above the title, the title, and once a job exists the move (`MoveLabel`) with the old panel's address in the line under it.
- **Steps** (`components/form/Steps`): from 640 px the six steps in one row under the header, each a number in a small ring and its name; done steps are links with a check, the current one is in brand blue with `aria-current="step"`, later ones are quiet. Under 640 px one line, "Step 2 of 6: Summary", over a thin bar.
- **Body:** form steps in one column at most 720 wide; steps with lists may use the content width. Actions sit at the bottom end: the step's own verb as the primary action last, Back before it.
- **Connect:** the destination as radio cards (logo, name, version), the current panel chosen; a panel that is not ready is disabled with its first finding's sentence. Then the old panel: address, kind (find out by itself, Marzban, PasarGuard) and the sign-in (username and password, or a PasarGuard API key). A certificate Karvan does not trust shows a warning card with its SHA-256 fingerprint (mono, in groups of two), subject and expiry, and "Trust this certificate", which trusts exactly that fingerprint for this migration. While Karvan reads, the form gives way to a counter ("Reading users: 12,500 of 20,000"; the second pass says "Checking again"), a thin meter and Cancel; when the read ends the wizard moves on to Summary. The address keeps the job (`?job=`), so a reload picks the counter up again.
- **Summary:** two cards side by side, the old panel (panel, address, admin, users, used traffic) and the destination (panel, where users go), then three layer cards of bar lists (users by status, expiry, protocols), then the destination's groups or inbounds with their protocols, hosts and "Users go here". Continue at the end.
- **Compare:** a strip of what the plan does (move, left out, renamed, configs change) and what stands in the way, then cards in this order:
  - Admin: a taken name offers the free name Karvan found and a merge; then how it lands (new admin or into one already there) and its name, saved with its own button. On PasarGuard the role picker and what the role allows sit next to the name and the password. The password is the reseller's own (or one Karvan makes when Karvan has none), or a new one typed or made in the browser with Generate, shown once with Copy and checked against the panel's rules as it is typed.
  - Where users go: one checkbox per group or inbound, "Use the default" once the plan has its own list.
  - Problems: one row per group, most relevant first (still needing a fix, then blockers, warnings and notes, then size): its title, how many users, a state badge, the fixes for all as radios best first with a "Config changes" flag, and "Choose for each user", which opens the group's users with a fix per user.
  - Categories: "Add a category" offers ready-made conditions in one click, inside the open category when there is one. From 1024 px a tree (All users, then the categories nested, each with its count and a "..." menu) sits beside the open category's users; below it a breadcrumb and the categories inside the open one, to drill into.
- **Changes:** the same strip, then the changes already in the plan, grouped by who they reach (All users, a category, a filter, picked users), each group with how many users it reaches and each change in words with a "Config changes" or "Link changes" flag and Remove. Then "Add a change": who (All users, a category, users that match a filter, or users picked from the list), then what (one operation picked from a list, then its fields), then Add. Signed numbers (`SignedAmountField`) take a typed sign or a sign button, since phone number keyboards often have no minus key, and say what they do under the field. Dates (`JalaliDateField`) are day, month and year in the operator's calendar, Jalali by default in Persian. New credentials sit behind a warning and a checkbox. Last, Templates: apply one (a dialog says what it replaces) or save the choices as one by name.
- **Preview:** the same strip, then the pre-flight report's card (open the printable page in a new tab, download JSON, download CSV), then every user as they will land: a row of filter buttons with their counts (All, Renamed, Configs change, Left out; the chosen one raised) and a search over the list, 50 a page. A row is the name (with the new one after an arrow), its final status, data limit and expiry, and a "Config changes" or "Link changes" flag.
- **Printable pages** (`/migrations/:id/preview/print`, later reports): outside the shell, one column on `bg-kumo-base`, a quiet bar with Print and Close that does not print, then the facts and a plain table with a hairline per row. Black text on white in print whatever the theme.
- **Start:** cards in order, each a title and its content:
  - Check again: one button reads both panels again with the read counter; then "Nothing changed since you reviewed" (success), or what changed: new, gone and changed users and the admin, each user with before and after, and a checkbox to accept it. Start waits for a check of this visit and, after a change, for the checkbox.
  - Reload: the three methods as radio cards, each with what it does on this panel as a short list; a method Karvan cannot run is disabled with why; manual shows its command.
  - Safe Mode: on or off with its rollback (undo, or a backup), and "Change in Settings". A backup rollback the panel cannot do says so as a warning.
  - What changes for users: a short flagged list (new subscription links for every moved user and that Old links keep the old ones working once set up; how many configs change; renamed users).
  - Then the actions: Back and Start migration. With Safe Mode off, Start opens a dialog that wants the admin username typed. Starting opens the live view.
- **Plan edits:** a choice saves at once and shows at once, on top of the saved plan; a value that needs typing (a name, a prefix) saves with its own button. A save that meets a change from another tab reads the plan again and makes the change once more; a second conflict opens a dialog with "Keep the saved plan" and "Apply my change again".
- **Users in the wizard:** a search field, then rows with the name (the new one after an arrow when the plan renames it, "Left out" when the user stays behind), 50 a page with `Pager`.
- **Errors:** every API code gets its sentence from `lib/errors` (D4), in an `InlineAlert` above the actions. No server text reaches the screen.

### Live progress
- `/migrations/:id/live`, a lazy page that Start opens. It works in every state of the job, also after a restart: it reads the job's row by id and the job's stored event stream, and the report once the job ends.
- **Header:** "Migrations" above the title; the title says what happens now (Moving users, Taking users back, Putting the backup back, then Migration result) with the job's state as a label at the end; the move (`MoveLabel`), the old panel's address and its admin in the line under it.
- **Stage rail:** Check, Backup, Write, Verify, Reload in one row from 640 px (a list under it): a status icon, the name and its time (how long it took, or "since 16:39" while it runs). With Safe Mode off, Backup reads "Off"; stages a failure skipped read "Didn't run"; a stage a restart cut off reads "Stopped". Undo and Restore join the rail only when they run. The running stage has `aria-current="step"`.
- **The scene** (the page's one bold thing), in a card under the rail: the counter in heading size ("820 of 1,170 moved", tabular), the server's estimate ("About 2 minutes left") and Stop at the end, a thin meter, then the drawing:
  - Move: the two panels as tiles (logo, name, a number), a dashed arc between them that runs in the page direction. Each batch sends dots along the arc in `chart-1`, as many as the batch's share of the run and at least three, so a small batch still reads as one; skipped users branch off to a small sink under the arc with their count.
  - Undo: the same tiles; dots travel back in the warning color, and the destination counts the users removed.
  - Restore: the backup as a tile next to the panel; while the restore runs, dots flow from the backup into the panel, which reads Offline with its timer ("Offline for 0:14"), then Online again.
  - The tiles and numbers are HTML; the SVG draws only the arc, the sink branch and the dots, in pixels so the dots stay round. At most 24 dots are in flight. One animation loop at about 30 frames a second runs only while dots fly, the card is on screen and the tab is visible.
  - `prefers-reduced-motion` or the browser's Data Saver: no drawing, the meter and the tiles' numbers as a line of text.
- **Log:** one plain sentence for each stage, log line and end, with its time to the second in the operator's zone, oldest first. Batches move the counter, not the log. It sits in the aside from 1024 px, under the scene below that.
- **Order on phones:** the scene (or the result) comes first and the rail under it, so the counter is on the first screen; from 640 px the rail heads the page.
- **Result:** when the job ends, a card with the state, one sentence on what it means, the counts that are not zero, the reload result (a manual reload or a failed one shows the command to run), the downloads (report as JSON and CSV, the new links as CSV, and the admin password CSV once, behind a warning that it can be downloaded only once), then Set up old links (prefilled from the job), Save as template, and Roll back for a failed or interrupted job that may have left users behind. A run sent back because the panels changed says nothing was written and links to Start.
- Stop and Roll back ask first in a dialog that says what happens with Safe Mode on and off.

### Migrations, reports and templates
- **Lists** (`/migrations`, `/reports`) follow Tables: the search field, then a state filter and the panel filter (the switcher's current panel; "All panels" clears it) and refresh. A row is the move (`MoveLabel`) with the admin as the link and the old panel's host under it, the destination with its logo, the state, the users moved and when. The server pages by cursor, so the footer says "Showing 21 to 40" with previous and next and no total. A search looks through the latest 100 jobs that match the filters and says so.
- A draft whose login Karvan dropped says "Login dropped" under its state and offers "Start again with the same address", which opens Connect with the destination, the address and the kind filled in. The password is always typed again.
- `/migrations/:id` opens what the job needs now: a draft its step (Connect again when Karvan signed out of it), a run or a job that may need a rollback the live view, a done or rolled back job its report.
- **Report page:** "Reports" above the title, the state and the move in the header, Print and the downloads at the end. Then the result (state, counts), the facts (times, Safe Mode and its backup, the admin and its password), the stages with their times, what a rollback cannot take back and until when, the reload, and every user (filter buttons with counts, a search, 50 a page). Its printable page (`/reports/:id/print`) follows Printable pages.
- **Templates:** one card per template: its name, when it was saved, and what it does as a short list in words (each category with its conditions and changes, the fixes for problems, how reset times are kept). Delete asks for the name typed. Templates are applied and saved in the wizard.

### Old links
- `/links` lists the sites, one card each, with "Add a site" at the end of the header. Ports that could not open come first as one warning naming them.
- **Site card:** the name, the move (`MoveLabel`, old panel to the destination) and the destination's name in the title row, a "..." menu (Edit, Delete) at the end. Then, in two columns from 1024 px:
  - Facts: the old address, the domains ("Every domain on this port" when there are none), the old path, the port, HTTPS or plain HTTP, lazy checks on or off.
  - Certificate (HTTPS only): the names it covers, who issued it, the date it ends, then each warning as a status line: a site domain it does not cover, ends within 14 days, ended, not valid yet, no key or a key that does not fit, or a file Karvan cannot read.
  - Users: "34 of 120 proven" with a meter of proven over moved, then pending and closed as quiet numbers. Without numbers (the destination did not answer) one line says so.
  - Old links file: "Upload old links" takes a text or CSV file of old links or tokens; while Karvan checks it, a counter ("Checking 120 of 400") over a thin meter, read every second in the background; at the end the counts that are not zero.
  - New links: the done migrations from this old panel into this destination, each a download of its new links file.
- **Form** (`/links/new`, `/links/:id`): one column at most 720 wide. Name; the destination as radio cards (as in Connect); the old panel (kind, address, the address Karvan reaches it by, certificate fingerprint); what the site answers (old path, domains one per line, port); the certificate as radio cards (files on this server, upload files, paste, none) with the chosen way's fields and "Check certificate", which shows the same certificate facts before saving; None says the links go over plain HTTP. Lazy checks and "The old panel is plain HTTP" are switches. A field the server refuses gets its sentence under it. Cancel, then Save. A finished migration's "Set up old links" fills in the destination, the kind and the address. A pasted key is never shown again: on edit its field is empty and an empty field keeps the saved key.
- Delete asks for the site's name typed.

### Backups
- `/backups` follows Tables, without a search or a pager (backups are few and the server does not page them): refresh at the end of the header, one sentence under the title on what Safe Mode keeps.
- **Row:** the move (`MoveLabel`) with the admin as the link to the job's page, the panel with its logo, the database kind (from how it was made) with the backup id in mono under it, the size, when it was taken, the job's state, and a "..." menu (Put back, Delete). Phones get cards with the move first.
- **Panel control:** a panel Karvan cannot stop and start gets one warning above the table: its name, why (the reason's sentence) and how to allow it (the fix). Put back stays in the menu for its backups but is disabled, with the reason's short label next to it. A running job's backup can be neither put back nor deleted.
- **Put back** asks for the backup id typed and says what goes: everything written to the panel since the backup, by Karvan and by anyone else. Once the server holds the panel the live view opens (`/migrations/:id/live?after=<seq>`), showing the rewind scene, its log and its result; a restore that failed says so above the result.
- **Delete** asks for the backup id typed and says the backup cannot come back.
- **Empty:** an icon, "No backups yet", that Safe Mode takes one before each migration writes, and a way to its settings.

### Security settings
- Access, Two-step sign-in, Sessions and Audit log are tabs of Settings and follow Settings pages: setting rows in cards, one primary action per card.
- **Asking again:** a change to the login or the address opens a small dialog that asks for the current password, and the code when two-step sign-in is on. Its error sits above the fields; a lock says when to try again.
- **Shown once** (a new username and password, the new hidden path): a card with each value in mono, a Copy button per value, a warning that they are not shown again, and "I saved them" as the only way to close it. The values live in component state only (D10) and go when the card closes or the page is left.
- **Address change:** the waiting address sits in its own card with the time it is served until (a clock that counts down), the link that opens it in a new tab, and that nothing changes until it is confirmed from there. Opened from the new address, the same card has Confirm as its primary action. A new port shows the firewall and proxy warning next to the port field.
- **Two-step setup:** the QR code (dark on a white tile in both themes, drawn as one SVG path), the secret in groups of four in mono with Copy, then the 6-digit field and Turn on. Turning it off asks again with the password and a code.
- **Sessions:** a table, the current browser first with a "This browser" badge: browser and system in words, sign-in IP, signed in and last used. Sign out per row; "Sign out other sessions" in the header.
- **Audit log:** a table newest first with action and outcome filters above it: when, what (the action in words, the outcome as icon and word, the reason under it for a failure), who and IP. "Load more" under the table instead of a pager, since the log only grows.
### Sign-in
- Two halves on large screens. Start side on `bg-kumo-base`: wordmark at the top, then a centered form (title, username, password, one blue button) and the recovery hint with the `karvan reset -password` command.
- End side on `bg-kumo-brand`: language and theme buttons at the top, the "Marzban ⇄ PasarGuard" line, and one short line, then a spinning dotted globe.
- The globe acts out four scenes in turn (`globe-scenes.ts`): a move and a merge in each direction. Two panels appear on land, each as its logo on a white tile with its name under it; users travel as dots along a dashed arc. In a move the sending panel fades out once the last user arrives; in a merge both stay.
- The globe is drawn on a canvas from a land mask made by `scripts/brand/globe.mjs` (`globe-land.ts`, about 2 KB gzip), so no map code ships. It is a lazy chunk with its own budget, loaded after the form.
- Under 1024 px only the form side shows; the language and theme buttons move into its top bar.

### Behavior
- **Kumo's own rules** (kumo-ui.com/skill): content text is 14 (15 in Persian), only headings are larger; sentence case; no letter spacing changes; `font-semibold` for headings and `font-medium` for bold inline text, never `font-bold`; related text sits closer than the block it belongs to; a ring instead of a border next to a shadow; nested corners are concentric (outer radius = inner radius + padding); icons line up with the first line of text; inline mono text at about 0.9 em; a hairline under anything sticky; collapsing content keeps its size while it closes; dialogs are always rendered and opened with `open`, never mounted on demand.
- **Primary action:** one per area, `variant="primary"`. Everything else is secondary or ghost.
- **Loading:** skeletons shaped like the content, never spinners for content. Spinners only inside a button that is working.
- **Empty states:** an icon, what is missing, why, and one action (`components/feedback/EmptyState`: Kumo's Empty in shape, with the card radius and the heading size of section 3).
- **Errors:** say what happened and how to fix it, in plain words. Never show raw server errors. Errors do not apologize.
- **Problems Karvan found** (a panel finding, why it cannot stop a panel): what is wrong, then how to fix it, then the machine value it is about (a path, a revision, a table) when there is one. The raw error text behind it never shows.
- **Dangerous actions:** need the name typed to confirm (delete, restore, turning Safe Mode off).
- **Status badges:** icon + word + Kumo status color (`StatusBadge`). Kumo's warning text on its own tint is under 4.5:1, so a warning badge keeps the word in the default color and the icon in the warning color.
  - A migration's state uses the same pieces: done is success, failed is danger, running is info, interrupted is warning, drafts and rolled back jobs are neutral.
- **Touch:** every target is at least 44 × 44 px on touch screens.
- **Not yet built:** navigation items for later pages stay visible but inert with a "Soon" tag, so the structure can be judged early.

## 9. Writing

The voice is plain and natural, like a well-made Iranian product site: not robotic, not stiff, not salesy, and not chatty. Nobody reading it should think a machine wrote it.

- **Short.** Say it once. No extra explanation, no "coming in the next version" notes, nothing that repeats what the screen already shows. If a line can go, it goes.
- **No hype.** No big promises, no "seamless", no "without any hassle".
- A button says what happens: "Sign in", not "Submit". The same action keeps the same name in the toast that confirms it.
- Name things the way the operator does ("users", "panels"), not the way the code does.
- **Persian** is standard written Persian, kept simple, and written in Persian, not translated word for word. Polite plural verbs ("وارد کنید", "دوباره امتحان کنید"), written forms ("را", "نمیشود"), no slang. Common technical words stay (پنل، کانفیگ).
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
| Charts with their numbers in text and a table for screen readers | A chart as the only place a value lives |
| Hover colors that change at once | `transition-colors` on hover |
| Panel logos next to panel names and moves | A bare "PasarGuard" or "Marzban" in lists |
| Test at 360 and 1440 px, fa and en, dark and light, keyboard only | Shipping a page seen in one language and one theme |
