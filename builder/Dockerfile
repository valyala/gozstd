ARG builder_image
FROM $builder_image
RUN apk add wget gcc musl-dev make git --no-cache && \
    mkdir /opt/cross-builder && \
    wget https://musl.cc/aarch64-linux-musl-cross.tgz -O /opt/cross-builder/aarch64-musl.tgz && \
    cd /opt/cross-builder && \
    tar zxf aarch64-musl.tgz -C ./  && \
    rm /opt/cross-builder/aarch64-musl.tgz

