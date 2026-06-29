# ---------------------- 第一阶段：编译专用临时镜像（builder，不会打进最终镜像） ----------------------
FROM lsiobase/ubuntu:noble AS builder
ARG TARGETARCH

# 一次性安装编译依赖并清理缓存
RUN apt-get update && apt-get install -y \
    libssl3 \
    libssl-dev \
    unzip \
    golang-go \
    && rm -rf /var/lib/apt/lists/*

# 复制go源码编译web程序
COPY ./web /tmp/web
RUN cd /tmp/web \
    && go mod init webui \
    && CGO_ENABLED=0 GOARCH=$TARGETARCH go build -o /tmp/webui \
    && chmod +x /tmp/webui

# ---------------------- 第二阶段：最终运行镜像（仅保留运行必需文件，无Go编译器） ----------------------
FROM lsiobase/ubuntu:noble

ARG TARGETARCH
ENV ARCH_VAR=$TARGETARCH
ENV S6_STAGE2_HOOK=/app/init.sh
ENV WEB_PORT=8087

# 仅保留运行时依赖，删除golang-go！同时清理apt缓存
RUN apt-get update && apt-get install -y \
    libssl3 \
    libssl-dev \
    unzip \
    && rm -rf /var/lib/apt/lists/*

# 复制系统服务、初始化脚本
COPY root/ /
COPY /src /app

# 从builder阶段复制编译完成的web二进制（唯一需要的Go产物）
COPY --from=builder /tmp/webui /app/web
RUN chmod +x /app/web

# 一次性下载AirConnect主程序，清理全部临时文件
RUN if [ "$ARCH_VAR" = "amd64" ]; then ARCH_VAR=linux-x86_64; elif [ "$ARCH_VAR" = "arm64" ]; then ARCH_VAR=linux-aarch64; fi \
    && curl -s https://api.github.com/repos/philippe44/AirConnect/releases/latest | grep browser_download_url | cut -d '"' -f 4 | xargs curl -L -o airconnect.zip \
    && unzip airconnect.zip -d /tmp/ \
    && mv /tmp/airupnp-$ARCH_VAR /usr/bin/airupnp-docker \
    && mv /tmp/aircast-$ARCH_VAR /usr/bin/aircast-docker \
    && chmod +x /usr/bin/airupnp-docker /usr/bin/aircast-docker \
    && rm -rf /tmp/* airconnect.zip

# 统一授权web服务全部配置文件
RUN chmod +x /etc/s6-overlay/s6-rc.d/web/run \
    && chmod 644 /etc/s6-overlay/s6-rc.d/web/type \
    && chmod 644 /etc/s6-overlay/s6-rc.d/web/dependencies

EXPOSE 8087
ENTRYPOINT ["/init"]
