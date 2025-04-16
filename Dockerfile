FROM alpine

ARG APP_VERSION
ARG TARGETARCH
ARG APP_FAMILY
ARG PROXY="proxy=0"

ENV APP_NAME=dpanel
ENV APP_ENV=production
ENV APP_FAMILY=$APP_FAMILY
ENV APP_VERSION=$APP_VERSION
ENV APP_SERVER_PORT=8080

ENV DOCKER_HOST=unix:///var/run/docker.sock
ENV STORAGE_LOCAL_PATH=/dpanel
ENV DB_DATABASE=${STORAGE_LOCAL_PATH}/dpanel.db
ENV TZ=Asia/Shanghai
ENV ACME_OVERRIDE_CONFIG_HOME=/dpanel/acme

# ====== 钉钉告警配置 ======
# 是否启用钉钉告警通知，设置为true启用，false禁用
ENV DINGTALK_ENABLED=false
# 钉钉机器人webhook地址，格式为：https://oapi.dingtalk.com/robot/send?access_token=xxxx
ENV DINGTALK_WEBHOOK_URL=""
# 钉钉机器人安全设置的签名密钥（机器人安全设置选择"加签"后的SEC密钥）
ENV DINGTALK_SECRET=""

# ====== 容器告警配置 ======
# 是否启用容器状态告警，设置为true启用，false禁用
ENV CONTAINER_ALERT_ENABLED=false
# 需要监控的容器名称，多个用逗号分隔，留空则监控所有容器
# 例如：ENV CONTAINER_ALERT_MONITOR="mysql,nginx,redis"
ENV CONTAINER_ALERT_MONITOR=""
# 排除监控的容器名称，多个用逗号分隔，即使开启监控所有容器，这些容器也不会被监控
# 例如：ENV CONTAINER_ALERT_EXCLUDE="test,temp,dev-nginx"
ENV CONTAINER_ALERT_EXCLUDE=""
# 容器停止时是否告警，默认为false（不告警）
ENV CONTAINER_ALERT_ON_STOP=false
# 容器异常退出时是否告警，默认为true（告警）
ENV CONTAINER_ALERT_ON_DIE=true
# 容器内存溢出(OOM)时是否告警，默认为true（告警）
ENV CONTAINER_ALERT_ON_OOM=true
# 容器健康检查失败时是否告警，默认为true（告警）
ENV CONTAINER_ALERT_ON_HEALTH=true
# 健康状态检查间隔时间（秒）
ENV CONTAINER_ALERT_INTERVAL=60

COPY ./docker/nginx/nginx.conf /etc/nginx/nginx.conf
COPY ./docker/nginx/include /etc/nginx/conf.d/include
COPY ./docker/script /app/script

COPY ./runtime/dpanel${APP_FAMILY:+"-${APP_FAMILY}"}-musl-${TARGETARCH} /app/server/dpanel
COPY ./runtime/config.yaml /app/server/config.yaml

COPY ./docker/entrypoint.sh /docker/entrypoint.sh

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories && \
  apk add --no-cache --update nginx musl docker-compose curl openssl tzdata git sqlite sqlite-dev && \
  mkdir -p /tmp/nginx/body /var/lib/nginx/cache/public /var/lib/nginx/cache/private && \
  export ${PROXY} && curl https://raw.githubusercontent.com/acmesh-official/acme.sh/master/acme.sh | sh -s -- --install-online --config-home /dpanel/acme && \
  chmod 755 /docker/entrypoint.sh

WORKDIR /app/server
VOLUME [ "/dpanel" ]

EXPOSE 443
EXPOSE 80
EXPOSE 8080

ENTRYPOINT [ "/docker/entrypoint.sh" ]