ARG builder_image
FROM $builder_image
RUN apk --update --no-cache add ca-certificates
RUN apk --no-cache add wget gcc musl-dev make git
RUN mkdir -p /opt/cross-builder && \
    wget --no-check-certificate https://musl.cc/aarch64-linux-musl-cross.tgz -O /opt/cross-builder/aarch64-musl.tgz && \
    cd /opt/cross-builder && \
    tar zxf aarch64-musl.tgz -C ./  && \
    rm /opt/cross-builder/aarch64-musl.tgz
