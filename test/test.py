import unittest
import subprocess
import os
import logging
import sys
import shutil
from typing import List, Tuple, Dict, Any
import time
import docker
import docker.models.containers as dmc 
import socket

home = os.getenv("NATS_CHAT_HOME")

class ComposeTestCase(unittest.TestCase):
    @staticmethod
    def getComposeFile() -> str:
        raise RuntimeError()
    
    def setUp(cls) -> None:
        args = (
            "docker",
            "compose",
            "--file",
            os.path.join(home, cls.getComposeFile()),
            "up",
            "--wait",
        )
        subprocess.Popen(args, stdout=subprocess.DEVNULL,
                         stderr=subprocess.STDOUT).wait()
        

    def tearDown(cls) -> None:
        args = (
            "docker",
            "compose",
            "--file",
            os.path.join(home, cls.getComposeFile()),
            "stop",
        )
        subprocess.Popen(args, stdout=subprocess.DEVNULL,
                         stderr=subprocess.STDOUT).wait()

        # Use --follow in separate thread?
        args = (
            "docker",
            "compose",
            "--file",
            os.path.join(home, cls.getComposeFile()),
            "logs",
        )
        subprocess.Popen(args).wait()

        args = (
            "docker",
            "compose",
            "--file",
            os.path.join(home, cls.getComposeFile()),
            "rm",
            "-f",
        )
        subprocess.Popen(args, stdout=subprocess.DEVNULL,
                         stderr=subprocess.STDOUT).wait()

        args = (
            "docker",
            "volume",
            "rm",
            "nats-chat_daemon-sock-1",
            "nats-chat_daemon-sock-2",
        )
        subprocess.Popen(args, stdout=subprocess.DEVNULL,
                         stderr=subprocess.STDOUT).wait()
        
class CliTestCase(ComposeTestCase):
    @staticmethod
    def getComposeFile() -> str:
        return "docker-compose.cli.yml"

    def test_double_address(self) -> None:
        logger = logging.getLogger("LOGGER")
        client = docker.from_env()

        try:
            c: dmc.Container
            c = client.containers.get("nats-chat-cli-1-1")
            self.assertTrue(c.exec_run("nats-chat-cli generate")[0] == 0)

            code1, out1 = c.exec_run("nats-chat-cli address")
            addr1 = out1.splitlines()[1].decode('utf-8')
            self.assertTrue(code1 == 0)

            code1, out1 = c.exec_run("nats-chat-cli address")
            addr2 = out1.splitlines()[1].decode('utf-8')
            self.assertTrue(code1 == 0)
            self.assertEqual(addr1, addr2)
            
        finally:
            client.close()

    def test_default_generate(self) -> None:
        logger = logging.getLogger("LOGGER")
        client = docker.from_env()

        try:
            c: dmc.Container
            c = client.containers.get("nats-chat-cli-1-1")
            self.assertEqual(c.exec_run("nats-chat-cli generate")[0], 0)
            self.assertEqual(c.exec_run("stat /root/.natschat/private.pem")[0], 0)
            self.assertEqual(c.exec_run("stat /root/.natschat/public.pem")[0], 0)
            
        finally:
            client.close()

class TestNatsChat(ComposeTestCase):
    @staticmethod
    def getComposeFile() -> str:
        return "docker-compose.full.yml"

    def test_one_message(self) -> None:
        logger = logging.getLogger("LOGGER")
        client = docker.from_env()

        try:
            c1: dmc.Container
            c1 = client.containers.get("nats-chat-cli-1-1")
            c2: dmc.Container
            c2 = client.containers.get("nats-chat-cli-2-1")
            self.assertTrue(c1.exec_run("nats-chat-cli generate")[0] == 0)
            self.assertTrue(c2.exec_run("nats-chat-cli generate")[0] == 0)

            code1, out1 = c1.exec_run("nats-chat-cli address")
            addr1 = out1.splitlines()[1].decode('utf-8')
            self.assertTrue(code1 == 0)
            code2, out2 = c2.exec_run("nats-chat-cli address")
            addr2 = out2.splitlines()[1].decode('utf-8')
            self.assertTrue(code2 == 0)

            code1, out1 = c1.exec_run("nats-chat-cli online --nats-url \"nats://nats:4444\"")
            self.assertTrue(code1 == 0)
            code2, out2 = c2.exec_run("nats-chat-cli online --nats-url \"nats://nats:4444\"")
            self.assertTrue(code2 == 0)

            code1, out1 = c1.exec_run("nats-chat-cli createchat --recepient {addr2}".format(addr2=addr2))
            self.assertTrue(code1 == 0)
            code2, out2 = c2.exec_run("nats-chat-cli createchat --recepient {addr1}".format(addr1=addr1))
            self.assertTrue(code2 == 0)

            s1: socket.SocketIO            
            code1, s1 = c1.exec_run("nats-chat-cli openchat", socket=True, stdin=True)
            self.assertTrue(code1 == None)
            s2: socket.SocketIO            
            code2, s2 = c2.exec_run("nats-chat-cli openchat", socket=True, stdin=True)
            self.assertTrue(code2 == None)
        
            try:
                s1._sock.send(b"I did not hit her\n")
                s2._sock.send(b"Oh, hi Mark!\n")

                s1.readline()
                out1 = s1.readline().decode("utf-8")

                s2.readline()   
                out2 = s2.readline().decode("utf-8")

                self.assertTrue("Oh, hi Mark!" in out1)
                self.assertTrue("I did not hit her" in out2)
            finally:
                s1.close()
                s2.close()

            code1, out1 = c1.exec_run("nats-chat-cli rmchat --recepient {addr2}".format(addr2=addr2))
            self.assertTrue(code1 == 0)
            code2, out2 = c2.exec_run("nats-chat-cli rmchat --recepient {addr1}".format(addr1=addr1))
            self.assertTrue(code2 == 0)

            code1, out1 = c1.exec_run("nats-chat-cli offline")
            self.assertTrue(code1 == 0)
            code2, out2 = c2.exec_run("nats-chat-cli offline")
            self.assertTrue(code2 == 0)
            
        finally:
            client.close()

