# Builder container
FROM golang AS builder

WORKDIR /go/src/mesh-agent
COPY . .

RUN go build -o /root/mesh/agent

# Runner container
FROM registry.cn-hangzhou.aliyuncs.com/aliware2018/services AS service
FROM registry.cn-hangzhou.aliyuncs.com/aliware2018/debian-jdk8

COPY --from=service /root/workspace/services/mesh-provider/target/mesh-provider-1.0-SNAPSHOT.jar /root/dists/mesh-provider.jar
COPY --from=service /root/workspace/services/mesh-consumer/target/mesh-consumer-1.0-SNAPSHOT.jar /root/dists/mesh-consumer.jar
COPY --from=service /usr/local/bin/docker-entrypoint.sh /usr/local/bin

COPY start-agent.sh /usr/local/bin

COPY --from=builder /root/mesh/agent /root/dists/consumer
COPY --from=builder /root/mesh/agent /root/dists/provider
COPY plugin_logger.xml /root/dists

RUN set -ex && chmod a+x /usr/local/bin/start-agent.sh  && mkdir -p /root/logs

EXPOSE 8087

ENTRYPOINT ["docker-entrypoint.sh"]