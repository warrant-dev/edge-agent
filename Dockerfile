FROM alpine:3.10.3

RUN addgroup -S warrant-edge && adduser -S warrant-edge -G warrant-edge
USER warrant-edge

WORKDIR ./
COPY ./bin/edge-agent ./

ENTRYPOINT ["./edge-agent"]

EXPOSE 3000
