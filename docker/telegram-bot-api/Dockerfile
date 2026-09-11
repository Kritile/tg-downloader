FROM aiogram/telegram-bot-api:latest

USER root

# Bot API Server's --proxy option is only for outgoing webhooks, not TDLib.
# proxychains-ng scopes the SOCKS5 routing to the Bot API process itself.
RUN apk add --no-cache proxychains-ng

COPY docker/telegram-bot-api/entrypoint.sh /usr/local/bin/telegram-bot-api-proxy-entrypoint
RUN chmod 0755 /usr/local/bin/telegram-bot-api-proxy-entrypoint

ENTRYPOINT ["/usr/local/bin/telegram-bot-api-proxy-entrypoint"]
