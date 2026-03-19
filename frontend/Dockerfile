FROM node:24-alpine AS build
RUN npm install -g serve
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci
COPY tsconfig.json tsup.config.ts ./
COPY src/ ./src/
RUN npm run build && \
    cp node_modules/leaflet/dist/leaflet.css dist/ && \
    cp -r node_modules/leaflet/dist/images dist/

FROM node:24-alpine
RUN npm install -g serve
WORKDIR /app
COPY --from=build /app/dist/ ./dist/
COPY index.html ./
EXPOSE 3000
CMD ["serve", ".", "-l", "3000"]
