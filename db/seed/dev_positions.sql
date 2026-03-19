-- Dev seed: realistic player movement across GTA map hotspots
-- Generates ~2M rows simulating 7 days of data at 2Hz polling

-- Players
INSERT INTO players (vrp_id, server, player_name)
SELECT i, 'main', 'Player_' || i FROM generate_series(1, 100) AS i
ON CONFLICT DO NOTHING;

INSERT INTO players (vrp_id, server, player_name)
SELECT i, 'beta', 'BetaTester_' || i FROM generate_series(1, 30) AS i
ON CONFLICT DO NOTHING;

-- GTA V map hotspots (game world coords)
CREATE TEMP TABLE hotspots (
    id INT, cx DOUBLE PRECISION, cy DOUBLE PRECISION, radius DOUBLE PRECISION
);
INSERT INTO hotspots VALUES
    (1,  -200,   -800,   400),   -- city center / legion square
    (2,  -1100,  -2900,  300),   -- airport
    (3,  1200,   -3100,  200),   -- docks
    (4,  -200,   6300,   250),   -- paleto bay
    (5,  1900,   3700,   300),   -- sandy shores
    (6,  300,    -100,   250),   -- vinewood / downtown
    (7,  800,    2000,   600),   -- highway north corridor
    (8,  -500,   -1800,  500),   -- highway south / del perro
    (9,  -1600,  3000,   200),   -- fort zancudo area
    (10, 2500,   5000,   400),   -- grapeseed / east highway
    (11, -800,   -200,   300),   -- west LS / morningwood
    (12, 400,    -2500,  200);   -- port / elysian island

-- ~4000 sessions, each 200-800 points (100-400 sec at 2Hz)
-- Hotspot picked by modulo so distribution is even
INSERT INTO player_positions (ts, server, vrp_id, x, y, z, job_group, job_name, vehicle_type, vehicle_name)
SELECT
    session_start + (s.n * INTERVAL '500 milliseconds'),
    server, vrp_id,
    cx + (random() - 0.5) * radius + sin(s.n::float / 30.0 + phase) * radius * 0.4,
    cy + (random() - 0.5) * radius + cos(s.n::float / 30.0 + phase) * radius * 0.4,
    CASE WHEN vtype = 'helicopter' THEN 50 + random() * 200 ELSE random() * 15 END,
    job_group, job_name, vtype, vname
FROM (
    SELECT
        gs.id AS session_id,
        NOW() - (random() * INTERVAL '7 days') AS session_start,
        CASE WHEN random() < 0.75 THEN 'main' ELSE 'beta' END AS server,
        (random() * 99 + 1)::INT AS vrp_id,
        h.cx, h.cy, h.radius,
        random() * 6.28 AS phase,
        (ARRAY['police','ambulance','mechanic','civilian','civilian','civilian','firefighter','taxi'])[
            (random() * 7 + 1)::INT] AS job_group,
        (ARRAY['Officer','Paramedic','Mechanic','Citizen','Citizen','Citizen','Firefighter','Driver'])[
            (random() * 7 + 1)::INT] AS job_name,
        (ARRAY['car','car','car','motorcycle','helicopter','boat','truck','foot','foot'])[
            (random() * 8 + 1)::INT] AS vtype,
        (ARRAY['Sultan','Elegy','Kuruma','Bati801','Maverick','Dinghy','Hauler','',''])[
            (random() * 8 + 1)::INT] AS vname
    FROM generate_series(1, 4000) AS gs(id)
    JOIN hotspots h ON h.id = ((gs.id - 1) % 12) + 1
) sessions,
LATERAL generate_series(0, 200 + (sessions.session_id % 600)) AS s(n);

DROP TABLE hotspots;
