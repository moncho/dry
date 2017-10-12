FROM alpine:latest

LABEL VERSION v0.9-beta.1

RUN set -x && \
    apk update && \
    apk upgrade && \
    apk add curl file && \
    curl https://moncho.github.io/dry/dryup.sh | sh && \
    apk del curl file && \
    rm -rf /var/cache/apk/* && \
    chmod 755 /usr/local/bin/dry

CMD sleep 1;/usr/local/bin/dry
