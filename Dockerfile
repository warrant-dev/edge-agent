FROM alpine:3.10.3 as compressor

WORKDIR ./
RUN apk add upx

COPY ./bin/edge-agent ./
RUN upx --best --lzma ./edge-agent

FROM alpine:3.10.3

RUN addgroup -S warrant-edge && adduser -S warrant-edge -G warrant-edge
USER warrant-edge

WORKDIR ./
COPY --from=compressor ./edge-agent ./

ENTRYPOINT ["./edge-agent"]

EXPOSE 3000
