# 0009 — Command Centre: a built console with a BFF, alongside the embedded console

**Status:** proposed (needs a decision; conflicts with 0007)

## Context

ADR 0007 (accepted) makes the operator console plain ES modules embedded in
`dh-control`: no bundler, no npm dependencies, no external fonts, so the
shipped files are the auditable source.

`command-centre/` is a second console: React + Vite, npm dependencies, and a
Node BFF (`server.ts`) that talks to `dh-control` over its operator API. It
keeps most of 0007's intent:

| 0007 property | Command Centre |
|---|---|
| No third-party loads, `default-src 'self'` | yes: fonts bundled via `@fontsource`, CSP `'self'` in production, no analytics, model use opt-in |
| Escapes every host-supplied value | yes: React escapes; no `dangerouslySetInnerHTML` |
| No optimistic state, stale views marked | yes: mutations succeed only on the committed `Result`; failed refreshes desaturate data under a stale banner |
| No build step; shipped files are the source | **no**: bundle built by Vite |
| No npm dependencies | **no**: see `command-centre/package.json` / `bun.lock` |
| Embedded in `dh-control` | **no**: a separate process in front of the operator API |

## Options

1. **Keep both.** The embedded console stays the default, audited surface;
   the Command Centre is an optional, separately deployed console for teams
   that want it. Amend 0007 to scope it to the embedded console.
2. **Port the Command Centre into `web/dist`** as plain ES modules (no React,
   no bundler), reusing its information architecture, normalized model and
   visual system. Keeps 0007 intact; larger effort.
3. **Replace the embedded console** with the Command Centre. Supersedes 0007;
   the project loses the "shipped files are the source" property.

## Recommendation

Option 1 now, with option 2 as the path if the Command Centre becomes the
primary console. Whatever is chosen, the BFF must stay an adapter: it holds no
operational state and never falls back to non-backend data.
