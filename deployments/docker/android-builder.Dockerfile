# Android 构建镜像:102 无法直连 Docker Hub(daocloud 镜像源白名单不含 cimg/android),
# 但 dl.google.com / services.gradle.org / maven central 可达,故自建:
# daocloud 白名单基座 eclipse-temurin:17-jdk + Google 官方源 cmdline-tools。
# SDK 平台包不预装(体积大),首跑经 gradle 自动下载到挂载卷 boss-android-sdk 复用。
# 构建:docker build -t 192.168.0.102:5000/boss/android-builder:1 -f deployments/docker/android-builder.Dockerfile deployments/docker
# (102 宿主执行;镜像随后 push 到私有 registry,workflow 直接引用)
FROM docker.m.daocloud.io/library/eclipse-temurin:17-jdk-jammy

ENV ANDROID_HOME=/opt/android-sdk
ENV PATH=${ANDROID_HOME}/cmdline-tools/latest/bin:${ANDROID_HOME}/platform-tools:${PATH}

RUN apt-get update \
    && apt-get install -y --no-install-recommends wget unzip \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p ${ANDROID_HOME}/cmdline-tools \
    && wget -q https://dl.google.com/android/repository/commandlinetools-linux-11076708_latest.zip -O /tmp/clt.zip \
    && unzip -q /tmp/clt.zip -d ${ANDROID_HOME}/cmdline-tools \
    && rm /tmp/clt.zip \
    && mv ${ANDROID_HOME}/cmdline-tools/cmdline-tools ${ANDROID_HOME}/cmdline-tools/latest \
    && yes | sdkmanager --licenses > /dev/null \
    && sdkmanager --install "platform-tools" > /dev/null \
    && apt-get purge -y wget unzip && apt-get autoremove -y
