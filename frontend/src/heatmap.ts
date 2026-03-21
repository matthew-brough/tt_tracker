import L from "leaflet";
import type { HexBin } from "./types";
import { gameToLatLng } from "./map";
import { HEATMAP_COLOR_STOPS } from "./config";

/* ---------- custom SVG renderer: re-applies gradient fill on style updates ---------- */

const HeatSVG = L.SVG.extend({
  _updateStyle(layer: any) {
    (L.SVG.prototype as any)._updateStyle.call(this, layer);
    const gid = layer.options?._gradientId;
    if (layer._path && gid && this._container?.querySelector(`#${gid}`)) {
      layer._path.setAttribute("fill", `url(#${gid})`);
      layer._path.setAttribute("fill-opacity", "1");
    }
  },
});

let heatRenderer: L.Renderer;
let heatGroup: L.LayerGroup;
let map: L.Map;

export function initHeatmap(m: L.Map) {
  map = m;
  heatRenderer = new (HeatSVG as any)({ padding: 0.5 });
  heatGroup = L.layerGroup();
}

/* ---------- geometry ---------- */

function hexVertices(cx: number, cy: number, size: number): L.LatLng[] {
  const verts: L.LatLng[] = [];
  for (let i = 0; i < 6; i++) {
    const angle = (Math.PI / 3) * i;
    verts.push(
      gameToLatLng(cx + size * Math.cos(angle), cy + size * Math.sin(angle)),
    );
  }
  return verts;
}

/* ---------- colour helpers ---------- */

function parseHex(hex: string): [number, number, number] {
  return [
    parseInt(hex.slice(1, 3), 16),
    parseInt(hex.slice(3, 5), 16),
    parseInt(hex.slice(5, 7), 16),
  ];
}

function toHex(r: number, g: number, b: number): string {
  return `#${((1 << 24) | (r << 16) | (g << 8) | b).toString(16).slice(1)}`;
}

function lerpHex(a: string, b: string, t: number): string {
  const ca = parseHex(a),
    cb = parseHex(b);
  return toHex(
    Math.round(ca[0] + (cb[0] - ca[0]) * t),
    Math.round(ca[1] + (cb[1] - ca[1]) * t),
    Math.round(ca[2] + (cb[2] - ca[2]) * t),
  );
}

function avgColor(colors: string[]): string {
  if (colors.length === 0) return "#000000";
  let r = 0,
    g = 0,
    b = 0;
  for (const c of colors) {
    const [cr, cg, cb] = parseHex(c);
    r += cr;
    g += cg;
    b += cb;
  }
  const n = colors.length;
  return toHex(Math.round(r / n), Math.round(g / n), Math.round(b / n));
}

function interpolateColor(t: number): string {
  const stops = HEATMAP_COLOR_STOPS;
  if (t <= 0) return stops[0][1];
  if (t >= 1) return stops[stops.length - 1][1];
  for (let i = 1; i < stops.length; i++) {
    if (t <= stops[i][0]) {
      const prev = stops[i - 1],
        curr = stops[i];
      return lerpHex(prev[1], curr[1], (t - prev[0]) / (curr[0] - prev[0]));
    }
  }
  return stops[stops.length - 1][1];
}

/* ---------- grid-accelerated neighbour lookup ---------- */

function findNeighborColors(
  bins: HexBin[],
  colors: string[],
): Map<number, string[]> {
  const cellSize = bins.length > 0 ? bins[0].edge * 3 : 100;
  const grid = new Map<string, number[]>();

  for (let i = 0; i < bins.length; i++) {
    const key = `${Math.floor(bins[i].x / cellSize)},${Math.floor(bins[i].y / cellSize)}`;
    let arr = grid.get(key);
    if (!arr) {
      arr = [];
      grid.set(key, arr);
    }
    arr.push(i);
  }

  const result = new Map<number, string[]>();
  for (let i = 0; i < bins.length; i++) {
    const gx = Math.floor(bins[i].x / cellSize);
    const gy = Math.floor(bins[i].y / cellSize);
    const thresh2 = (bins[i].edge * 2.2) ** 2;
    const nbrs: string[] = [];
    for (let dx = -1; dx <= 1; dx++) {
      for (let dy = -1; dy <= 1; dy++) {
        const cell = grid.get(`${gx + dx},${gy + dy}`);
        if (!cell) continue;
        for (const j of cell) {
          if (j === i) continue;
          const ddx = bins[i].x - bins[j].x;
          const ddy = bins[i].y - bins[j].y;
          if (ddx * ddx + ddy * ddy <= thresh2) nbrs.push(colors[j]);
        }
      }
    }
    result.set(i, nbrs);
  }
  return result;
}

