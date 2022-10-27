FROM golang:alpine3.15 AS build

WORKDIR /usr/datahop
COPY go.mod go.sum /usr/datahop/
RUN go mod download
COPY . /usr/datahop/

RUN apk add --update --no-cache make  gcc git musl-dev libc-dev linux-headers bash \
        && make binary

FROM alpine:3.15

RUN addgroup -g 10000 datahop \
    && adduser -u 10000 -G datahop -h /home/datahop -D datahop
USER datahop
EXPOSE 4501

COPY --from=build /usr/datahop/dist/datahop /usr/local/bin/datahop

ENTRYPOINT ["datahop"]