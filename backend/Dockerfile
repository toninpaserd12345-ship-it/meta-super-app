FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /api ./cmd/api

FROM gcr.io/distroless/static-debian12:debug-nonroot AS debug

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /api /api
COPY --from=debug --chown=nonroot:nonroot /bin/sh /bin/sh
WORKDIR /data
EXPOSE 8080
ENTRYPOINT ["/api"]
