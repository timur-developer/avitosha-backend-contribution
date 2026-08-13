FROM golang:1.25.0-alpine3.22 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/auth-service ./cmd/auth \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/game-service ./cmd/game

FROM alpine:3.22.0

WORKDIR /app

COPY --from=build /out/api /usr/local/bin/api
COPY --from=build /out/auth-service /usr/local/bin/auth-service
COPY --from=build /out/game-service /usr/local/bin/game-service

EXPOSE 8080 9091 9092

CMD ["/usr/local/bin/api"]