# user_home = os.getenv("HOME")
# user_nats_profile = os.path.join(user_home, ".natschat")

# def format_tuple(seq: Tuple, kv: Dict[str, str]) -> Tuple:
#     return tuple(el.format(**kv) for el in seq)

# class TestGenerate(unittest.TestCase):
#     profile_path = os.path.join(home, "profile")
#     @staticmethod
#     def dogenerate(out=None) -> None:
#         args = (
#             os.path.join(home, "build/nats-chat"),
#             "generate",
#         )
#         if out != None:
#             args = (*args, "--out", out)
#         p1 = subprocess.Popen(args)
#         p1.wait()

#     def setUp(cls) -> None:
#         os.mkdir(TestGenerate.profile_path)

#     def tearDown(cls) -> None:
#         shutil.rmtree(TestGenerate.profile_path)
#         if os.path.isdir(user_nats_profile):
#             shutil.rmtree(user_nats_profile)

#     def test_generate(self) -> None:
#         TestGenerate.dogenerate()
#         self.assertTrue(os.path.isfile(os.path.join(user_nats_profile, "public.pem")))
#         self.assertTrue(os.path.isfile(os.path.join(user_nats_profile, "private.pem")))

#     def test_generate_out(self) -> None:
#         TestGenerate.dogenerate(TestGenerate.profile_path)
#         self.assertTrue(os.path.isfile(os.path.join(TestGenerate.profile_path, "public.pem")))
#         self.assertTrue(os.path.isfile(os.path.join(TestGenerate.profile_path, "private.pem")))

# class TestRun(unittest.TestCase):
#     profilePath1 = os.path.join(home, "profile1")
#     profilePath2 = os.path.join(home, "profile2")

#     @staticmethod
#     def dorun(profile=None, recepient=None, natsUrl=None) -> Any:
#         args = (
#             os.path.join(home, "build/nats-chat"),
#             "run",
#         )
#         if profile != None:
#             args = (*args, "--profile", profile)
#         if recepient != None:
#             args = (*args, "--recepient-key", recepient)
#         if natsUrl != None:
#             args = (*args, "--nats-url", natsUrl)

#         return subprocess.Popen(args, stdout=subprocess.PIPE, stdin=subprocess.PIPE)
    
#     def setUp(cls) -> None:
#         os.mkdir(TestRun.profilePath1)
#         os.mkdir(TestRun.profilePath2)
#         TestGenerate.dogenerate(TestRun.profilePath1)
#         TestGenerate.dogenerate(TestRun.profilePath2)

#     def tearDown(cls) -> None:
#         shutil.rmtree(TestRun.profilePath1)
#         shutil.rmtree(TestRun.profilePath2)
    
#     def test_hello(self):
#         logger = logging.getLogger("LOGGER")
#         pkey1 = os.path.join(TestRun.profilePath1, "public.pem")
#         pkey2 = os.path.join(TestRun.profilePath2, "public.pem")
#         natsUrl = "nats://0.0.0.0:4444"

#         try:
#             p1 = TestRun.dorun(TestRun.profilePath1, pkey2, natsUrl)
#             p2 = TestRun.dorun(TestRun.profilePath2, pkey1, natsUrl)

#             p1.stdin.write(bytes("Hello!\n", 'utf-8'))
#             p1.stdin.flush()
#             p2.stdin.write(bytes("Nice to meet you!\n", 'utf-8'))
#             p2.stdin.flush()
#             t1 = p1.stdout.readline().decode('utf-8')
#             t2 = p2.stdout.readline().decode('utf-8')
#             p1.stdin.close()
#             p1.wait()
#             p2.stdin.close()
#             p2.wait()
#             self.assertTrue("Nice to meet you!" in t1)    
#             self.assertTrue("Hello!" in t2)
#         finally:
#             p1.stdout.close()
#             p2.stdout.close()
    
#     def test_offline_closes(self):
#         logger = logging.getLogger("LOGGER")
#         pkey1 = os.path.join(TestRun.profilePath1, "public.pem")
#         pkey2 = os.path.join(TestRun.profilePath2, "public.pem")
#         natsUrl = "nats://0.0.0.0:4444"

#         try:
#             p1 = TestRun.dorun(TestRun.profilePath1, pkey2, natsUrl)
#             p2 = TestRun.dorun(TestRun.profilePath2, pkey1, natsUrl)

#             p1.stdin.close()
#             p1.wait()
#             # Interrupting Scanln not implemented
#             p2.stdin.write(bytes("Nice to meet you!\n", 'utf-8'))
#             p2.stdin.flush()
#             p2.wait(timeout=10)
#         except TimeoutError:
#             self.fail("Timeout exceeded")
#         finally:
#             p1.stdout.close()
#             p2.stdin.close()
#             p2.stdout.close()
#         self.assertTrue(True)

if __name__ == '__main__':
    logging.basicConfig(stream=sys.stderr)
    logging.getLogger("LOGGER").setLevel(logging.DEBUG)
    unittest.main()
