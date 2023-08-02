import unittest
import subprocess
import signal
import os
import logging
import sys
import time
import shutil
import docker

home = os.getenv("NATS_CHAT_HOME")

class TestNatsChat(unittest.TestCase):
    def test_hello(self):
        logger = logging.getLogger("LOGGER")
        client = docker.from_env()
        container = client.containers.run("nats:alpine3.18", detach=True, auto_remove=True, 
                                          ports={"4444/tcp": ("0.0.0.0", 4444)},
                                          command="nats-server --config /etc/nats/nats-server.conf -p 4444 -D --trace")
        while container.status != "running":
            container.reload()
            time.sleep(0.1)

        args = (
            os.path.join(home, "build/nats-chat"),
            "run",
            "--profile",
            os.path.join(home, "test/{sender}"),
            "--recepient-key",
            os.path.join(home, "test/{recepient}/public.pem"),
            "--nats-url",
            "nats://0.0.0.0:4444",
        )

        try:
            args1 = tuple(arg.format(sender="testprofile1", recepient="testprofile2") for arg in args)
            args2 = tuple(arg.format(sender="testprofile2", recepient="testprofile1") for arg in args)

            p1 = subprocess.Popen(args1, stdout=subprocess.PIPE, stdin=subprocess.PIPE)
            p2 = subprocess.Popen(args2, stdout=subprocess.PIPE, stdin=subprocess.PIPE)

            p1.stdin.write(bytes("Hello!\n", 'utf-8'))
            p1.stdin.flush()
            p2.stdin.write(bytes("Nice to meet you!\n", 'utf-8'))
            p2.stdin.flush()
            t1 = p1.stdout.readline().decode('utf-8')
            t2 = p2.stdout.readline().decode('utf-8')
            p1.send_signal(signal.SIGINT)
            p1.stdin.close()
            p1.stdout.close()
            p1.wait()
            p2.send_signal(signal.SIGINT)
            p2.stdin.close()
            p2.stdout.close()
            p2.wait()
            logger.info(t1)
            logger.info(t2)
            self.assertTrue("Nice to meet you!" in t1)    
            self.assertTrue("Hello!" in t2)
        finally:
            container.stop()
            client.close()

    def test_generate(self):
        logger = logging.getLogger("LOGGER")
        temp = os.path.join(home, "temp")
        args = (
            os.path.join(home, "build/nats-chat"),
            "generate",
            "--out",
            temp,
        )

        os.mkdir(temp)
        try:
            p1 = subprocess.Popen(args)
            p1.wait()
            self.assertTrue(os.path.isfile(os.path.join(temp, "public.pem")))
            self.assertTrue(os.path.isfile(os.path.join(temp, "private.pem")))
        finally:
            shutil.rmtree(temp)

if __name__ == '__main__':
    logging.basicConfig(stream=sys.stderr)
    logging.getLogger("LOGGER").setLevel(logging.DEBUG)
    unittest.main()
