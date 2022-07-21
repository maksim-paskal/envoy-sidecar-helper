FROM alpine:latest

COPY ./envoy-sidecar-helper /app/envoy-sidecar-helper

RUN addgroup -g 30001 -S app \
&& adduser -u 30001 -D -S -G app app

USER 30001

ENTRYPOINT [ "/app/envoy-sidecar-helper" ]