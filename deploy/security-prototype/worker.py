"""Bounded static scanner. Repository code is never executed.

Copyright The MatrixHub Authors. Licensed under Apache-2.0.
"""
import hashlib
import io
import json
import os
import pathlib
import pickletools
import socket
import struct
import subprocess
import sys
import tempfile
import threading
import time
import zipfile
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlsplit


def budget(name, default, maximum):
    value = int(os.environ.get(name, default))
    if not 1 <= value <= maximum:
        raise ValueError(f"invalid {name}")
    return value


LIMIT = budget("SCAN_MAX_FILE_BYTES", 1 << 30, 1 << 30)
ARCHIVE_LIMIT = budget("SCAN_ARCHIVE_BYTES", 1280 << 20, 2 << 30)
PICKLE_LIMIT = budget("SCAN_PICKLE_BYTES", 8 << 20, 8 << 20)
TOTAL_PICKLE_LIMIT = budget("SCAN_TOTAL_PICKLE_BYTES", 32 << 20, 64 << 20)
MAX_MEMBERS = budget("SCAN_ARCHIVE_MEMBERS", 100, 4096)
TIMEOUT = budget("SCAN_TIMEOUT_SECONDS", 540, 3300)
SPOOL_DIR = os.environ.get("SCAN_SPOOL_DIR", "/tmp")
PICKLE_SUFFIXES = {".pkl", ".pickle", ".pt", ".pth", ".bin", ".ckpt", ".joblib", ".dill"}
UNSUPPORTED_SERIALIZATION_SUFFIXES = {".npy", ".npz"}
NESTED_ARCHIVE_MAGIC = (b"PK\x03\x04", b"PK\x05\x06", b"\x1f\x8b", b"BZh", b"\xfd7zXZ\x00", b"7z\xbc\xaf\x27\x1c", b"Rar!")
SCAN_SLOT = threading.BoundedSemaphore(1)


class ScanFailure(Exception):
    pass


def remaining(deadline):
    seconds = deadline - time.monotonic()
    if seconds <= 0:
        raise ScanFailure("scanner_timeout")
    return seconds


def clamd(content, deadline):
    try:
        with socket.create_connection((os.environ.get("CLAMD_HOST", "clamav"), 3310), timeout=min(5, remaining(deadline))) as sock:
            sock.sendall(b"zINSTREAM\0")
            content.seek(0)
            while True:
                sock.settimeout(min(40, remaining(deadline)))
                chunk = content.read(65536)
                if not chunk:
                    break
                sock.sendall(struct.pack("!I", len(chunk)) + chunk)
            sock.sendall(b"\0\0\0\0")
            answer = b""
            while not answer.endswith(b"\0"):
                sock.settimeout(remaining(deadline))
                part = sock.recv(4096)
                if not part or len(answer) + len(part) > 4096:
                    raise ScanFailure("scanner_incomplete")
                answer += part
    except TimeoutError as exc:
        raise ScanFailure("scanner_timeout") from exc
    except OSError as exc:
        raise ScanFailure("scanner_unavailable") from exc
    if b"Heuristics.Limits.Exceeded" in answer or b"size limit exceeded" in answer:
        raise ScanFailure("scanner_limit_exceeded")
    if b"Heuristics.Encrypted" in answer:
        raise ScanFailure("encrypted_archive")
    if answer.endswith(b" OK\0"):
        return False
    if answer.endswith(b" FOUND\0"):
        return True
    raise ScanFailure("scanner_incomplete")


def clamd_version():
    with socket.create_connection((os.environ.get("CLAMD_HOST", "clamav"), 3310), timeout=5) as sock:
        sock.settimeout(5)
        sock.sendall(b"zVERSION\0")
        return sock.recv(512).rstrip(b"\0\n").decode("ascii", errors="replace")


def ruleset_identity():
    root = pathlib.Path(__file__).parent
    identity = {"clamav": clamd_version(), "fickling": "0.1.12",
                "limit": LIMIT, "pickle_limit": PICKLE_LIMIT,
                "archive_limit": ARCHIVE_LIMIT, "members": MAX_MEMBERS,
                "total_pickle_limit": TOTAL_PICKLE_LIMIT, "timeout": TIMEOUT,
                "adapter": "streamed-static-v4", "archive_ratio": 200,
                "worker_sha256": hashlib.sha256(pathlib.Path(__file__).read_bytes()).hexdigest(),
                "analyzer_sha256": hashlib.sha256((root / "analyze.py").read_bytes()).hexdigest(),
                "clamd_config_sha256": hashlib.sha256((root / "clamd-security.conf").read_bytes()).hexdigest()}
    return hashlib.sha256(json.dumps(identity, sort_keys=True).encode()).hexdigest()


