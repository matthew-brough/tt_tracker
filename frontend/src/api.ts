import type { PlayerState, HexBin, HeatmapParams, FilterOption } from "./types";

export async function fetchPlayers(server: string): Promise<PlayerState[]> {
  const res = await fetch(`/api/players?server=${encodeURIComponent(server)}`);
  if (!res.ok) throw new Error(`players: ${res.status}`);
  return (await res.json()) ?? [];
}

export async function fetchHeatmap(params: HeatmapParams): Promise<HexBin[]> {
  const url = new URL("/api/heatmap", location.origin);
  url.searchParams.set("server", params.server);
  if (params.job) url.searchParams.set("job", params.job);
  if (params.vehicle) url.searchParams.set("vehicle", params.vehicle);
  url.searchParams.set("edge", String(params.edge));
  url.searchParams.set("minx", String(params.minX));
  url.searchParams.set("miny", String(params.minY));
  url.searchParams.set("maxx", String(params.maxX));
  url.searchParams.set("maxy", String(params.maxY));
  if (params.range) {
    const seconds =
      { "1h": 3600, "6h": 21600, "24h": 86400, "7d": 604800 }[params.range] ??
      3600;
    const to = new Date();
    const from = new Date(to.getTime() - seconds * 1000);
    url.searchParams.set("from", from.toISOString());
    url.searchParams.set("to", to.toISOString());
  }
  const res = await fetch(url.toString());
  if (!res.ok) throw new Error(`heatmap: ${res.status}`);
  return (await res.json()) ?? [];
}

export async function fetchFilterOptions(
  server: string,
  type: "job" | "vehicle",
  search: string
): Promise<FilterOption[]> {
  const url = new URL("/api/filter-options", location.origin);
  url.searchParams.set("server", server);
  url.searchParams.set("type", type);
  if (search) url.searchParams.set("search", search);
  url.searchParams.set("limit", "20");
  const res = await fetch(url.toString());
  if (!res.ok) throw new Error(`filter-options: ${res.status}`);
  return (await res.json()) ?? [];
}
