import { setPlayersVisible, setTrailsVisible } from "./players";
import { setHeatmapVisible } from "./heatmap";

export interface ControlCallbacks {
  onServerChange: (server: string) => void;
  onHeatmapParamsChange: () => void;
  map: L.Map;
}

export interface ControlState {
  server: string;
  heatmapEnabled: boolean;
  job: string;
  vehicle: string;
  range: string;
}

let state: ControlState;
let callbacks: ControlCallbacks;

export function getControlState(): ControlState {
  return state;
}

export function initControls(
  initialState: ControlState,
  cbs: ControlCallbacks
) {
  state = initialState;
  callbacks = cbs;

  // Player visibility
  const playersCheck = document.getElementById("toggle-players") as HTMLInputElement | null;
  playersCheck?.addEventListener("change", () => {
    setPlayersVisible(playersCheck.checked);
  });

  // Trail visibility
  const trailsCheck = document.getElementById("toggle-trails") as HTMLInputElement | null;
  trailsCheck?.addEventListener("change", () => {
    setTrailsVisible(trailsCheck.checked);
  });

  // Heatmap visibility
  const heatmapCheck = document.getElementById("toggle-heatmap") as HTMLInputElement | null;
  heatmapCheck?.addEventListener("change", () => {
    state.heatmapEnabled = heatmapCheck.checked;
    setHeatmapVisible(heatmapCheck.checked);
    if (heatmapCheck.checked) callbacks.onHeatmapParamsChange();
  });

  // Server select
  const serverSelect = document.getElementById("server-select") as HTMLSelectElement | null;
  if (serverSelect) {
    serverSelect.value = state.server;
    serverSelect.addEventListener("change", () => {
      state.server = serverSelect.value;
      callbacks.onServerChange(state.server);
    });
  }

  // Heatmap filters with debounce
  let debounceTimer: ReturnType<typeof setTimeout>;
  const debounce = (fn: () => void) => {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(fn, 500);
  };

  const jobInput = document.getElementById("filter-job") as HTMLInputElement | null;
  jobInput?.addEventListener("input", () => {
    debounce(() => {
      state.job = jobInput.value.trim();
      callbacks.onHeatmapParamsChange();
    });
  });

  const vehicleInput = document.getElementById("filter-vehicle") as HTMLInputElement | null;
  vehicleInput?.addEventListener("input", () => {
    debounce(() => {
      state.vehicle = vehicleInput.value.trim();
      callbacks.onHeatmapParamsChange();
    });
  });

  const rangeSelect = document.getElementById("filter-range") as HTMLSelectElement | null;
  rangeSelect?.addEventListener("change", () => {
    state.range = rangeSelect.value;
    callbacks.onHeatmapParamsChange();
  });

  // Re-fetch heatmap on viewport change (pan/zoom)
  let moveDebounce: ReturnType<typeof setTimeout>;
  callbacks.map.on("moveend", () => {
    if (!state.heatmapEnabled) return;
    clearTimeout(moveDebounce);
    moveDebounce = setTimeout(() => callbacks.onHeatmapParamsChange(), 300);
  });
}