/* ---------- SVG gradient management ---------- */

function clearGradients() {
  const svg = (heatRenderer as any)._container as SVGElement | undefined;
  if (!svg) return;
  svg.querySelectorAll("defs [id^='hm-']").forEach((el) => el.remove());
}

interface GradDef {
  id: string;
  center: string;
  edge: string;
  centerOpacity: number;
  edgeOpacity: number;
}

function injectGradients(gradients: GradDef[]) {
  const svg = (heatRenderer as any)._container as SVGElement;
  if (!svg) return;

  let defs = svg.querySelector("defs");
  if (!defs) {
    defs = document.createElementNS("http://www.w3.org/2000/svg", "defs");
    svg.insertBefore(defs, svg.firstChild);
  }

  const ns = "http://www.w3.org/2000/svg";
  for (const g of gradients) {
    const grad = document.createElementNS(ns, "radialGradient");
    grad.setAttribute("id", g.id);
    grad.setAttribute("gradientUnits", "objectBoundingBox");
    grad.setAttribute("cx", "0.5");
    grad.setAttribute("cy", "0.5");
    grad.setAttribute("r", "0.7");

    const s1 = document.createElementNS(ns, "stop");
    s1.setAttribute("offset", "0%");
    s1.setAttribute("stop-color", g.center);
    s1.setAttribute("stop-opacity", String(g.centerOpacity));

    const s2 = document.createElementNS(ns, "stop");
    s2.setAttribute("offset", "60%");
    s2.setAttribute("stop-color", g.center);
    s2.setAttribute("stop-opacity", String(g.centerOpacity * 0.85));

    const s3 = document.createElementNS(ns, "stop");
    s3.setAttribute("offset", "100%");
    s3.setAttribute("stop-color", g.edge);
    s3.setAttribute("stop-opacity", String(g.edgeOpacity));

    grad.appendChild(s1);
    grad.appendChild(s2);
    grad.appendChild(s3);
    defs.appendChild(grad);
  }
}

/* ---------- main update ---------- */

export function updateHeatmap(bins: HexBin[]) {
  heatGroup.clearLayers();
  clearGradients();
  if (bins.length === 0) return;

  const densities = bins.map((b) => b.count / (b.edge * b.edge));
  const maxDensity = Math.max(...densities);

  // Per-bin colour and intensity
  const colors: string[] = [];
  const ts: number[] = [];
  for (let i = 0; i < bins.length; i++) {
    const t = maxDensity > 0 ? densities[i] / maxDensity : 0;
    ts.push(t);
    colors.push(interpolateColor(t));
  }

  const neighborMap = findNeighborColors(bins, colors);

  // Sort coarse→fine so fine bins render on top
  const order = bins
    .map((_, i) => i)
    .sort((a, b) => bins[b].edge - bins[a].edge);

  const gradDefs: GradDef[] = [];

  for (const i of order) {
    const bin = bins[i];
    const t = ts[i];
    const centerColor = colors[i];
    const nbrs = neighborMap.get(i)!;
    const edgeColor =
      nbrs.length > 0 ? lerpHex(centerColor, avgColor(nbrs), 0.5) : centerColor;

    const centerOpacity = 0.15 + 0.55 * t;
    const edgeOpacity = Math.max(0.05, centerOpacity * 0.4);
    const gid = `hm-${i}`;

    gradDefs.push({
      id: gid,
      center: centerColor,
      edge: edgeColor,
      centerOpacity,
      edgeOpacity,
    });

    const verts = hexVertices(bin.x, bin.y, bin.edge);
    const poly = L.polygon(verts, {
      renderer: heatRenderer,
      color: edgeColor,
      weight: 0.5,
      opacity: 0.2,
      fillColor: centerColor,
      fillOpacity: centerOpacity,
      _gradientId: gid,
    } as any);
    poly.bindTooltip(`Count: ${bin.count}`);
    heatGroup.addLayer(poly);
  }

  // Inject gradient defs after the SVG paths are in the DOM
  requestAnimationFrame(() => {
    injectGradients(gradDefs);
    // Force-apply gradient fills on all paths
    heatGroup.eachLayer((layer: any) => {
      if (layer._path && layer.options._gradientId) {
        layer._path.setAttribute(
          "fill",
          `url(#${layer.options._gradientId})`,
        );
        layer._path.setAttribute("fill-opacity", "1");
      }
    });
  });
}

export function setHeatmapVisible(visible: boolean) {
  if (visible) {
    heatGroup.addTo(map);
  } else {
    heatGroup.remove();
  }
}

export function isHeatmapVisible(): boolean {
  return map.hasLayer(heatGroup);
}
