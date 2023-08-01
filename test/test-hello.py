import unittest
import subprocess
import os
import logging
import sys
import time
import shutil

home = os.getenv("NATS_CHAT_HOME")

class TestNatsChat(unittest.TestCase):
    def test_hello(self):
        logger = logging.getLogger("LOGGER")

        args1 = (
            os.path.join(home, "build/nats-chat"),
            "run",
            "--profile",
            os.path.join(home, "test/testprofile1"),
            "--recepient-key",
            os.path.join(home, "test/testprofile2/public.pem"),
            "--nats-url",
            "nats://0.0.0.0:4444",
        )

        args2 = (
            os.path.join(home, "build/nats-chat"),
            "run",
            "--profile",
            os.path.join(home, "test/testprofile2"),
            "--recepient-key",
            os.path.join(home, "test/testprofile1/public.pem"),
            "--nats-url",
            "nats://0.0.0.0:4444",
        )

        p1 = subprocess.Popen(args1, stdout=subprocess.PIPE, stdin=subprocess.PIPE)
        p2 = subprocess.Popen(args2, stdout=subprocess.PIPE, stdin=subprocess.PIPE)

        time.sleep(1)

        p1.stdin.write(bytes("Hello!\n", 'utf-8'))
        p1.stdin.flush()
        p2.stdin.write(bytes("Nice to meet you!\n", 'utf-8'))
        p2.stdin.flush()
        t1 = p1.stdout.readline().decode('utf-8')
        t2 = p2.stdout.readline().decode('utf-8')
        p1.kill()
        p1.stdin.close()
        p1.stdout.close()
        p1.wait()
        p2.kill()
        p2.stdin.close()
        p2.stdout.close()
        p2.wait()
        logger.info(t1)
        logger.info(t2)
        self.assertTrue("Nice to meet you!" in t1)    
        self.assertTrue("Hello!" in t2)

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
