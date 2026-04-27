FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /crush-a2a ./cmd/crush-a2a

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /crush-a2a /usr/local/bin/crush-a2a
ENTRYPOINT ["crush-a2a"]
