FROM golang:1.25-alpine AS builder

RUN apk add --no-cache \
    bash \
    build-base \
    ca-certificates \
    git \
    linux-headers \
    libmnl-dev \
    pkgconf

WORKDIR /src

ARG AMNEZIAWG_GO_REF=master
ARG AMNEZIAWG_TOOLS_REF=master

RUN git clone --depth 1 --branch "${AMNEZIAWG_GO_REF}" https://github.com/amnezia-vpn/amneziawg-go.git \
    && git clone --depth 1 --branch "${AMNEZIAWG_TOOLS_REF}" https://github.com/amnezia-vpn/amneziawg-tools.git

RUN cd /src/amneziawg-go \
    && make -j"$(nproc)" \
    && install -Dm755 amneziawg-go /out/usr/local/bin/amneziawg-go

RUN cd /src/amneziawg-tools/src \
    && make -j"$(nproc)" \
    && install -Dm755 wg /out/usr/bin/awg \
    && install -Dm755 wg-quick/linux.bash /out/usr/bin/awg-quick

FROM alpine:3.22

RUN apk add --no-cache \
    bash \
    ca-certificates \
    iproute2 \
    iptables \
    libmnl \
    openresolv

COPY --from=builder /out/ /
COPY docker/telegram-vpn/entrypoint.sh /usr/local/bin/telegram-vpn-entrypoint

RUN chmod 0755 /usr/local/bin/telegram-vpn-entrypoint \
    && mkdir -p /etc/amneziawg

ENV WG_QUICK_USERSPACE_IMPLEMENTATION=amneziawg-go

ENTRYPOINT ["/usr/local/bin/telegram-vpn-entrypoint"]
