FROM alpine:latest

LABEL maintainer "Moncho"

VERSION 0.8-beta.4

RUN set -x && \
    apk update && \
    apk upgrade && \
    apk add curl file && \
    curl https://moncho.github.io/dry/dryup.sh | sh && \
    apk del curl file && \
    rm -rf /var/cache/apk/* && \
    chmod 755 /usr/local/bin/dry

CMD sleep 1;/usr/local/bin/dry
