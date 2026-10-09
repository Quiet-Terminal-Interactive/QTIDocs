FROM golang:1.27-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server && \
    go build -trimpath -ldflags="-s -w" -o /out/platform ./cmd/platform && \
    go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker && \
    mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot AS runtime
WORKDIR /app
COPY --from=build /out/server /out/platform /out/worker ./
COPY --from=build --chown=65532:65532 /out/data /data

ENTRYPOINT ["/app/server"]
