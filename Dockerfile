FROM node:14-alpine as builder

RUN apk add --no-cache \
        bash \
        git \
        go \
        make

WORKDIR /src
# Cache go installation first
COPY Makefile /src
RUN make go
# Build command binaries and static assets
COPY . /src
RUN make build

FROM nginx:1

RUN sed -i 's@}@application/wasm wasm;}@' /etc/nginx/mime.types

COPY --from=builder /src/out /usr/share/nginx/html
