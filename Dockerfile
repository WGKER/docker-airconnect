# ---------------------- 第一阶段：编译专用临时镜像（builder，不会打进最终镜像） ----------------------
FROM lsiobase/ubuntu:noble AS builder
# 接收流水线传入的版本号
ARG VERSION
ARG TARGETARCH

# 一次性安装编译依赖并清理缓存
RUN apt-get update && apt-get install -y \
    libssl3 \
    libssl-dev \
    unzip \
    golang-go \
    jq \
    && rm -rf /var/lib/apt/lists/*

# 复制go源码编译web程序
COPY ./web /tmp/web
RUN cd /tmp/web \
    && go mod init webui \
    && CGO_ENABLED=0 GOARCH=$TARGETARCH go build -o /tmp/webui \
    && chmod +x /tmp/webui

# ---------------------- 第二阶段：最终运行镜像（仅保留运行必需文件，无Go编译器） ----------------------
FROM lsiobase/ubuntu:noble

# 接收流水线构建参数
ARG VERSION
ARG TARGETARCH

# 关键：注入环境变量，Go程序读取 APP_VERSION 展示页面版本
ENV APP_VERSION=${VERSION}
ENV ARCH_VAR=$TARGETARCH
ENV S6_STAGE2_HOOK=/app/init.sh
ENV WEB_PORT=8087

# 仅保留运行时依赖，删除golang-go！同时清理apt缓存
RUN apt-get update && apt-get install -y \
    libssl3 \
    libssl-dev \
    unzip \
    jq \
    curl \
    && rm -rf /var/lib/apt/lists/*

# 复制系统服务、初始化脚本
COPY root/ /
COPY /src /app

# 从builder阶段复制编译完成的web二进制（唯一需要的Go产物）
COPY --from=builder /tmp/webui /app/web
RUN chmod +x /app/web

# 【优化】稳定拉取对应版本AirConnect，不再用grep切割，使用jq精准解析
RUN if [ "$ARCH_VAR" = "amd64" ]; then ARCH_VAR=linux-x86_64; elif [ "$ARCH_VAR" = "arm64" ]; then ARCH_VAR=linux-aarch64; fi \
    # 如果流水线传入VERSION，就下载指定版本；否则自动拉最新版
    && if [ -n "$VERSION" ] && [ "$VERSION" != "null" ]; then \
        echo "使用流水线指定版本 $VERSION 下载AirConnect"; \
        curl -s -H "Accept: application/vnd.github.v3+json" https://api.github.com/repos/philippe44/AirConnect/releases/tags/$VERSION > release.json; \
       else \
        echo "未指定版本，自动拉取最新Release"; \
        curl -s -H "Accept: application/vnd.github.v3+json" https://api.github.com/repos/philippe44/AirConnect/releases/latest > release.json; \
       fi \
    && DOWNLOAD_URL=$(jq -r '.assets[] | select(.name | contains("linux")) | .browser_download_url' release.json | head -n1) \
    && curl -L "$DOWNLOAD_URL" -o airconnect.zip \
    && unzip airconnect.zip -d /tmp/ \
    && mv /tmp/airupnp-$ARCH_VAR /usr/bin/airupnp-docker \
    && mv /tmp/aircast-$ARCH_VAR /usr/bin/aircast-docker \
    && chmod +x /usr/bin/airupnp-docker /usr/bin/aircast-docker \
    && rm -rf /tmp/* airconnect.zip release.json

# 统一授权web服务全部配置文件
RUN chmod -R +x /etc/services.d/ \
    && chmod 755 /app/web

EXPOSE 8087
ENTRYPOINT ["/init"]
