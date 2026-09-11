#!/usr/bin/env sh

set -eu

proxy=${XRAY_SOCKS5_PROXY:-}
if [ -z "$proxy" ]; then
    echo "XRAY_SOCKS5_PROXY is required for telegram-bot-api outbound traffic" >&2
    exit 1
fi

proxy=${proxy#socks5://}
proxy=${proxy#socks5h://}
credentials=""
server="$proxy"
case "$proxy" in
    *@*)
        credentials=${proxy%@*}
        server=${proxy#*@}
        ;;
esac

host=${server%:*}
port=${server##*:}
if [ -z "$host" ] || [ -z "$port" ] || [ "$host" = "$server" ]; then
    echo "XRAY_SOCKS5_PROXY must be [socks5://][user:password@]host:port" >&2
    exit 1
fi

proxychains_config=$(mktemp)
cleanup() {
    rm -f "$proxychains_config"
}
trap cleanup EXIT INT TERM

{
    echo 'strict_chain'
    echo 'proxy_dns'
    echo 'tcp_read_time_out 15000'
    echo 'tcp_connect_time_out 8000'
    echo '[ProxyList]'
    if [ -n "$credentials" ]; then
        user=${credentials%%:*}
        password=${credentials#*:}
        printf 'socks5 %s %s %s %s\n' "$host" "$port" "$user" "$password"
    else
        printf 'socks5 %s %s\n' "$host" "$port"
    fi
} >"$proxychains_config"

# The upstream entrypoint starts telegram-bot-api and binds its HTTP listener
# in this container. proxychains only intercepts outbound connect calls.
exec proxychains4 -f "$proxychains_config" /docker-entrypoint.sh "$@"
