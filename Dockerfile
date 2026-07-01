FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/freshtrack ./cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/freshtrack /app/freshtrack
COPY web /app/web
EXPOSE 8080
CMD ["/app/freshtrack"]
