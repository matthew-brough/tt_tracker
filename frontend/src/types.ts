export interface Position {
  x: number;
  y: number;
  z: number;
}

export interface PlayerState {
  vrp_id: number;
  name: string;
  x: number;
  y: number;
  z: number;
  job_group?: string;
  job_name?: string;
  vehicle_type?: string;
  vehicle_name?: string;
  trail?: Position[];
}

export interface HexBin {
  x: number;
  y: number;
  count: number;
  edge: number;
}

export interface HeatmapParams {
  server: string;
  job?: string;
  vehicle?: string;
  range?: string;
  edge: number;
  minX: number;
  minY: number;
  maxX: number;
  maxY: number;
}
