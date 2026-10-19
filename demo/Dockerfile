FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY . .
RUN go build -o /falcon .

FROM alpine:3.20
COPY --from=build /falcon /usr/local/bin/falcon
COPY demo/config.toml /etc/falcon/config.toml
EXPOSE 8000 9900
ENTRYPOINT ["falcon", "-config=/etc/falcon/config.toml"]
