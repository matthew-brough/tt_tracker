import L from "leaflet";

const CX = 123.58,
  CY = 150;
const SX = 0.014228,
  SY = 0.014238;

export const CUSTOM_CRS: L.CRS = L.extend({}, L.CRS.Simple, {
  projection: L.Projection.LonLat,
  scale: (zoom: number) => Math.pow(2, zoom),
  zoom: (sc: number) => Math.log(sc) / 0.6931471805599453,
  distance: (a: L.LatLng, b: L.LatLng) => {
    const dx = b.lng - a.lng,
      dy = b.lat - a.lat;
    return Math.sqrt(dx * dx + dy * dy);
  },
  transformation: new L.Transformation(SX, CX, -SY, CY),
  infinite: true,
});

export function gameToLatLng(x: number, y: number): L.LatLng {
  return L.latLng(y, x);
}

export function getGameBounds(map: L.Map): { minX: number; minY: number; maxX: number; maxY: number } {
  const b = map.getBounds();
  return { minX: b.getWest(), minY: b.getSouth(), maxX: b.getEast(), maxY: b.getNorth() };
}

export function initMap(elementId: string): L.Map {
  const map = L.map(elementId, {
    crs: CUSTOM_CRS,
    zoom: 3,
    center: L.latLng(1500, 400),
    zoomControl: true,
    attributionControl: false,
    maxZoom: 8,
    minZoom: 0,
  });

  L.tileLayer("/tiles/colour/{z}/{x}/{y}.webp", {
    maxZoom: 8,
    minZoom: 0,
    noWrap: true,
  }).addTo(map);

  // Constrain panning to map area (matches source site bounds)
  const maxBounds = L.latLngBounds(
    map.unproject(L.point(-10000, 75000), 8),
    map.unproject(L.point(75000, -20000), 8),
  );
  map.setMaxBounds(maxBounds);

  return map;
}
