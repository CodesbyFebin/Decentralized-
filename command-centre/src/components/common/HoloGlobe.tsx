import React, { useEffect, useMemo, useRef } from 'react';

/**
 * HoloGlobe — canvas-rendered holographic earth used across the command surfaces.
 *
 * Draws: atmosphere halo, dotted continents with "city light" clusters, a faint
 * graticule, great-circle arcs with travelling pulses, glowing node markers
 * (dot or isometric cube), and keeps HTML callout cards pinned to their marker
 * as the globe rotates. Pauses when off-screen and honours reduced motion.
 */

export interface GlobeMarker {
  id: string;
  lat: number;
  lng: number;
  color: string;
  kind?: 'dot' | 'cube';
  size?: number;
  /** HTML callout pinned to the marker. Positioned by the render loop. */
  callout?: React.ReactNode;
  /** Pixel offset of the callout anchor relative to the marker. */
  calloutOffset?: [number, number];
}

export interface GlobeArc {
  from: [number, number];
  to: [number, number];
  color: string;
}

interface Props {
  markers?: GlobeMarker[];
  arcs?: GlobeArc[];
  /** Longitude (deg) facing the viewer at mount. */
  focusLng?: number;
  /** Tilt toward the viewer (deg). Positive shows more of the northern hemisphere. */
  tilt?: number;
  /** Degrees per second of auto rotation. 0 disables. */
  speed?: number;
  /** Globe center as a fraction of the canvas box. */
  center?: [number, number];
  /** Radius as a fraction of min(width, height). */
  radius?: number;
  /** Rim light intensity 0..1 */
  glow?: number;
  className?: string;
  /** Dot density: smaller = denser. Degrees between samples. */
  resolution?: number;
  /** Pixels on the right kept clear of callouts (e.g. for an overlaid legend). */
  safeRight?: number;
}

