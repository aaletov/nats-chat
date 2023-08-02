# nats-chat

```
docker run --rm -d --name nats -p 4444:4444 nats:alpine3.18 -p 4444 -D --trace
./build/nats-chat run --profile ./temp/client1 --recepient-key ./temp/client2/public.pem --nats-url nats://0.0.0.0:4444
./build/nats-chat run --profile ./temp/client2 --recepient-key ./temp/client1/public.pem --nats-url nats://0.0.0.0:4444
```
