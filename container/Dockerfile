FROM alpine:latest
RUN apk update && apk add bash iproute2 && rm -fr /var/cache/apk
ADD . /
ENTRYPOINT ["/routestrator.sh"]