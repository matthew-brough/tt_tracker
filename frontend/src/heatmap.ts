import L from "leaflet";
import type { HexBin } from "./types";
import { gameToLatLng } from "./map";
import { HEATMAP_COLOR_STOPS } from "./config";

let heatGroup: L.LayerGroup;
let map: L.Map;

export function initHeatmap(m: L.Map) {
  map = m;
  heatGroup = L.layerGroup();
}

function hexVertices(cx: number, cy: number, size: number): L.LatLng[] {
  const verts: L.LatLng[] = [];
  for (let i = 0; i < 6; i++) {
    const angle = (Math.PI / 3) * i;
    verts.push(gameToLatLng(cx + size * Math.cos(angle), cy + size * Math.sin(angle)));
  }
  return verts;
}

function interpolateColor(t: number): string {
  const stops = HEATMAP_COLOR_STOPS;
  if (t <= 0) return stops[0][1];
  if (t >= 1) return stops[stops.length - 1][1];

  for (let i = 1; i < stops.length; i++) {
    if (t <= stops[i][0]) {
      const prev = stops[i - 1];
      const curr = stops[i];
      const ratio = (t - prev[0]) / (curr[0] - prev[0]);
      return lerpHex(prev[1], curr[1], ratio);
    }
  }
  return stops[stops.length - 1][1];
}

function lerpHex(a: string, b: string, t: number): string {
  const parse = (hex: string) => [
    parseInt(hex.slice(1, 3), 16),
    parseInt(hex.slice(3, 5), 16),
    parseInt(hex.slice(5, 7), 16),
  ];
  const ca = parse(a),
    cb = parse(b);
  const r = Math.round(ca[0] + (cb[0] - ca[0]) * t);
  const g = Math.round(ca[1] + (cb[1] - ca[1]) * t);
  const bl = Math.round(ca[2] + (cb[2] - ca[2]) * t);
  return `#${((1 << 24) | (r << 16) | (g << 8) | bl).toString(16).slice(1)}`;
}

export function updateHeatmap(bins: HexBin[]) {
  heatGroup.clearLayers();
  if (bins.length === 0) return;

  // Normalize by density (count / edge²) so different resolutions are comparable
  const densities = bins.map((b) => b.count / (b.edge * b.edge));
  const maxDensity = Math.max(...densities);

  // Sort coarse→fine so fine bins render on top
  const sorted = [...bins].sort((a, b) => b.edge - a.edge);

  for (const bin of sorted) {
    const density = bin.count / (bin.edge * bin.edge);
    const t = maxDensity > 0 ? density / maxDensity : 0;
    const color = interpolateColor(t);
    // Scale opacity: empty fill cells are subtle, dense cells are prominent
    const fillOpacity = 0.1 + 0.5 * t;
    const verts = hexVertices(bin.x, bin.y, bin.edge);
    const poly = L.polygon(verts, {
      color: color,
      fillColor: color,
      fillOpacity,
      weight: 1,
      opacity: 0.6,
    });
    poly.bindTooltip(`Count: ${bin.count}`);
    heatGroup.addLayer(poly);
  }
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
