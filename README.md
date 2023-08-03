# nats-chat

## Example

You should have running nats-server, which you have access to

```
docker run --rm -d --name nats -p 4444:4444 nats:alpine3.18 -p 4444 -D --trace
```

Clients should generate and exchange public keys using other open channels and
then execute `run` command with appropriate arguments

```
nats-chat generate
# ...key exchange
nats-chat run --recepient-key $PATH_TO_RECEPIENT_PKEY \
  --nats-url $NATS_URL
```
