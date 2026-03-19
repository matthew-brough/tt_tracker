FROM golang:1.26-alpine AS build
WORKDIR /src
ENV GOWORK=off

# Copy module files first for caching
COPY shared/go.mod shared/go.sum ./shared/
COPY api/go.mod api/go.sum ./api/

# Download deps
RUN cd api && go mod download

# Copy source
COPY shared/ ./shared/
COPY api/ ./api/

# Build
RUN cd api && CGO_ENABLED=0 go build -o /bin/api ./cmd/api

FROM gcr.io/distroless/static
COPY --from=build /bin/api /api
ENTRYPOINT ["/api"]
