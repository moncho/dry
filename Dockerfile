FROM alpine:latest

# get dependancies
RUN apk update && apk add curl file

# install dry
RUN curl -sSf https://moncho.github.io/dry/dryup.sh | sh
RUN chmod 755 /usr/local/bin/dry
CMD sleep 1;/usr/local/bin/dry
