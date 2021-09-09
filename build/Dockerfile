FROM alpine:3.13.6

ENV TZ=Asia/Shanghai

RUN apk -U add --no-cache ca-certificates tzdata \
    && ln -snf /usr/share/zoneinfo/$TZ /etc/localtime \
    && echo $TZ > /etc/timezone

COPY takeover /app

ENTRYPOINT ["/app/takeover"]
