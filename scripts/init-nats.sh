#!/bin/sh
set -e

NATS_URL="nats://nats:4222"

until nats --server="$NATS_URL" server check jetstream >/dev/null 2>&1; do
  sleep 1
done

nats --server="$NATS_URL" stream add USERS \
  --subjects="user.created,user.created.dlq" \
  --storage=file \
  --retention=limits \
  --discard=old \
  --max-age=168h \
  --replicas=1 \
  --defaults >/dev/null 2>&1 || \
nats --server="$NATS_URL" stream edit USERS \
  --subjects="user.created,user.created.dlq" --force >/dev/null

nats --server="$NATS_URL" stream add MESSAGES \
  --subjects="messages.send,messages.created,chat.messages.created.dlq,message.messages.send.dlq" \
  --storage=file \
  --retention=limits \
  --discard=old \
  --max-age=168h \
  --replicas=1 \
  --defaults >/dev/null 2>&1 || \
nats --server="$NATS_URL" stream edit MESSAGES \
  --subjects="messages.send,messages.created,chat.messages.created.dlq,message.messages.send.dlq" --force >/dev/null

nats --server="$NATS_URL" consumer add USERS user-created \
  --filter="user.created" \
  --ack=explicit \
  --pull \
  --deliver=all \
  --max-deliver=-1 \
  --wait=30s \
  --defaults >/dev/null 2>&1 || true

nats --server="$NATS_URL" consumer add MESSAGES message-send \
  --filter="messages.send" \
  --ack=explicit \
  --pull \
  --deliver=all \
  --max-deliver=-1 \
  --wait=30s \
  --defaults >/dev/null 2>&1 || true

nats --server="$NATS_URL" consumer add MESSAGES chat-message-created \
  --filter="messages.created" \
  --ack=explicit \
  --pull \
  --deliver=all \
  --max-deliver=-1 \
  --wait=30s \
  --defaults >/dev/null 2>&1 || true