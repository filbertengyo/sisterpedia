# syntax=docker/dockerfile:1
# Build context is the repository root (the WASM stage needs go.mod, pkg/, cmd/wasm/).
ARG GO_VERSION=1.27

# 1) Lattigo CKKS -> WebAssembly (browser crypto bridge)
FROM golang:${GO_VERSION}-alpine AS wasm
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY config ./config
COPY pkg ./pkg
COPY cmd/wasm ./cmd/wasm
RUN GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o /out/crypto.wasm ./cmd/wasm \
 && cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" /out/wasm_exec.js

# 2) JS dependencies
FROM node:24-alpine AS deps
WORKDIR /app
COPY web/package.json web/package-lock.json ./
RUN npm ci

# 3) Next.js build (wasm assets placed in public/)
FROM node:24-alpine AS build
WORKDIR /app
ARG NEXT_PUBLIC_SERVER_API=http://localhost:8000
ARG NEXT_PUBLIC_MOCK=0
ENV NEXT_PUBLIC_SERVER_API=$NEXT_PUBLIC_SERVER_API \
    NEXT_PUBLIC_MOCK=$NEXT_PUBLIC_MOCK \
    NEXT_TELEMETRY_DISABLED=1
COPY --from=deps /app/node_modules ./node_modules
COPY web/ .
COPY --from=wasm /out/crypto.wasm /out/wasm_exec.js ./public/
RUN npm run build

# 4) Runtime
FROM node:24-alpine
WORKDIR /app
ENV NODE_ENV=production PORT=3000 HOSTNAME=0.0.0.0 NEXT_TELEMETRY_DISABLED=1
RUN adduser -D -u 10001 app
COPY --from=build --chown=app:app /app/.next/standalone ./
COPY --from=build --chown=app:app /app/.next/static ./.next/static
COPY --from=build --chown=app:app /app/public ./public
USER app
EXPOSE 3000
CMD ["node", "server.js"]
