FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 go build -o /out/boss-server ./cmd/server

FROM alpine:3.20
RUN adduser -D app
COPY --from=build /out/boss-server /usr/local/bin/boss-server
USER app
ENTRYPOINT ["boss-server"]
