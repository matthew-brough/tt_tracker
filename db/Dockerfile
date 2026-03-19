FROM timescale/timescaledb:latest-pg17

RUN apk add --no-cache postgis \
    && cp /usr/lib/postgresql17/postgis*.so /usr/local/lib/postgresql/ \
    && cp /usr/share/postgresql17/extension/postgis* /usr/local/share/postgresql/extension/ \
    && cp /usr/share/postgresql17/extension/address_standardizer* /usr/local/share/postgresql/extension/ 2>/dev/null; true
