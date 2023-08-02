import unittest
import subprocess
import signal
import os
import logging
import sys
import time
import shutil
import docker
from typing import List, Tuple, Dict, Any

user_home = os.getenv("HOME")
home = os.getenv("NATS_CHAT_HOME")
temp = os.path.join(home, "temp")

def format_tuple(seq: Tuple, kv: Dict[str, str]) -> Tuple:
    return tuple(el.format(**kv) for el in seq)

class TestGenerate(unittest.TestCase):
    @staticmethod
    def dogenerate(out=None) -> None:
        args = (
            os.path.join(home, "build/nats-chat"),
            "generate",
        )
        if out != None:
            args = (*args, "--out", out)
        p1 = subprocess.Popen(args)
        p1.wait()

    def setUp(cls) -> None:
        os.mkdir(temp)

    def tearDown(cls) -> None:
        shutil.rmtree(temp)

    def test_generate(self) -> None:
        TestGenerate.dogenerate(temp)
        self.assertTrue(os.path.isfile(os.path.join(temp, "public.pem")))
        self.assertTrue(os.path.isfile(os.path.join(temp, "private.pem")))

class TestRun(unittest.TestCase):
    @staticmethod
    def dorun(profile=None, recepientKey=None, natsUrl=None) -> Any:
        args = (
            os.path.join(home, "build/nats-chat"),
            "run",
        )
        if profile != None:
            args = (*args, "--profile", profile)
        if recepientKey != None:
            args = (*args, "--recepient-key", recepientKey)
        if natsUrl != None:
            args = (*args, "--nats-url", natsUrl)

        return subprocess.Popen(args, stdout=subprocess.PIPE, stdin=subprocess.PIPE)
    
    def setUp(cls) -> None:
        os.mkdir(temp)
        profile1 = os.path.join(temp, "profile1")
        profile2 = os.path.join(temp, "profile2")
        os.mkdir(profile1)
        os.mkdir(profile2)
        TestGenerate.dogenerate(profile1)
        TestGenerate.dogenerate(profile2)

    def tearDown(cls) -> None:
        shutil.rmtree(temp)
    
    def test_hello(self):
        logger = logging.getLogger("LOGGER")
        profile1 = os.path.join(temp, "profile1")
        profile2 = os.path.join(temp, "profile2")
        pkey1 = os.path.join(profile1, "public.pem")
        pkey2 = os.path.join(profile2, "public.pem")
        natsUrl = "nats://0.0.0.0:4444"

        try:
            p1 = TestRun.dorun(profile1, pkey2, natsUrl)
            p2 = TestRun.dorun(profile2, pkey1, natsUrl)

            p1.stdin.write(bytes("Hello!\n", 'utf-8'))
            p1.stdin.flush()
            p2.stdin.write(bytes("Nice to meet you!\n", 'utf-8'))
            p2.stdin.flush()
            t1 = p1.stdout.readline().decode('utf-8')
            t2 = p2.stdout.readline().decode('utf-8')
            p1.stdin.close()
            p1.wait()
            p2.stdin.close()
            p2.wait()
            self.assertTrue("Nice to meet you!" in t1)    
            self.assertTrue("Hello!" in t2)
        finally:
            p1.stdout.close()
            p2.stdout.close()

if __name__ == '__main__':
    logging.basicConfig(stream=sys.stderr)
    logging.getLogger("LOGGER").setLevel(logging.DEBUG)
    unittest.main()