def pickle_check(data, deadline):
    try:
        run = subprocess.run(
            [sys.executable, "-B", str(pathlib.Path(__file__).with_name("analyze.py"))],
            input=data, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
            timeout=min(10, remaining(deadline)), check=False,
        )
        if run.returncode != 0 or len(run.stdout) > 4096:
            raise ScanFailure("static_analysis_incomplete")
        result = json.loads(run.stdout)
        if result.get("status") not in {"passed", "warning", "blocked"}:
            raise ScanFailure("static_analysis_incomplete")
        return result
    except subprocess.TimeoutExpired as exc:
        raise ScanFailure("static_analysis_incomplete") from exc
    except (ValueError, KeyError) as exc:
        raise ScanFailure("static_analysis_incomplete") from exc


def looks_like_pickle(content, binary_prefix=True):
    """Recognise binary protocols and bounded protocol 0/1 opcode prefixes."""
    content.seek(0)
    head = content.read(65536)
    if binary_prefix and head.startswith(b"\x80"):
        return True
    count = 0
    try:
        for opcode, _, offset in pickletools.genops(head):
            count += 1
            # Recognisable serialization operations are checked even when the
            # rest of a renamed file is truncated or larger than this prefix.
            if opcode.name in {"GLOBAL", "INST", "REDUCE", "NEWOBJ", "BUILD"}:
                return True
            if opcode.name == "STOP":
                return True
            if count >= 32:
                return True
    except (ValueError, UnicodeError):
        return count >= 16
    return False


