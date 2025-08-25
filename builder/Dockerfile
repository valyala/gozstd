ARG builder_image
FROM $builder_image


ARG zig_version
ENV ZIG_VERSION=$zig_version

RUN apk add --no-cache \
        ca-certificates \
        wget \
        make \
        git \
        xz

RUN wget -O zig.tar.xz "https://ziglang.org/download/${ZIG_VERSION}/zig-x86_64-linux-${ZIG_VERSION}.tar.xz" && \
    tar -xf zig.tar.xz --strip-components=1 -C /usr/bin/ && \
    chmod +x /usr/bin/zig

ENTRYPOINT ["/bin/sh"]
