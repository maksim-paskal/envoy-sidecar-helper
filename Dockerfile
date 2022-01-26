FROM alpine:latest

COPY ./envoy-sidecar-helper /app/envoy-sidecar-helper

RUN addgroup -g 101 -S app \
&& adduser -u 101 -D -S -G app app

USER 101

ENTRYPOINT [ "/app/envoy-sidecar-helper" ]