/* Coarse continent outlines [lat, lng]. Low resolution on purpose: rendered as glowing dots. */
const LAND: [number, number][][] = [
  // North America
  [[71,-156],[70,-141],[69,-128],[70,-115],[68,-108],[72,-95],[69,-85],[64,-81],[62,-75],[60,-65],[55,-60],[52,-56],[47,-53],[45,-62],[44,-66],[41,-70],[39,-74],[35,-76],[32,-80],[30,-81],[25,-80],[27,-82],[30,-84],[30,-89],[29,-95],[26,-97],[22,-98],[19,-96],[18,-94],[21,-90],[21,-87],[16,-88],[15,-84],[11,-84],[9,-79],[8,-77],[8,-82],[13,-88],[15,-93],[17,-100],[20,-105],[23,-106],[24,-110],[28,-112],[31,-114],[31,-117],[34,-120],[38,-123],[42,-124],[48,-125],[54,-131],[58,-137],[60,-145],[59,-152],[57,-157],[55,-163],[58,-158],[60,-162],[63,-166],[66,-164],[68,-166]],
  // Greenland
  [[83,-35],[82,-55],[78,-72],[76,-68],[71,-55],[66,-53],[60,-44],[62,-41],[66,-36],[70,-24],[74,-19],[78,-19],[81,-15]],
  // South America
  [[12,-72],[11,-64],[9,-61],[6,-57],[4,-52],[1,-50],[-2,-44],[-5,-36],[-8,-35],[-13,-38],[-18,-39],[-23,-42],[-25,-48],[-29,-49],[-33,-53],[-35,-57],[-38,-57],[-41,-63],[-46,-67],[-50,-68],[-53,-69],[-55,-67],[-54,-72],[-50,-75],[-45,-74],[-38,-73],[-30,-71],[-22,-70],[-18,-70],[-15,-75],[-9,-79],[-5,-81],[-1,-80],[2,-79],[7,-77],[9,-76],[11,-75]],
  // Eurasia
  [[71,26],[70,33],[68,41],[67,44],[69,55],[72,68],[73,80],[76,95],[77,105],[74,113],[73,128],[72,142],[70,160],[67,176],[65,179],[64,178],[61,170],[60,164],[57,163],[54,160],[51,157],[55,155],[59,154],[59,143],[54,137],[50,140],[46,138],[43,133],[40,129],[38,128],[35,126],[37,122],[38,118],[35,119],[31,121],[28,121],[25,119],[23,117],[22,114],[21,110],[18,106],[17,107],[12,109],[10,106],[9,105],[11,103],[13,100],[10,99],[7,100],[3,101],[1,104],[4,103],[6,102],[8,98],[13,98],[16,97],[16,95],[19,94],[21,92],[22,90],[21,87],[20,86],[17,82],[15,80],[10,80],[8,77],[10,76],[14,74],[18,73],[21,72],[23,68],[25,66],[25,62],[26,57],[24,57],[24,56],[22,59],[18,56],[16,52],[13,45],[15,43],[20,41],[24,38],[28,35],[29,33],[30,32],[31,34],[34,35],[36,36],[37,32],[36,29],[37,27],[40,26],[41,29],[41,33],[42,36],[41,41],[43,40],[45,37],[47,39],[46,35],[45,33],[46,31],[45,29],[43,28],[42,28],[41,26],[40,23],[38,24],[37,22],[39,20],[42,19],[44,15],[46,13],[45,12],[43,14],[41,16],[40,18],[38,16],[40,15],[42,12],[44,9],[43,6],[43,3],[41,2],[39,0],[37,-1],[36,-5],[37,-7],[37,-9],[40,-9],[43,-9],[44,-2],[46,-1],[48,-4],[49,-1],[50,1],[51,3],[53,5],[54,8],[57,8],[57,10],[55,10],[55,13],[54,14],[55,19],[57,21],[59,23],[60,22],[61,21],[64,21],[66,24],[65,21],[63,18],[61,17],[59,18],[56,16],[56,13],[58,11],[59,10],[58,6],[59,5],[62,5],[64,10],[67,14],[69,16],[70,20]],
  // Great Britain
  [[58.6,-5],[57.5,-2],[56,-2.8],[55,-1.5],[53.5,0.2],[52.6,1.7],[51.2,1.4],[50.7,-1],[50.1,-5.4],[51.6,-5],[52.5,-4],[53.3,-4.6],[54.5,-3.4],[55.5,-5],[57.5,-6]],
  // Ireland
  [[55.3,-7.3],[54.3,-5.6],[52.2,-6.3],[51.5,-9.6],[52.8,-10],[54.3,-8.6]],
  // Iceland
  [[66.4,-23],[66.3,-15],[64.6,-13.6],[63.4,-18.8],[64,-22.6]],
  // Africa
  [[37,10],[37,3],[35,-2],[35.8,-6],[33,-8],[28,-13],[24,-16],[21,-17],[15,-17],[12,-16],[9,-13],[5,-8],[4,-3],[6,1],[6,4],[4,7],[4,9],[1,9.5],[-2,9],[-6,12],[-9,13],[-12,13.6],[-17,11.8],[-22,14],[-27,15],[-29,16.5],[-34,18.4],[-34.5,20],[-34,25],[-33,27],[-30,31],[-27,32.8],[-24,35.5],[-20,35],[-16,40],[-11,40],[-7,39],[-3,40],[1,42],[4,47],[8,50],[11.5,51],[11.5,49],[12,44],[15,40],[18,38],[22,36.7],[27,34],[30,32.5],[31.5,30],[31,25],[32.5,21],[31,18],[32.5,15],[33.5,11]],
  // Madagascar
  [[-12,49.3],[-15.5,50.2],[-20,48.8],[-25,47],[-25.2,44.2],[-21.5,43.5],[-16,44.5],[-13.5,48]],
  // Australia
  [[-11,132],[-12,136.8],[-15,135.5],[-17,140.5],[-11,142.5],[-15,145],[-19,146.5],[-23,150.8],[-28,153.5],[-33,152],[-37.5,150],[-39,146.5],[-38,141],[-35.5,138],[-33,137.5],[-35,135.5],[-32,132],[-31.5,127],[-33.8,123.5],[-35,117.8],[-34,115],[-31,115],[-26,113.5],[-22,114],[-20,118.5],[-17,122.5],[-14,126],[-14.5,129.5]],
  // Tasmania
  [[-40.8,144.7],[-41,148.2],[-43.5,147],[-43,145.2]],
  // New Zealand
  [[-34.5,172.8],[-37.5,176],[-41.5,175.5],[-40.5,173],[-43.5,172.8],[-46.6,169],[-45.5,166.8],[-42,171.5]],
  // Japan
  [[45.4,141.8],[43.3,145.5],[41.5,141],[39,142],[35.5,140.5],[34.5,137.5],[33.5,135.3],[34,131],[31,130.5],[33.2,129.5],[35.5,133],[36.7,136.8],[38,138.8],[40.5,140]],
  // Sumatra
  [[5.6,95.3],[3.5,99],[0.5,103],[-3,106],[-5.8,105.8],[-4,102],[-1,100],[2,97.8]],
  // Java
  [[-6,106],[-6.8,111],[-7.8,114.5],[-8.6,114],[-8,110],[-7.5,106.4]],
  // Borneo
  [[7,116.8],[5,119],[1,118.5],[-2,116.5],[-4,114.5],[-3.3,111],[-1,110],[1.5,109.5],[3,113],[5,115.5]],
  // Sulawesi (coarse)
  [[1.4,124.8],[-1,121],[-5.5,120.5],[-4,122.5],[-1,123],[0.5,120.5]],
  // New Guinea
  [[-0.8,131],[-2.5,134],[-2.6,141],[-5.5,146],[-8,147.5],[-10.5,150.5],[-9,147],[-8.2,143],[-9,140.5],[-7,138.5],[-4.5,136],[-3.8,132.8]],
  // Philippines (coarse)
  [[18.5,121],[14,124],[10,126],[6.5,126],[7,122],[10,122.5],[13,120.5],[16,120]],
  // Sri Lanka
  [[9.8,80.2],[8,81.9],[6,81.5],[6.2,80],[8.5,79.8]],
  // Cuba
  [[23,-84],[23.2,-80.5],[21.5,-76.5],[20,-74.2],[20,-77.5],[21.5,-80],[22,-83.5]],
  // Taiwan
  [[25.3,121.5],[22,120.8],[23,120.1],[24.5,120.7]],
  // Korea
  [[42.5,130.5],[39.5,128.2],[35.1,129.2],[34.7,126.4],[37.7,126.1],[39.8,124.3]]
];

