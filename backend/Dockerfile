FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /api /api
WORKDIR /data
ENV META_WHATSAPP_BUSINESS_ACCOUNT_IDS=1352953933654328
EXPOSE 8080
ENTRYPOINT ["/api"]
