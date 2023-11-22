FROM docker.io/golang:1.21.4-alpine3.18

RUN apk update && apk --no-cache add make gcc musl-dev git bash
WORKDIR /opt
COPY . .
RUN make build

CMD ["./build/demo"]