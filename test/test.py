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
    
    def test_double_generate_errors(self) -> None:
        logger = logging.getLogger("LOGGER")
        client = docker.from_env()

        try:
            c: dmc.Container
            c = client.containers.get("nats-chat-cli-1-1")
            self.assertEqual(c.exec_run("nats-chat-cli generate")[0], 0)
            self.assertEqual(c.exec_run("nats-chat-cli generate")[0], 1)
            
        finally:
            client.close()

    def test_double_address(self) -> None:
        logger = logging.getLogger("LOGGER")
        client = docker.from_env()

        try:
            c: dmc.Container
            c = client.containers.get("nats-chat-cli-1-1")
            self.assertEqual(c.exec_run("nats-chat-cli generate")[0], 0)

            code1, out1 = c.exec_run("nats-chat-cli address")
            addr1 = out1.splitlines()[1].decode('utf-8')
            self.assertEqual(code1, 0)

            code1, out1 = c.exec_run("nats-chat-cli address")
            addr2 = out1.splitlines()[1].decode('utf-8')
            self.assertEqual(code1, 0)
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

    def test_custom_generate(self) -> None:
        logger = logging.getLogger("LOGGER")
        client = docker.from_env()

        try:
            c: dmc.Container
            c = client.containers.get("nats-chat-cli-1-1")
            self.assertEqual(c.exec_run("nats-chat-cli generate --out /root/temp")[0], 0)
            self.assertEqual(c.exec_run("stat /root/temp/private.pem")[0], 0)
            self.assertEqual(c.exec_run("stat /root/temp/public.pem")[0], 0)
            
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
            self.assertEqual(c1.exec_run("nats-chat-cli generate")[0], 0)
            self.assertEqual(c2.exec_run("nats-chat-cli generate")[0], 0)

            code1, out1 = c1.exec_run("nats-chat-cli address")
            addr1 = out1.splitlines()[1].decode('utf-8')
            self.assertEqual(code1, 0)
            code2, out2 = c2.exec_run("nats-chat-cli address")
            addr2 = out2.splitlines()[1].decode('utf-8')
            self.assertEqual(code2, 0)

            code1, out1 = c1.exec_run("nats-chat-cli online --nats-url \"nats://nats:4444\"")
            self.assertEqual(code1, 0)
            code2, out2 = c2.exec_run("nats-chat-cli online --nats-url \"nats://nats:4444\"")
            self.assertEqual(code2, 0)

            code1, out1 = c1.exec_run("nats-chat-cli createchat --recepient {addr2}".format(addr2=addr2))
            self.assertEqual(code1, 0)
            code2, out2 = c2.exec_run("nats-chat-cli createchat --recepient {addr1}".format(addr1=addr1))
            self.assertEqual(code2, 0)

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
            self.assertEqual(code1, 0)
            code2, out2 = c2.exec_run("nats-chat-cli rmchat --recepient {addr1}".format(addr1=addr1))
            self.assertEqual(code2, 0)

            code1, out1 = c1.exec_run("nats-chat-cli offline")
            self.assertEqual(code1, 0)
            code2, out2 = c2.exec_run("nats-chat-cli offline")
            self.assertEqual(code2, 0)
            
        finally:
            client.close()

if __name__ == '__main__':
    logging.basicConfig(stream=sys.stderr)
    logging.getLogger("LOGGER").setLevel(logging.DEBUG)
    unittest.main()
