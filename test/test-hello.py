import unittest
import subprocess
import os
import logging
import sys
import shutil
from typing import List, Tuple, Dict, Any

home = os.getenv("NATS_CHAT_HOME")
user_home = os.getenv("HOME")
user_nats_profile = os.path.join(user_home, ".natschat")

def format_tuple(seq: Tuple, kv: Dict[str, str]) -> Tuple:
    return tuple(el.format(**kv) for el in seq)

class TestGenerate(unittest.TestCase):
    profile_path = os.path.join(home, "profile")
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
        os.mkdir(TestGenerate.profile_path)

    def tearDown(cls) -> None:
        shutil.rmtree(TestGenerate.profile_path)
        if os.path.isdir(user_nats_profile):
            shutil.rmtree(user_nats_profile)

    def test_generate(self) -> None:
        TestGenerate.dogenerate()
        self.assertTrue(os.path.isfile(os.path.join(user_nats_profile, "public.pem")))
        self.assertTrue(os.path.isfile(os.path.join(user_nats_profile, "private.pem")))

    def test_generate_out(self) -> None:
        TestGenerate.dogenerate(TestGenerate.profile_path)
        self.assertTrue(os.path.isfile(os.path.join(TestGenerate.profile_path, "public.pem")))
        self.assertTrue(os.path.isfile(os.path.join(TestGenerate.profile_path, "private.pem")))

class TestRun(unittest.TestCase):
    profilePath1 = os.path.join(home, "profile1")
    profilePath2 = os.path.join(home, "profile2")

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
        os.mkdir(TestRun.profilePath1)
        os.mkdir(TestRun.profilePath2)
        TestGenerate.dogenerate(TestRun.profilePath1)
        TestGenerate.dogenerate(TestRun.profilePath2)

    def tearDown(cls) -> None:
        shutil.rmtree(TestRun.profilePath1)
        shutil.rmtree(TestRun.profilePath2)
    
    def test_hello(self):
        logger = logging.getLogger("LOGGER")
        pkey1 = os.path.join(TestRun.profilePath1, "public.pem")
        pkey2 = os.path.join(TestRun.profilePath2, "public.pem")
        natsUrl = "nats://0.0.0.0:4444"

        try:
            p1 = TestRun.dorun(TestRun.profilePath1, pkey2, natsUrl)
            p2 = TestRun.dorun(TestRun.profilePath2, pkey1, natsUrl)

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
    
    def test_offline_closes(self):
        logger = logging.getLogger("LOGGER")
        pkey1 = os.path.join(TestRun.profilePath1, "public.pem")
        pkey2 = os.path.join(TestRun.profilePath2, "public.pem")
        natsUrl = "nats://0.0.0.0:4444"

        try:
            p1 = TestRun.dorun(TestRun.profilePath1, pkey2, natsUrl)
            p2 = TestRun.dorun(TestRun.profilePath2, pkey1, natsUrl)

            p1.stdin.close()
            p1.wait()
            # Interrupting Scanln not implemented
            p2.stdin.write(bytes("Nice to meet you!\n", 'utf-8'))
            p2.stdin.flush()
            p2.wait(timeout=10)
        except TimeoutError:
            self.fail("Timeout exceeded")
        finally:
            p1.stdout.close()
            p2.stdin.close()
            p2.stdout.close()
        self.assertTrue(True)

if __name__ == '__main__':
    logging.basicConfig(stream=sys.stderr)
    logging.getLogger("LOGGER").setLevel(logging.DEBUG)
    unittest.main()
