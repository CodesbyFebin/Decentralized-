# Decentralized.Host Command Centre

Operator console for Decentralized.Host: a React + Vite front end with a small
Express BFF (`server.ts`) that serves `/api/v1/*` from a demo platform store.

## Run locally

Prerequisites: Node.js 22+ (or Bun).

```bash
npm install          # or: bun install
npm run dev          # http://localhost:3000 (Vite middleware + API)
npm run lint         # tsc --noEmit
npm run build && NODE_ENV=production npm start
```

Set `GEMINI_API_KEY` in `.env.local` to enable model-backed RAG Copilot answers.

## Surfaces

| Route | What it shows |
| --- | --- |
| `/` | Dashboard: greeting, live globe, KPIs, resource usage, RAG Copilot, activity, health |
| `/deploy` | Universal Deploy: source → build → artifact → targets pipeline, strategies, KPIs, live deployment map, ownership mix, deployments table. `/deploy/new[:preset]` opens the deployment wizard |
| `/nodes` | Nodes & Compute: owned / community / DePIN fleet, contribution, node table, Add a Node (real `dh node invite` → `dh-noded --join-file` → `dh node approve` flow) |
| `/storage` | Storage & Data: capacity, replication, integrity, distributed storage map, storage nodes, Add Storage |

The holographic globe (`src/components/common/HoloGlobe.tsx`) is a dependency-free
canvas renderer: dotted continents, great-circle arcs with travelling pulses, node
markers and HTML callouts that track rotation and avoid overlapping. It pauses
off-screen and respects `prefers-reduced-motion`.

Fonts are bundled with `@fontsource-variable/*`, so the console loads nothing from
third-party hosts.

See [`design.md`](design.md) for the visual system and
[`docs/command-centre/reality-matrix.md`](docs/command-centre/reality-matrix.md)
for which numbers are live, derived or simulated.
