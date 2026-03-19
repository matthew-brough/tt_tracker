FROM golang:1.26-alpine AS build
WORKDIR /src
ENV GOWORK=off

# Copy module files first for caching
COPY shared/go.mod shared/go.sum ./shared/
COPY collector/go.mod collector/go.sum ./collector/

# Download deps
RUN cd collector && go mod download

# Copy source
COPY shared/ ./shared/
COPY collector/ ./collector/

# Build
RUN cd collector && CGO_ENABLED=0 go build -o /bin/collector ./cmd/collector

FROM gcr.io/distroless/static
COPY --from=build /bin/collector /collector
ENTRYPOINT ["/collector"]
