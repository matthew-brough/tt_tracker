import { initMap, getGameBounds } from "./map";
import { initPlayers, updatePlayers } from "./players";
import { initHeatmap, updateHeatmap, isHeatmapVisible } from "./heatmap";
import { initControls, getControlState } from "./controls";
import { fetchPlayers, fetchHeatmap } from "./api";
import { PLAYER_POLL_MS, DEFAULT_SERVER, TARGET_HEX_SPAN, MIN_EDGE, MAX_EDGE } from "./config";

const map = initMap("map");
initPlayers(map);
initHeatmap(map);

const countEl = document.querySelector("#player-count span");

async function pollPlayers() {
  try {
    const state = getControlState();
    const players = await fetchPlayers(state.server);
    updatePlayers(players);
    if (countEl) countEl.textContent = String(players.length);
  } catch (e) {
    console.error("player poll failed:", e);
  }
}

async function pollHeatmap() {
  try {
    if (!isHeatmapVisible()) return;
    const state = getControlState();
    const bounds = getGameBounds(map);
    const span = Math.max(bounds.maxX - bounds.minX, bounds.maxY - bounds.minY);
    const edge = Math.min(MAX_EDGE, Math.max(MIN_EDGE, Math.round(span / TARGET_HEX_SPAN)));
    const bins = await fetchHeatmap({
      server: state.server,
      job: state.job || undefined,
      vehicle: state.vehicle || undefined,
      range: state.range || undefined,
      edge,
      ...bounds,
    });
    updateHeatmap(bins);
  } catch (e) {
    console.error("heatmap poll failed:", e);
  }
}

function onServerChange() {
  updatePlayers([]);
  pollPlayers();
  if (isHeatmapVisible()) pollHeatmap();
}

initControls(
  {
    server: DEFAULT_SERVER,
    heatmapEnabled: false,
    job: "",
    vehicle: "",
    range: "1h",
  },
  {
    onServerChange,
    onHeatmapParamsChange: pollHeatmap,
    map,
  }
);

// Initial fetch + polling loop (heatmap only fetches on toggle/filter change)
pollPlayers();
setInterval(pollPlayers, PLAYER_POLL_MS);
