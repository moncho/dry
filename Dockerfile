# build stage
FROM alpine:latest AS build-phase

LABEL VERSION v0.9-beta.6

RUN set -x && \
    apk update && \
    apk upgrade && \
    apk add curl file && \
    curl https://moncho.github.io/dry/dryup.sh | sh && \
    apk del curl file && \
    rm -rf /var/cache/apk/* && \
    chmod 755 /usr/local/bin/dry

# final stage
FROM alpine
WORKDIR /app
COPY --from=build-phase /usr/local/bin/dry /app

CMD sleep 1;/app/dry
