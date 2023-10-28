FROM golang:latest AS build

WORKDIR /build
COPY . .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -a -o main .

FROM alpine:latest

RUN apk update && \
    apk upgrade && \
    apk add ca-certificates && \
    apk add tzdata

WORKDIR /app

COPY --from=build /build/main ./

RUN pwd && find .

CMD ["./main"]
