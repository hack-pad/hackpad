FROM node:14 as builder

RUN apt-get update && \
        apt-get install -y golang

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
