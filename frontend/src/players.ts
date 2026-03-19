import L from "leaflet";
import type { PlayerState } from "./types";
import { gameToLatLng } from "./map";
import { JOB_COLORS, TRAIL_OPTIONS } from "./config";

interface PlayerEntry {
  marker: L.CircleMarker;
  trail: L.Polyline;
}

const players = new Map<number, PlayerEntry>();
let playerGroup: L.LayerGroup;
let trailGroup: L.LayerGroup;

export function initPlayers(map: L.Map) {
  playerGroup = L.layerGroup().addTo(map);
  trailGroup = L.layerGroup().addTo(map);
}

function jobColor(jobGroup?: string): string {
  if (!jobGroup) return JOB_COLORS.default;
  return JOB_COLORS[jobGroup] ?? JOB_COLORS.default;
}

export function updatePlayers(data: PlayerState[]) {
  const seen = new Set<number>();

  for (const p of data) {
    seen.add(p.vrp_id);
    const pos = gameToLatLng(p.x, p.y);
    const color = jobColor(p.job_group);

    let entry = players.get(p.vrp_id);
    if (entry) {
      entry.marker.setLatLng(pos);
      entry.marker.setStyle({ color, fillColor: color });
      entry.marker.setTooltipContent(p.name || `#${p.vrp_id}`);
    } else {
      const marker = L.circleMarker(pos, {
        radius: 5,
        color,
        fillColor: color,
        fillOpacity: 0.8,
        weight: 1,
      });
      marker.bindTooltip(p.name || `#${p.vrp_id}`, { direction: "top", offset: [0, -6] });
      marker.bindPopup(""); // placeholder, updated below
      const trail = L.polyline([], {
        color,
        ...TRAIL_OPTIONS,
      });
      entry = { marker, trail };
      players.set(p.vrp_id, entry);
      playerGroup.addLayer(marker);
      trailGroup.addLayer(trail);
    }

    // Update popup content
    const parts = [`<b>${p.name || `#${p.vrp_id}`}</b>`, `ID: ${p.vrp_id}`];
    if (p.job_group) parts.push(`Job: ${p.job_group}${p.job_name ? ` (${p.job_name})` : ""}`);
    if (p.vehicle_name) parts.push(`Vehicle: ${p.vehicle_name}`);
    parts.push(`Pos: ${p.x.toFixed(1)}, ${p.y.toFixed(1)}, ${p.z.toFixed(1)}`);
    entry.marker.setPopupContent(parts.join("<br>"));

    // Update trail
    if (p.trail && p.trail.length > 0) {
      const trailLatLngs = p.trail.map((t) => gameToLatLng(t.x, t.y));
      trailLatLngs.push(pos); // current position at end
      entry.trail.setLatLngs(trailLatLngs);
      entry.trail.setStyle({ color });
    } else {
      entry.trail.setLatLngs([]);
    }
  }

  // Remove players no longer present
  for (const [id, entry] of players) {
    if (!seen.has(id)) {
      playerGroup.removeLayer(entry.marker);
      trailGroup.removeLayer(entry.trail);
      players.delete(id);
    }
  }
}

export function setPlayersVisible(visible: boolean) {
  playerGroup.eachLayer((l) => {
    (l as L.CircleMarker).setStyle({ opacity: visible ? 1 : 0, fillOpacity: visible ? 0.8 : 0 });
  });
}

export function setTrailsVisible(visible: boolean) {
  trailGroup.eachLayer((l) => {
    (l as L.Polyline).setStyle({ opacity: visible ? TRAIL_OPTIONS.opacity : 0 });
  });
}
