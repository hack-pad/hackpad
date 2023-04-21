FROM golang:1.20 as go-builder
WORKDIR /src
# Cache go installation first
COPY Makefile /src
RUN make go
# Build command binaries and static assets
COPY . /src
RUN make go-static

FROM node:14 as node-builder
WORKDIR /src
COPY Makefile /src
COPY ./server /src/server
COPY --from=go-builder /src/server/public/wasm /src/server/public/wasm
RUN make node-static

FROM nginx:1

RUN sed -i 's@}@application/wasm wasm;}@' /etc/nginx/mime.types

COPY --from=node-builder /src/server/build/ /usr/share/nginx/html
RUN test -f /usr/share/nginx/html/index.html
