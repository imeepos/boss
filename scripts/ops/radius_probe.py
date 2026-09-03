#!/usr/bin/env python3
# 最小 RADIUS Access-Request 探针(零依赖,契约对齐 internal/domain/aaa/radius/handler.go):
# User-Name=LOID + NAS-IP-Address;Access-Accept 携带 FramedPool(套餐带宽)/Session-Timeout。
# 服务端仅按 LOID 鉴权(serveAuth 不读口令),探针与 cmd/oltsim 客户端一致不携带 User-Password。
# 用法: radius_probe.py --server HOST:1812 --secret SECRET --loid LOID [--timeout 5]
# 输出单行: probe code=<Access-Accept|Access-Reject|TIMEOUT|SHORT|BADAUTH> loid=.. bandwidth=.. reply=..
# 退出码: 0=收到合法应答; 2=超时; 3=应答非法(过短/认证器校验失败,典型为共享密钥不符)。
import argparse
import hashlib
import os
import socket
import struct
import sys

ATTR_USER_NAME = 1
ATTR_NAS_IP = 4
ATTR_REPLY_MESSAGE = 18
ATTR_FRAMED_POOL = 88
CODE_ACCEPT = 2
CODE_REJECT = 3


def attr(code: bytes, value: bytes) -> bytes:
    return bytes([code, len(value) + 2]) + value


def build_request(loid: str) -> tuple[bytes, bytes]:
    packet_id = os.urandom(1)[0]
    req_auth = os.urandom(16)
    body = attr(ATTR_USER_NAME, loid.encode()) + attr(ATTR_NAS_IP, socket.inet_aton("127.0.0.1"))
    header = struct.pack("!BBH", 1, packet_id, 20 + len(body))
    return header + req_auth + body, req_auth


def parse_attrs(data: bytes) -> dict:
    out = {}
    i = 0
    while i + 2 <= len(data):
        atype, alen = data[i], data[i + 1]
        if alen < 2 or i + alen > len(data):
            break
        out.setdefault(atype, data[i + 2:i + alen])
        i += alen
    return out


def probe(server: str, secret: str, loid: str, timeout: float) -> int:
    host, _, port = server.partition(":")
    packet, req_auth = build_request(loid)
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.settimeout(timeout)
    try:
        sock.sendto(packet, (host, int(port or 1812)))
        resp, _ = sock.recvfrom(4096)
    except socket.timeout:
        print("probe code=TIMEOUT loid=" + loid)
        return 2
    finally:
        sock.close()
    if len(resp) < 20:
        print("probe code=SHORT loid=" + loid)
        return 3
    code, _, length = struct.unpack("!BBH", resp[:4])
    # Response Authenticator = MD5(Code+ID+Length+RequestAuth+Attributes+Secret)(RFC 2865 3)
    digest = hashlib.md5(resp[:4] + req_auth + resp[20:length] + secret.encode()).digest()
    if digest != resp[4:20]:
        print("probe code=BADAUTH loid=" + loid)
        return 3
    attrs = parse_attrs(resp[20:length])
    bw = attrs.get(ATTR_FRAMED_POOL, b"").decode(errors="replace")
    reply = attrs.get(ATTR_REPLY_MESSAGE, b"").decode(errors="replace")
    if code == CODE_ACCEPT:
        name = "Access-Accept"
    elif code == CODE_REJECT:
        name = "Access-Reject"
    else:
        name = str(code)
    print("probe code=%s loid=%s bandwidth=%s reply=%s" % (name, loid, bw, reply))
    return 0


def main() -> int:
    ap = argparse.ArgumentParser(description="minimal RADIUS Access-Request probe")
    ap.add_argument("--server", required=True, help="HOST:PORT (UDP 1812)")
    ap.add_argument("--secret", required=True, help="NAS shared secret")
    ap.add_argument("--loid", required=True)
    ap.add_argument("--timeout", type=float, default=5.0)
    args = ap.parse_args()
    return probe(args.server, args.secret, args.loid, args.timeout)


if __name__ == "__main__":
    sys.exit(main())
