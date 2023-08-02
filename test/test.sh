#/bin/sh
mkdir /var/log/nats
nats-server --config /etc/nats/nats-server.conf -p 4444 -D --trace \
  --log /var/log/nats/test.log&
python3 ./test/test-hello.py