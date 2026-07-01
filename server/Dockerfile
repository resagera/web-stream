FROM golang:1.25.7-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -o /out/home-stream-server ./cmd/server

FROM alpine:3.22

WORKDIR /app
COPY --from=build /out/home-stream-server /app/home-stream-server

ENV ADDR=:8080
ENV DATA_DIR=/app/data
EXPOSE 8080

CMD ["/app/home-stream-server"]
