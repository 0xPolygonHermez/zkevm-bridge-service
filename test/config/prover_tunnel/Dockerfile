FROM alpine:latest

RUN apk update && \
    apk add socat

EXPOSE 50051 50052 50061 50071

RUN echo "sleep infinity" > /usr/bin/zkProver
RUN chmod +x /usr/bin/zkProver

COPY docker-entrypoint.sh /
RUN chmod +x /docker-entrypoint.sh
ENTRYPOINT ["/docker-entrypoint.sh"]
