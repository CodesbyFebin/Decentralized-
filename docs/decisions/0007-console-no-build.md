# 0007 — Console as plain ES modules, embedded, no third parties

**Status:** accepted

The operator console is plain ES modules and CSS in `web/dist`, embedded into
`dh-control`. It has no bundler, no npm dependencies and no external fonts.
It is served under `default-src 'self'` and revalidated on every load.

**Why.** "No mandatory SaaS dependency", "no phone-home" and "self-hosted by
default" apply to the console too. With no build step, the shipped files are
the source, which can be audited as-is.

**Consequences.**
- Rendering escapes every interpolated value, because hosts supply names and
  reasons.
- The console never updates state optimistically and marks stale views
  visibly.
- `DH_CONSOLE_DIR` serves the files from disk during development.