/* Inland water carved out of the continent polygons above. */
const WATER: [number, number][][] = [
  [[63,-94],[60,-94],[57,-92],[55,-87],[52,-80],[56,-77],[60,-78],[62,-82],[64,-87]], // Hudson Bay
  [[47,49],[44,47],[40,50],[37,53],[40,54],[44,52],[46,53]], // Caspian Sea
  [[46.5,30],[44.5,29],[41.3,28.5],[41,33],[41.4,41],[43.5,40],[45.3,37],[46.5,32]] // Black Sea
];

function inPoly(lat: number, lng: number, poly: [number, number][]): boolean {
  let inside = false;
  for (let i = 0, j = poly.length - 1; i < poly.length; j = i++) {
    const [yi, xi] = poly[i];
    const [yj, xj] = poly[j];
    if (yi > lat !== yj > lat && lng < ((xj - xi) * (lat - yi)) / (yj - yi) + xi) inside = !inside;
  }
  return inside;
}

/* Deterministic pseudo random so city lights are stable between renders. */
function hash(a: number, b: number): number {
  const s = Math.sin(a * 127.1 + b * 311.7) * 43758.5453;
  return s - Math.floor(s);
}

interface LandDot { x: number; y: number; z: number; light: number; }

const landCache = new Map<number, LandDot[]>();

