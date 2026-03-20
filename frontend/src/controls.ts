import { setPlayersVisible, setTrailsVisible } from "./players";
import { setHeatmapVisible } from "./heatmap";
import { fetchFilterOptions } from "./api";
import type { FilterOption } from "./types";

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

function initSearchDropdown(
  id: string,
  type: "job" | "vehicle",
  onSelect: (value: string) => void
) {
  const wrapper = document.getElementById(id);
  if (!wrapper) return;

  const input = wrapper.querySelector("input") as HTMLInputElement;
  const list = wrapper.querySelector(".dd-list") as HTMLElement;
  let debounceTimer: ReturnType<typeof setTimeout>;
  let selected = "";

  async function loadOptions(search: string) {
    let options: FilterOption[];
    try {
      options = await fetchFilterOptions(state.server, type, search);
    } catch {
      options = [];
    }
    list.innerHTML = "";
    if (options.length === 0) {
      const empty = document.createElement("div");
      empty.className = "dd-empty";
      empty.textContent = search ? "no matches" : "type to search...";
      list.appendChild(empty);
      return;
    }
    for (const opt of options) {
      const item = document.createElement("div");
      item.className = "dd-item";
      item.innerHTML = `<span>${opt.value}</span><span class="dd-count">${opt.count.toLocaleString()}</span>`;
      item.addEventListener("mousedown", (e) => {
        e.preventDefault(); // prevent input blur
        selected = opt.value;
        input.value = opt.value;
        wrapper!.classList.remove("open");
        onSelect(opt.value);
      });
      list.appendChild(item);
    }
  }

  input.addEventListener("focus", () => {
    wrapper.classList.add("open");
    loadOptions(input.value.trim());
  });

  input.addEventListener("input", () => {
    selected = "";
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => {
      loadOptions(input.value.trim());
    }, 300);
  });

  input.addEventListener("blur", () => {
    // small delay to allow mousedown on items
    setTimeout(() => {
      wrapper.classList.remove("open");
      // if the user typed something that doesn't match a selection, clear it
      if (!selected && input.value.trim() !== "") {
        // keep what they typed as a freeform filter
        selected = input.value.trim();
        onSelect(selected);
      } else if (!selected && input.value.trim() === "") {
        onSelect("");
      }
    }, 150);
  });

  // allow clearing with escape
  input.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      input.value = "";
      selected = "";
      wrapper.classList.remove("open");
      onSelect("");
    }
  });
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

  // Searchable filter dropdowns
  initSearchDropdown("dd-job", "job", (value) => {
    state.job = value;
    callbacks.onHeatmapParamsChange();
  });

  initSearchDropdown("dd-vehicle", "vehicle", (value) => {
    state.vehicle = value;
    callbacks.onHeatmapParamsChange();
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
