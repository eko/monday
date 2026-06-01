---
version: alpha
spec: https://github.com/google-labs-code/design.md
name: txeh-docs
description: Local design adoption record for the txeh.txn2.com documentation site. References txn2/www DESIGN.md as the canonical visual identity for tokens, typography, components, voice, and accessibility rules. Records only the decisions and MkDocs Material learnings that the canonical does not cover.
upstream:
  design: https://github.com/txn2/www/blob/master/DESIGN.md
  tokens: https://github.com/txn2/www/blob/master/tokens.json
adoption: token-alignment
stack:
  generator: MkDocs
  theme: Material for MkDocs
  templates: docs/overrides/
  styles: docs/stylesheets/extra.css
---

## What is canonical

The canonical visual identity for txn2 lives in [`txn2/www/DESIGN.md`](https://github.com/txn2/www/blob/master/DESIGN.md) with tokens in [`txn2/www/tokens.json`](https://github.com/txn2/www/blob/master/tokens.json). This file defers to those for everything below. If a value here disagrees with upstream, upstream wins.

| Concern              | Source of truth |
|----------------------|-----------------|
| Color palette        | upstream `tokens.json` `color.*` |
| Typography stack     | upstream `tokens.json` `font.*` |
| Type scale           | upstream `DESIGN.md` Typography table |
| Spacing / measure    | upstream `tokens.json` `size.*` |
| Component contracts  | upstream `DESIGN.md` Components |
| Voice / copy rules   | upstream `DESIGN.md` Voice and Copy |
| Accessibility rules  | upstream `DESIGN.md` Do's and Don'ts |
| Mermaid theme        | upstream `DESIGN.md` `mcp__card--feature` block |

Tokens are mirrored as CSS custom properties in `docs/stylesheets/extra.css` `:root`. They are duplicated for runtime use, not as a divergence point. When upstream changes a token, update the value in `extra.css` and ship.

## Adoption level: token alignment

Per the upstream downstream contract, three levels are valid:

1. Reference. Link to upstream, no visual changes.
2. Token alignment. Keep MkDocs Material, re-skin via `extra.css` against upstream tokens.
3. Full re-skin. Replace MkDocs Material with custom layouts.

txeh runs at **level 2**, matching its sister project kubefwd. The site keeps Material's instant nav, search, sidebar, version selector, code copy, and content extensions. The visual layer is replaced. The homepage is a custom Material template that takes over `block header`, `block container`, and `block footer` for full-bleed treatment.

## File map

| Path | Role |
|------|------|
| `mkdocs.yml`                   | Single dark `slate` palette. `font: false` so CSS loads the upstream Google Fonts URL with trimmed axes. |
| `docs/index.md`                | Stub front matter with `template: home.html`. All homepage HTML lives in the template. |
| `docs/overrides/main.html`     | Adds the upstream Google Fonts `<link>` plus OG and Twitter meta. Inherited by every page. |
| `docs/overrides/home.html`     | Custom homepage template. Overrides `block header` (rail), `block tabs` (empty), `block container` (page--home shell with hero, sections, flagship cards, stack, coda), `block footer` (home-footer). |
| `docs/overrides/404.html`      | Restyled not-found page. Inherits `main.html`, uses `.page--home` shell so the rail and footer match. |
| `docs/stylesheets/extra.css`   | All design rules. Two halves: homepage components scoped under `.page--home`, and Material chrome restyle for inner pages via `[data-md-color-scheme="slate"]` variable overrides. |

## Project-specific components

Components ported from upstream verbatim, with txeh content:

- `.rail` (replaces Material `.md-header` on the homepage). Brand links to `./`. Live UTC clock in meta. txn2.com link in meta as `part of <em class="serif">txn2</em> ↗`.
- `.hero` with three Fraunces rows (txeh / hosts file / management.).
- `.section`, `.section__index`, `.section__title`.
- `.flagship__card`. Two cards: a CLI demo card (`add`, `remove`, `list`, `show`) and a Go library card (`NewHostsDefault`, `AddHost`, `Save`). Top accent line animates on hover per upstream spec.
- `.terminal`, `.terminal__bar`, `.terminal__body` with `.t-prompt`, `.t-ok`, `.t-mute` classes. The only block with shadow.
- `.stack`, `.stack__row`. Each row links to a real anchor in the docs (cli/, library/, api/, troubleshooting/).
- `.coda` and `.home-footer` (renamed from upstream `.footer` to avoid collision with markdown that uses `class="footer"`; see kubefwd Learning #5).

Components from upstream **not used** here, with reason:

- `.mcp__card--feature` and the MCP grid. txeh is a single library, not an MCP catalog.
- The 5-column footer's `sponsors / craig` columns. The txeh home-footer has `about / docs / interfaces / code / txn2 / org` columns instead, since this site is project-scoped.

Custom additions specific to txeh:

- `home-footer__col--meta` includes a `txn2 / org` panel that backlinks to txn2.com explicitly. Per the org-wide rule that every sister project must clearly link home.
- The CLI flagship card's terminal demo uses real `txeh add`, `txeh list ip`, and `txeh remove host` invocations against `127.0.0.1`. The library card shows the `NewHostsDefault` / `AddHost` / `Save` path.

## MkDocs Material learnings

The Material learnings list applies to every MkDocs Material project re-skinned to the txn2 identity. txeh inherits them all from kubefwd. Read the full set in [`txn2/kubefwd/DESIGN.md`](https://github.com/txn2/kubefwd/blob/master/DESIGN.md) "MkDocs Material learnings". Brief summary so this file remains useful in isolation:

1. Override the homepage via a separate template, not via CSS hacks.
2. Re-skin inner pages via Material variable overrides on `[data-md-color-scheme="slate"]`.
3. `font: false` to load fonts directly from CSS.
4. Scope every homepage component class under `.page--home`.
5. Rename `.footer` to `.home-footer` to avoid collision.
6. `h3` and `h4` are technical reference, not display type. Switch them to Instrument Sans bold; flip any heading containing inline code to JetBrains Mono via `:has(code)` with a `@supports not selector(:has(*))` fallback.
7. Tabbed content nests boxes by default. Strip `.tabbed-set` background and border, keep only the label underline.
8. Mermaid via Material's `--md-mermaid-*` CSS variables, not via separate init.
9. Guard inline scripts against `navigation.instant` rehydration. The live UTC clock uses a `window.__txehClock` sentinel.
10. Drop the light/dark toggle. Single `scheme: slate`.
11. Atmospheric overlays at low z-index (grain and vignette at z-index 1, below rail at z-50).
12. Hugo-only token compilation does not exist in MkDocs. Token sync is a manual edit to `extra.css`.

## Voice and copy

Defers to upstream. Briefly:

- No em-dashes (U+2014) or en-dashes (U+2013) anywhere, including code comments and template comments. Use commas, periods, colons, parentheses, slashes, hyphens.
- No AI-tell vocabulary: `seamless`, `leverage`, `comprehensive`, `robust`, `delve`, `unleash`, `elevate`, `embark`, `tapestry`, `not just X but Y`, `as an AI`, `let me X`.
- Sentence case for body. Lowercase for rail and label text. Title case rare.
- Section indices: `§ 01 / title` with slash, never an em-dash.
- Year ranges use a hyphen: `2018-2026`.
- Verify before commit: `grep -RE "—|–" docs/ mkdocs.yml`.

## Updating

When the upstream `txn2/www/DESIGN.md` or `tokens.json` changes:

1. Read the upstream diff. Identify which tokens, components, or rules changed.
2. Update the matching CSS variables in `docs/stylesheets/extra.css` `:root`.
3. If a component contract changed (padding, border, hover behavior), update the homepage template in `docs/overrides/home.html` and the matching CSS rules.
4. Update this file's File map / Project-specific components sections if a new component is added or removed.
5. Run `mkdocs build --strict` and verify in the browser before committing. Verify the home page hero, flagship cards, terminal, stack, coda, and home-footer. Verify an inner page (`/cli/`, `/library/`) still inherits the look.

Keep this file thin. If a section grows past 30 lines, ask whether it belongs upstream instead.