function buildLand(step: number): LandDot[] {
  const cached = landCache.get(step);
  if (cached) return cached;
  const dots: LandDot[] = [];
  for (let lat = -58; lat <= 82; lat += step) {
    const ring = Math.max(1, Math.round((360 / step) * Math.cos((lat * Math.PI) / 180)));
    for (let k = 0; k < ring; k++) {
      const lng = -180 + (k * 360) / ring;
      if (!LAND.some((p) => inPoly(lat, lng, p))) continue;
      if (WATER.some((p) => inPoly(lat, lng, p))) continue;
      const v = toVec(lat, lng);
      const r = hash(lat, lng);
      // ~14% of land dots become warm "city light" clusters.
      dots.push({ ...v, light: r > 0.86 ? 1 + r : r });
    }
  }
  landCache.set(step, dots);
  return dots;
}

function toVec(lat: number, lng: number) {
  const phi = (lat * Math.PI) / 180;
  const lam = (lng * Math.PI) / 180;
  return { x: Math.cos(phi) * Math.sin(lam), y: Math.sin(phi), z: Math.cos(phi) * Math.cos(lam) };
}

function slerp(a: ReturnType<typeof toVec>, b: ReturnType<typeof toVec>, t: number) {
  const dot = Math.min(1, Math.max(-1, a.x * b.x + a.y * b.y + a.z * b.z));
  const om = Math.acos(dot);
  if (om < 1e-4) return a;
  const s = Math.sin(om);
  const k1 = Math.sin((1 - t) * om) / s;
  const k2 = Math.sin(t * om) / s;
  return { x: a.x * k1 + b.x * k2, y: a.y * k1 + b.y * k2, z: a.z * k1 + b.z * k2 };
}

function hexA(hex: string, a: number): string {
  const h = hex.replace('#', '');
  const n = parseInt(h.length === 3 ? h.split('').map((c) => c + c).join('') : h, 16);
  return `rgba(${(n >> 16) & 255},${(n >> 8) & 255},${n & 255},${a})`;
}

