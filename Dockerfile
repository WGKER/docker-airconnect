FROM lsiobase/ubuntu:noble

# Pulling TARGET_ARCH from build arguments and setting ENV variable
ARG TARGETARCH
ENV ARCH_VAR=$TARGETARCH
ENV S6_STAGE2_HOOK=/app/init.sh
# 声明web端口
ENV WEB_PORT=8087

# Add Supervisor
RUN apt-get update && apt-get install -y \
    libssl3 \
    libssl-dev \
    unzip \
    # 安装依赖：运行依赖 + golang编译环境
    golang-go
    
COPY root/ /
COPY /src /app

# ========== 编译Go Web管理程序 ==========
# 复制Go源码到临时目录编译
COPY ./web /tmp/web
RUN echo "=== 查看/tmp/web文件 ===" \
    && ls -l /tmp/web \
    && cd /tmp/web \
    && CGO_ENABLED=0 GOARCH=$ARCH_VAR go build -v -o /app/web \
    && chmod +x /app/web \
    && rm -rf /tmp/web

# Grab latest version of the app, extract binaries, cleanup tmp dir
RUN if [ "$ARCH_VAR" = "amd64" ]; then ARCH_VAR=linux-x86_64; elif [ "$ARCH_VAR" = "arm64" ]; then ARCH_VAR=linux-aarch64; fi \
    && curl -s https://api.github.com/repos/philippe44/AirConnect/releases/latest | grep browser_download_url | cut -d '"' -f 4 | xargs curl -L -o airconnect.zip \
    && unzip airconnect.zip -d /tmp/ \
    && mv /tmp/airupnp-$ARCH_VAR /usr/bin/airupnp-docker \
    && mv /tmp/aircast-$ARCH_VAR /usr/bin/aircast-docker \
    && chmod +x /usr/bin/airupnp-docker \
    && chmod +x /usr/bin/aircast-docker \
    && rm -r /tmp/* \
    && rm airconnect.zip

# 赋予web服务脚本执行权限
RUN chmod +x etc/s6-overlay/s6-rc.d/web/run

# 对外声明8087端口（docker run -p 8087:8087 映射）
EXPOSE 8087

ENTRYPOINT ["/init"]
