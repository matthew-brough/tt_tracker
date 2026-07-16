FROM timescale/timescaledb:2.28.3-pg18

RUN apk add --no-cache postgis=3.6.1-r0 \
    && cp /usr/lib/postgresql18/postgis*.so /usr/local/lib/postgresql/ \
    && cp /usr/share/postgresql18/extension/postgis* /usr/local/share/postgresql/extension/ \
    && cp /usr/share/postgresql18/extension/address_standardizer* /usr/local/share/postgresql/extension/
