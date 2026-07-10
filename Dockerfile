# ---- Stage 1: build ----
FROM golang:1.25.12-alpine AS builder

WORKDIR /src

# Cache dependency layer
COPY go.mod ./
RUN go mod download

COPY . .

# ARG diisi dari CI (git sha), dipakai untuk versioning
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/go-k8s-deployment .

# ---- Stage 2: runtime ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=builder /out/go-k8s-deployment .

USER app
EXPOSE 8080

ENTRYPOINT ["./go-k8s-deployment"]