def scan(path, content, deadline):
    result = {"status": "passed", "findings": [], "checks": [], "file_type": "other"}
    analyzed = 0
    total_metadata = 0

    def analyze(data):
        nonlocal analyzed, total_metadata
        total_metadata += len(data)
        if len(data) > PICKLE_LIMIT:
            raise ScanFailure("pickle_metadata_limit")
        if total_metadata > TOTAL_PICKLE_LIMIT:
            raise ScanFailure("scan_budget_exceeded")
        verdict = pickle_check(data, deadline)
        analyzed += 1
        if "fickling/0.1.12" not in result["checks"]:
            result["checks"].append("fickling/0.1.12")
        if verdict["status"] == "blocked" or (verdict["status"] == "warning" and result["status"] != "blocked"):
            result["status"] = verdict["status"]
        if verdict["status"] != "passed":
            result["findings"].append({"scanner": "fickling", "version": "0.1.12",
                                       "rule": verdict["rule"], "severity": verdict["severity"]})
        return verdict

    try:
        version = clamd_version()
        result["checks"].append(version)
        if clamd(content, deadline):
            result["status"] = "blocked"
            result["findings"].append({"scanner": "clamav", "version": version,
                                       "rule": "CLAMAV_SIGNATURE_MATCH", "severity": "high"})
        suffix = pathlib.PurePosixPath(path).suffix.lower()
        content.seek(0)
        if zipfile.is_zipfile(content):
            result["file_type"] = "zip"
            with zipfile.ZipFile(content) as archive:
                info = archive.infolist()
                if len(info) > MAX_MEMBERS:
                    raise ScanFailure("archive_member_limit")
                # Duplicate names can shadow the metadata seen by a model loader.
                if len({item.filename for item in info}) != len(info):
                    raise ScanFailure("unsupported_serialization")
                if sum(item.file_size for item in info) > ARCHIVE_LIMIT:
                    raise ScanFailure("archive_expansion_limit")
                for item in info:
                    remaining(deadline)
                    if item.flag_bits & 1:
                        raise ScanFailure("encrypted_archive")
                    if item.file_size / max(item.compress_size, 1) > 200:
                        raise ScanFailure("archive_ratio_limit")
                pytorch_roots = set()
                checked_metadata = set()
                # A directory name alone cannot establish opaque tensor storage.
                # Only the bounded ML reconstruction profile can enable it; that
                # profile always retains a medium review finding, never a pass.
                if suffix in {".pt", ".pth", ".bin", ".ckpt"}:
                    for item in info:
                        if item.is_dir() or not item.filename.endswith("/data.pkl"):
                            continue
                        remaining(deadline)
                        if item.file_size > PICKLE_LIMIT:
                            raise ScanFailure("pickle_metadata_limit")
                        with archive.open(item) as member:
                            verdict = analyze(member.read(PICKLE_LIMIT + 1))
                        checked_metadata.add(item.filename)
                        if verdict.get("rule") == "ML_CONSTRUCTION_REVIEW":
                            pytorch_roots.add(item.filename[:-len("data.pkl")])
                for item in info:
                    remaining(deadline)
                    if item.is_dir():
                        continue
                    if item.filename in checked_metadata:
                        continue
                    storage = any(item.filename.startswith(root + "data/") and
                        item.filename[len(root + "data/"):].isascii() and
                        item.filename[len(root + "data/"):].isdecimal() for root in pytorch_roots)
                    if storage:
                        if "pytorch:raw-tensor-storage" not in result["checks"]:
                            result["checks"].append("pytorch:raw-tensor-storage")
                    with archive.open(item) as member:
                        # Prefix recognition also applies to renamed members;
                        # only candidate metadata is read into bounded memory.
                        prefix = member.read(65536)
                        member_suffix = pathlib.PurePosixPath(item.filename).suffix.lower()
                        if prefix.startswith(NESTED_ARCHIVE_MAGIC) or prefix.startswith(b"\x93NUMPY") or member_suffix in UNSUPPORTED_SERIALIZATION_SUFFIXES:
                            raise ScanFailure("unsupported_serialization")
                        candidate = member_suffix in PICKLE_SUFFIXES
                        # Raw floats can start with 0x80 by coincidence. Require
                        # recognizable opcodes here, but still inspect embedded
                        # Pickle calls instead of blindly skipping data/N bytes.
                        candidate = candidate or looks_like_pickle(io.BytesIO(prefix), binary_prefix=not storage)
                        if candidate:
                            if item.file_size > PICKLE_LIMIT:
                                raise ScanFailure("pickle_metadata_limit")
                            data = prefix + member.read(PICKLE_LIMIT + 1 - len(prefix))
                            analyze(data)
                if suffix in PICKLE_SUFFIXES | UNSUPPORTED_SERIALIZATION_SUFFIXES and not analyzed:
                    raise ScanFailure("unsupported_serialization")
        elif suffix in PICKLE_SUFFIXES or looks_like_pickle(content):
            result["file_type"] = "pickle"
            content.seek(0)
            analyze(content.read(PICKLE_LIMIT + 1))
        else:
            content.seek(0)
            prefix = content.read(16)
            if prefix.startswith(NESTED_ARCHIVE_MAGIC) or prefix.startswith(b"\x93NUMPY") or suffix in UNSUPPORTED_SERIALIZATION_SUFFIXES:
                raise ScanFailure("unsupported_serialization")
            result["file_type"] = {".json": "json-by-name", ".py": "script-by-name",
                                   ".so": "binary-by-name", ".dll": "binary-by-name",
                                   ".safetensors": "safetensors-by-name"}.get(suffix, "other")
        if not analyzed:
            result["checks"].append("pickle:not-applicable")
    except ScanFailure as exc:
        result.update(status="failed", error=str(exc))
    except (zipfile.BadZipFile, RuntimeError, NotImplementedError, EOFError):
        result.update(status="failed", error="unsupported_serialization")
    return result


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def respond(self, result, code):
        body = json.dumps(result).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Connection", "close")
        self.end_headers()
        self.close_connection = True
        try:
            self.wfile.write(body)
        except (OSError, TimeoutError):
            pass

    def do_GET(self):
        if self.path != "/identity":
            self.send_error(404)
            return
        try:
            self.respond({"ruleset": ruleset_identity()}, 200)
        except Exception:
            self.respond({"error": "scanner_unavailable"}, 503)

    def do_POST(self):
        parsed = urlsplit(self.path)
        if parsed.path != "/scan":
            self.send_error(404)
            return
        if not SCAN_SLOT.acquire(blocking=False):
            self.respond({"status": "failed", "error": "scanner_busy"}, 503)
            return
        result = {"status": "failed", "findings": [], "checks": []}
        try:
            self.connection.settimeout(10)
            deadline = time.monotonic() + TIMEOUT
            try:
                size = int(self.headers.get("Content-Length", "-1"))
            except ValueError as exc:
                raise ScanFailure("truncated_request") from exc
            if not 0 <= size <= LIMIT:
                raise ScanFailure("file_size_limit")
            path = parse_qs(parsed.query).get("path", [""])[0]
            ruleset = ruleset_identity()
            with tempfile.TemporaryFile(dir=SPOOL_DIR) as content:
                unread = size
                digest = hashlib.sha256()
                while unread:
                    remaining(deadline)
                    chunk = self.rfile.read(min(65536, unread))
                    if not chunk:
                        raise ScanFailure("truncated_request")
                    content.write(chunk)
                    digest.update(chunk)
                    unread -= len(chunk)
                result = scan(path, content, deadline)
                result.update(sha256=digest.hexdigest(), size=size)
            if ruleset_identity() != ruleset:
                raise ScanFailure("scanner_incomplete")
            result["ruleset"] = ruleset
        except ScanFailure as exc:
            result.update(status="failed", error=str(exc))
        except TimeoutError:
            result.update(status="failed", error="scanner_timeout")
        except Exception:
            result.update(status="failed", error="scanner_incomplete")
        finally:
            SCAN_SLOT.release()
        self.respond(result, 503 if result["status"] == "failed" else 200)


if __name__ == "__main__":
    ThreadingHTTPServer(("0.0.0.0", 8080), Handler).serve_forever()
