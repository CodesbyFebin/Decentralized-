# Decentralized.Host Command Centre
## DESIGN.md — Product UI Architecture & Visual System

Version: 1.0  
Design Language: Spatial Sovereign Glass  
Product: Decentralized.Host Command Centre  
Primary Experience: Desktop infrastructure control plane  
Secondary: Tablet / Mobile operational console

---

# 1. DESIGN MISSION

Decentralized.Host must feel like an advanced infrastructure command centre rather than a conventional SaaS admin dashboard.

The visual language combines:
- Apple-grade interface discipline
- Spatial glass interfaces
- Deep-space infrastructure visualization
- Holographic distributed-network graphics
- Modern developer tooling
- Subtle alien/futuristic intelligence
- Production control-plane usability

The target emotional response is:
> “This looks like infrastructure software from the future, but I immediately understand how to operate it.”

The design must remain operational:
- Visual spectacle is concentrated around infrastructure topology, the dashboard, node network, deployment visualization, and Copilot.
- Tables, settings, logs, forms, and configuration surfaces become calmer and high-contrast.

---

# 2. CORE DESIGN PRINCIPLES

## 2.1 Spatial, Not Flat
The UI exists on several perceived depth layers:
- Layer 0: Deep-space cosmic environment
- Layer 1: Planetary/network topology visualization
- Layer 2: Ambient lighting and rim halos
- Layer 3: Application shell and navigation rails
- Layer 4: Glass panels (primary, secondary, floating)
- Layer 5: Interactive controls and input fields
- Layer 6: Dialogs, inspectors, and command palette

## 2.2 Futuristic But Usable
Users must immediately recognize standard infrastructure workflows: navigation, buttons, forms, tables, filters, search, dialogs, warnings, and deployment states without novel interaction confusion.

## 2.3 Operational Truth Controls Visual State
Visual confidence must be supported by backend state:
- LIVE / VERIFIED: Luminous cyan/violet, verified cryptographic seal
- HEALTHY / RUNNING: Emerald (#10D981) / Cyan (#20DDF7)
- DEGRADED: Amber (#F59E0B / #FBBF24)
- FAILED / OFFLINE: Coral / Red (#EF4444 / #FB4F64)
- UNKNOWN: Neutral slate (#64748B)
- STALE: Desaturated, visible last observation timestamp

---

# 3. COLOR SYSTEM & TOKENS

## 3.1 Cosmic Space Base
```css
--space-000: #01040B;
--space-050: #020711;
--space-100: #040B18;
--space-150: #061126;
--space-200: #08162C;
--space-250: #0A1B35;
--space-300: #0D2342;
```

## 3.2 Primary Spectrum & Gradient
```css
--cyan:     #20DDF7;
--electric: #248BFF;
--indigo:   #5965FF;
--violet:   #A855F7;

--gradient-primary: linear-gradient(90deg, #20DDF7 0%, #248BFF 35%, #5965FF 68%, #A855F7 100%);
--btn-primary: linear-gradient(90deg, #20DDF7 0%, #248BFF 42%, #7C4DFF 72%, #A855F7 100%);
```

## 3.3 Semantic Colors
```css
--success:  #10D981;
--online:   #06F6D0;
--warning:  #F59E0B;
--degraded: #FBBF24;
--error:    #EF4444;
--offline:  #FB4F64;
--info:     #38BDF8;
--unknown:  #64748B;
```

---

# 4. GLASS ARCHITECTURE

- **Primary Panel Glass**:
  `background: rgba(8, 20, 42, 0.62); backdrop-filter: blur(24px) saturate(135%); border: 1px solid rgba(125, 190, 255, 0.16); box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.07), 0 8px 32px rgba(36, 139, 255, 0.15); border-radius: 24px;`
- **Secondary Glass**:
  `background: rgba(8, 22, 45, 0.48); backdrop-filter: blur(18px) saturate(120%); border: 1px solid rgba(125, 190, 255, 0.10); border-radius: 20px;`
- **Floating Major Surfaces**:
  `background: rgba(8, 20, 42, 0.84); backdrop-filter: blur(32px) saturate(150%); border: 1px solid rgba(130, 200, 255, 0.22); border-radius: 24px;`

---

# 5. DASHBOARD & TOPOLOGY ARCHITECTURE (Exact Clone of Reference Image)

1. **Top Command Bar**:
   - Left brand header with luminous 3D cube logo, `Decentralized.Host`, and tagline `YOUR DATA. YOUR RULES.`
   - Center omnibar: `Search apps, nodes, domains, logs, or ask the AI copilot...` with `⌘ K` indicator.
   - Right cluster: Day/Night toggle, notifications with ping indicator, user avatar badge (`Febin Francis / Owner`).
2. **Hero Spatial Section**:
   - Date banner (`Wednesday, Sep 24, 2026`)
   - Large greeting: `Welcome back, Febin 👋`
   - Dynamically derived subtitle: `Your decentralized infrastructure is running smoothly.`
   - Quick Action Glass Pills: `Deploy App`, `Add Domain`, `Manage Nodes`, `View Logs`, `Ask Copilot →`
   - Embedded 3D earth rim with region floating tags (`North America`, `Europe`, `Asia`, `Africa`).
   - Global Network summary pill: `8 / 10 nodes online` with `View Nodes →`.
3. **Metric Cards Row**:
   - `Applications`: 12 (↑ +20%) with cyan sparkline
   - `Nodes Online`: 8 / 10 (↑ +12%) with emerald sparkline
   - `Storage Used`: 428 GB (↑ +18%) with violet sparkline
   - `Domains`: 8 (↑ +14%) with amber sparkline
   - `SSL Certificates`: 6 (↑ +33%) with electric blue sparkline
4. **Middle Two-Column Grid**:
   - **Node Network (Holographic Map)**: Live interactive 3D globe with glowing arcs, region pills, and status counter strip (`Online: 8`, `Degraded: 1`, `Offline: 1`, `Unknown: 0`).
   - **Resource Usage**: Circular donut progress gauges for `CPU (28%)`, `Memory (46%)`, `Storage (42%)`, `Network (52%)` + 24h wave chart with cyan, violet, and electric lines.
   - **RAG Copilot Floating Card**: 3D luminous AI humanoid orb with `Ask a question`, `Diagnose an issue`, `Plan an improvement`, `Execute an action` modes and `Start a conversation →` gradient button.
5. **Bottom Three-Column Grid**:
   - **Recent Activity**: Live event feed with success/online/pending badges and timestamps.
   - **Top Applications**: Data table listing domains (`agentswarm.in`, `ewastekochi.com`, etc.), `Running` status badges, traffic, and last deploy.
   - **System Health**: Health cluster (`API Server`, `Database`, `Node Agents`, `Storage`, `Security`) with live status beacons.
