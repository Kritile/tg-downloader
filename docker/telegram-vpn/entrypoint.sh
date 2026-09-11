#!/usr/bin/env bash

set -euo pipefail

config=/etc/amneziawg/awg0.conf
if [[ ! -r "$config" ]]; then
    echo "missing AmneziaWG config: $config" >&2
    exit 1
fi

if [[ ! -c /dev/net/tun ]]; then
    echo "/dev/net/tun is required for the AmneziaWG userspace client" >&2
    exit 1
fi

docker_subnet=$(ip -4 route show dev eth0 proto kernel scope link | awk 'NR == 1 {print $1}')
gateway=$(ip -4 route show default dev eth0 | awk 'NR == 1 {print $3}')
endpoint=$(awk -F= '
    /^[[:space:]]*Endpoint[[:space:]]*=/ {
        value = $2
        gsub(/[[:space:]]/, "", value)
        print value
        exit
    }
' "$config")

if [[ -z "$docker_subnet" || -z "$gateway" || -z "$endpoint" ]]; then
    echo "could not determine Docker route or AWG endpoint route" >&2
    exit 1
fi

endpoint_host=${endpoint%:*}
if [[ "$endpoint_host" == "$endpoint" || "$endpoint_host" == *:* ]]; then
    echo "the Telegram VPN endpoint must be an IPv4 host:port for this container setup" >&2
    exit 1
fi

cleanup() {
    ip -4 rule del priority 100 not fwmark 51820 table 51820 2>/dev/null || true
    ip -4 rule del priority 200 table main suppress_prefixlength 0 2>/dev/null || true
    ip -4 route flush table 51820 2>/dev/null || true
    awg-quick down "$config" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

# Table=off in the supplied config prevents awg-quick from replacing the
# namespace default route. The policy table below routes all other IPv4
# traffic into awg0, while Docker-internal traffic and the UDP endpoint stay
# on eth0 to avoid both broken service discovery and recursive VPN routing.
awg-quick up "$config"
ip -4 route add "$docker_subnet" dev eth0 table 51820
ip -4 route add "$endpoint_host/32" via "$gateway" dev eth0 table 51820
ip -4 route add default dev awg0 table 51820
ip -4 rule add priority 100 not fwmark 51820 table 51820
ip -4 rule add priority 200 table main suppress_prefixlength 0

echo "AmneziaWG tunnel established; internal subnet ${docker_subnet} and endpoint ${endpoint_host} remain direct"
exec tail -f /dev/null
