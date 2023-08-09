# nats-chat

## Overview

nats-chat is a chat which requires nothing but nats for communication and uses
cryptography to identify user, your address is generated using double hash
algorithm (SHA256 + MD5) on your public key. The app consists of two parts: CLI
and daemon, CLI handles key management and daemon is responsible for any network
activity.

## Example

You should have running nats-server, which you have access to

```
docker run --rm -d --name nats -p 4444:4444 nats:alpine3.18 -p 4444 -D --trace
```

Clients should generate and exchange their addresses using other open channels and
then execute `run` command with appropriate arguments

```
nats-chat-daemon&
nats-chat-cli generate
# ...addresses exchange
nats-chat-cli address
# <sender_address>
nats-chat-cli online --nats-url "nats://0.0.0.0:4444"
nats-chat-cli createchat --recepient <recepient_address> 
nats-chat-cli openchat --recepient <recepient_address> 
```
