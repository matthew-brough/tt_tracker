export const PLAYER_POLL_MS = 10000;
export const DEFAULT_SERVER = "main";
export const TARGET_HEX_SPAN = 20;
export const MIN_EDGE = 15;
export const MAX_EDGE = 500;

export const TRAIL_OPTIONS = {
  weight: 2,
  opacity: 0.5,
} as const;

export const JOB_COLORS: Record<string, string> = {
  police: "#3b82f6",
  fire: "#ef4444",
  ems: "#f59e0b",
  mechanic: "#10b981",
  taxi: "#eab308",
  default: "#6b7280",
};

export const HEATMAP_COLOR_STOPS = [
  [0.0, "#3b82f6"],
  [0.25, "#06b6d4"],
  [0.5, "#22c55e"],
  [0.75, "#f59e0b"],
  [1.0, "#ef4444"],
] as const;

export const TIME_RANGES: Record<string, number> = {
  "1h": 3600,
  "6h": 21600,
  "24h": 86400,
  "7d": 604800,
};