export const HoloGlobe: React.FC<Props> = ({
  markers = [],
  arcs = [],
  focusLng = 20,
  tilt = 18,
  speed = 3,
  center = [0.5, 0.5],
  radius = 0.42,
  glow = 1,
  className = '',
  resolution = 1.6,
  safeRight = 0
}) => {
  const wrapRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const calloutRefs = useRef<Record<string, HTMLDivElement | null>>({});
  const land = useMemo(() => buildLand(resolution), [resolution]);

  // Latest props for the render loop without restarting it.
  const live = useRef({ markers, arcs, center, radius, glow, tilt, speed });
  live.current = { markers, arcs, center, radius, glow, tilt, speed };
  const safeRightRef = useRef(safeRight);
  safeRightRef.current = safeRight;

  useEffect(() => {
    const canvas = canvasRef.current;
    const wrap = wrapRef.current;
    if (!canvas || !wrap) return;
    const maybeCtx = canvas.getContext('2d');
    if (!maybeCtx) return;
    const ctx: CanvasRenderingContext2D = maybeCtx;

    const reduced = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
    let raf = 0;
    let visible = true;
    let w = 0;
    let h = 0;
    let rotation = (-focusLng * Math.PI) / 180;
    let last = performance.now();

    const resize = () => {
      const r = wrap.getBoundingClientRect();
      const dpr = Math.min(window.devicePixelRatio || 1, 2);
      w = r.width;
      h = r.height;
      canvas.width = Math.max(1, Math.round(w * dpr));
      canvas.height = Math.max(1, Math.round(h * dpr));
      canvas.style.width = `${w}px`;
      canvas.style.height = `${h}px`;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    };
    resize();
    const ro = new ResizeObserver(() => {
      resize();
      if (reduced) draw(performance.now());
    });
    ro.observe(wrap);
    const io = new IntersectionObserver(([e]) => {
      visible = e.isIntersecting;
      if (visible && !reduced) {
        last = performance.now();
        cancelAnimationFrame(raf);
        raf = requestAnimationFrame(frame);
      }
    });
    io.observe(wrap);

    function draw(now: number) {
      const { markers, arcs, center, radius, glow, tilt } = live.current;
      const R = Math.min(w, h) * radius;
      const cx = w * center[0];
      const cy = h * center[1];
      const tl = (tilt * Math.PI) / 180;
      const cosR = Math.cos(rotation), sinR = Math.sin(rotation);
      const cosT = Math.cos(tl), sinT = Math.sin(tl);

      const project = (v: { x: number; y: number; z: number }, lift = 1) => {
        // rotate around Y (spin) then X (tilt)
        const x1 = v.x * cosR + v.z * sinR;
        const z1 = -v.x * sinR + v.z * cosR;
        const y2 = v.y * cosT - z1 * sinT;
        const z2 = v.y * sinT + z1 * cosT;
        return { sx: cx + x1 * R * lift, sy: cy - y2 * R * lift, z: z2, x: x1 * lift, y: y2 * lift };
      };

      ctx.clearRect(0, 0, w, h);

      // Outer atmosphere halo
      const halo = ctx.createRadialGradient(cx, cy, R * 0.9, cx, cy, R * 1.32);
      halo.addColorStop(0, `rgba(32,221,247,${0.34 * glow})`);
      halo.addColorStop(0.35, `rgba(36,139,255,${0.16 * glow})`);
      halo.addColorStop(0.7, `rgba(124,77,255,${0.07 * glow})`);
      halo.addColorStop(1, 'rgba(124,77,255,0)');
      ctx.fillStyle = halo;
      ctx.beginPath();
      ctx.arc(cx, cy, R * 1.32, 0, Math.PI * 2);
      ctx.fill();

      // Ocean body
      const body = ctx.createRadialGradient(cx - R * 0.35, cy - R * 0.4, R * 0.1, cx, cy, R);
      body.addColorStop(0, '#0B2A5C');
      body.addColorStop(0.55, '#071A3C');
      body.addColorStop(1, '#030A1C');
      ctx.fillStyle = body;
      ctx.beginPath();
      ctx.arc(cx, cy, R, 0, Math.PI * 2);
      ctx.fill();

      // Graticule
      ctx.lineWidth = 0.6;
      ctx.strokeStyle = 'rgba(96,165,250,0.10)';
      for (let lat = -60; lat <= 60; lat += 30) {
        ctx.beginPath();
        let pen = false;
        for (let lng = -180; lng <= 180; lng += 6) {
          const p = project(toVec(lat, lng));
          if (p.z > 0) {
            pen ? ctx.lineTo(p.sx, p.sy) : ctx.moveTo(p.sx, p.sy);
            pen = true;
          } else pen = false;
        }
        ctx.stroke();
      }
      for (let lng = -180; lng < 180; lng += 30) {
        ctx.beginPath();
        let pen = false;
        for (let lat = -84; lat <= 84; lat += 6) {
          const p = project(toVec(lat, lng));
          if (p.z > 0) {
            pen ? ctx.lineTo(p.sx, p.sy) : ctx.moveTo(p.sx, p.sy);
            pen = true;
          } else pen = false;
        }
        ctx.stroke();
      }

      // Land dots
      const dotR = Math.max(0.75, R / 210);
      for (let i = 0; i < land.length; i++) {
        const d = land[i];
        const p = project(d);
        if (p.z <= 0.02) continue;
        const shade = 0.25 + 0.75 * p.z;
        if (d.light > 1) {
          ctx.fillStyle = `rgba(255,${190 + Math.round(40 * (d.light - 1))},120,${0.85 * shade})`;
          ctx.fillRect(p.sx - dotR, p.sy - dotR, dotR * 2.1, dotR * 2.1);
        } else {
          ctx.fillStyle = `rgba(${70 + Math.round(60 * d.light)},${170 + Math.round(60 * d.light)},255,${0.55 * shade})`;
          ctx.fillRect(p.sx - dotR * 0.8, p.sy - dotR * 0.8, dotR * 1.6, dotR * 1.6);
        }
      }

      // Terminator shading for depth
      const shade = ctx.createRadialGradient(cx - R * 0.3, cy - R * 0.35, R * 0.2, cx, cy, R * 1.02);
      shade.addColorStop(0, 'rgba(0,0,0,0)');
      shade.addColorStop(0.75, 'rgba(2,7,17,0.18)');
      shade.addColorStop(1, 'rgba(2,7,17,0.55)');
      ctx.fillStyle = shade;
      ctx.beginPath();
      ctx.arc(cx, cy, R, 0, Math.PI * 2);
      ctx.fill();

      // Rim light
      ctx.lineWidth = 1.6;
      const rim = ctx.createLinearGradient(cx - R, cy - R, cx + R, cy + R);
      rim.addColorStop(0, `rgba(32,221,247,${0.9 * glow})`);
      rim.addColorStop(0.5, `rgba(36,139,255,${0.55 * glow})`);
      rim.addColorStop(1, `rgba(168,85,247,${0.8 * glow})`);
      ctx.strokeStyle = rim;
      ctx.shadowColor = 'rgba(32,221,247,0.8)';
      ctx.shadowBlur = 18 * glow;
      ctx.beginPath();
      ctx.arc(cx, cy, R, 0, Math.PI * 2);
      ctx.stroke();
      ctx.shadowBlur = 0;

      // Arcs
      const t = now / 1000;
      ctx.lineCap = 'round';
      arcs.forEach((a, idx) => {
        const va = toVec(a.from[0], a.from[1]);
        const vb = toVec(a.to[0], a.to[1]);
        const dist = Math.acos(Math.min(1, Math.max(-1, va.x * vb.x + va.y * vb.y + va.z * vb.z)));
        const height = 0.08 + dist * 0.16;
        const N = 48;
        const pts = [] as { sx: number; sy: number; vis: boolean }[];
        for (let i = 0; i <= N; i++) {
          const u = i / N;
          const v = slerp(va, vb, u);
          const p = project(v, 1 + Math.sin(u * Math.PI) * height);
          const occluded = p.z < 0 && p.x * p.x + p.y * p.y < 1;
          pts.push({ sx: p.sx, sy: p.sy, vis: !occluded });
        }
        ctx.strokeStyle = hexA(a.color, 0.5);
        ctx.lineWidth = 1.2;
        ctx.shadowColor = a.color;
        ctx.shadowBlur = 8;
        ctx.beginPath();
        let pen = false;
        for (const p of pts) {
          if (p.vis) {
            pen ? ctx.lineTo(p.sx, p.sy) : ctx.moveTo(p.sx, p.sy);
            pen = true;
          } else pen = false;
        }
        ctx.stroke();
        // travelling pulse
        const head = ((t * 0.35 + idx * 0.37) % 1) * N;
        for (let k = 0; k < 8; k++) {
          const i = Math.floor(head) - k;
          if (i < 1 || !pts[i].vis || !pts[i - 1].vis) continue;
          ctx.strokeStyle = hexA(a.color, 0.95 * (1 - k / 8));
          ctx.lineWidth = 2.4 - k * 0.2;
          ctx.beginPath();
          ctx.moveTo(pts[i - 1].sx, pts[i - 1].sy);
          ctx.lineTo(pts[i].sx, pts[i].sy);
          ctx.stroke();
        }
        ctx.shadowBlur = 0;
      });

      // Markers + callouts. Callout boxes are laid out top-to-bottom and nudged apart when they overlap.
      const placed: { x: number; y: number; w: number; h: number }[] = [];
      const order = markers
        .map((m, idx) => ({ m, idx, p: project(toVec(m.lat, m.lng), 1.01) }))
        .sort((a, b) => a.p.sy - b.p.sy);
      const calloutPos = new Map<string, { x: number; y: number }>();
      for (const { m, p } of order) {
        const el = calloutRefs.current[m.id];
        if (!el || p.z <= 0) continue;
        const [ox, oy] = m.calloutOffset ?? [14, -18];
        const cw = el.offsetWidth;
        const ch = el.offsetHeight;
        // Flip to the left of the marker when the card would run past the right edge / safe zone.
        const flip = p.sx + ox + cw > w - 8 - safeRightRef.current;
        let x = Math.max(4, flip ? p.sx - ox - cw : p.sx + ox);
        let y = p.sy + oy - ch / 2;
        for (const r of placed) {
          if (x < r.x + r.w && x + cw > r.x && y < r.y + r.h + 4 && y + ch > r.y) y = r.y + r.h + 4;
        }
        placed.push({ x, y, w: cw, h: ch });
        calloutPos.set(m.id, { x, y });
      }

      markers.forEach((m, idx) => {
        const p = project(toVec(m.lat, m.lng), 1.01);
        const el = calloutRefs.current[m.id];
        const front = p.z > 0;
        const pos = calloutPos.get(m.id);
        if (el && pos) {
          el.style.transform = `translate(${Math.round(pos.x)}px, ${Math.round(pos.y)}px)`;
          el.style.opacity = front ? String(Math.min(1, p.z * 8)) : '0';
          el.style.pointerEvents = front ? 'auto' : 'none';
        }
        if (!front) return;
        const size = (m.size ?? 1) * Math.max(3, R / 70);
        const pulse = 0.5 + 0.5 * Math.sin(t * 2.2 + idx);
        // pulse ring
        ctx.strokeStyle = hexA(m.color, 0.45 * (1 - pulse));
        ctx.lineWidth = 1;
        ctx.beginPath();
        ctx.arc(p.sx, p.sy, size * (1.6 + pulse * 1.6), 0, Math.PI * 2);
        ctx.stroke();
        ctx.shadowColor = m.color;
        ctx.shadowBlur = 16;
        if (m.kind === 'cube') drawCube(ctx, p.sx, p.sy, size * 1.25, m.color);
        else {
          ctx.fillStyle = m.color;
          ctx.beginPath();
          ctx.arc(p.sx, p.sy, size * 0.62, 0, Math.PI * 2);
          ctx.fill();
          ctx.fillStyle = '#fff';
          ctx.beginPath();
          ctx.arc(p.sx, p.sy, size * 0.25, 0, Math.PI * 2);
          ctx.fill();
        }
        ctx.shadowBlur = 0;
      });
    }

    function frame(now: number) {
      const dt = Math.min(0.05, (now - last) / 1000);
      last = now;
      rotation -= ((live.current.speed * Math.PI) / 180) * dt;
      draw(now);
      if (visible) raf = requestAnimationFrame(frame);
    }

    if (reduced) draw(performance.now());
    else raf = requestAnimationFrame(frame);

    return () => {
      cancelAnimationFrame(raf);
      ro.disconnect();
      io.disconnect();
    };
  }, [land, focusLng]);

  return (
    <div ref={wrapRef} className={`${/(^|\s)!?absolute(\s|$)/.test(className) ? '' : 'relative '}overflow-hidden ${className}`}>
      <canvas ref={canvasRef} className="absolute inset-0 block" aria-hidden="true" />
      {markers.map((m) =>
        m.callout ? (
          <div
            key={m.id}
            ref={(el) => {
              calloutRefs.current[m.id] = el;
            }}
            className="absolute left-0 top-0 transition-opacity duration-300 will-change-transform"
            style={{ opacity: 0 }}
          >
            {m.callout}
          </div>
        ) : null
      )}
    </div>
  );
};

function drawCube(ctx: CanvasRenderingContext2D, x: number, y: number, s: number, color: string) {
  const h = s * 0.5;
  const top = [[x, y - s], [x + s * 0.87, y - h], [x, y], [x - s * 0.87, y - h]];
  const left = [[x - s * 0.87, y - h], [x, y], [x, y + s], [x - s * 0.87, y + h]];
  const right = [[x + s * 0.87, y - h], [x, y], [x, y + s], [x + s * 0.87, y + h]];
  const face = (pts: number[][], a: number) => {
    ctx.fillStyle = hexA(color, a);
    ctx.beginPath();
    pts.forEach(([px, py], i) => (i ? ctx.lineTo(px, py) : ctx.moveTo(px, py)));
    ctx.closePath();
    ctx.fill();
  };
  face(top, 0.95);
  face(left, 0.55);
  face(right, 0.75);
  ctx.strokeStyle = 'rgba(255,255,255,0.85)';
  ctx.lineWidth = 0.8;
  ctx.beginPath();
  [...top, top[0]].forEach(([px, py], i) => (i ? ctx.lineTo(px, py) : ctx.moveTo(px, py)));
  ctx.moveTo(x, y);
  ctx.lineTo(x, y + s);
  ctx.stroke();
}
