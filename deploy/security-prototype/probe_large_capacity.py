"""Streaming, synthetic weight-container capacity test; no model loading.
Copyright The MatrixHub Authors. Licensed under Apache-2.0.
"""
import hashlib
import io
import json
import os
import pathlib
import pickle
import subprocess
import threading
import time
import urllib.request
import zipfile

from huggingface_hub import HfApi
from probe import ENDPOINT, api_json

ROOT = pathlib.Path(os.environ.get("MH_PROBE_RESULT_ROOT", ".mh-local/upgrade-20261010"))
CONTAINERS = os.environ.get("MH_PROBE_STATS_CONTAINERS", "").split()


def fixture(path, target):
    metadata = pickle.dumps({"fixture": "static-capacity", "safe": [1, 2, 3]}, protocol=4)
    def start(output):
        output.writestr("model/data.pkl", metadata)
        return output.open("model/data/0", "w", force_zip64=True)
    measure = io.BytesIO()
    with zipfile.ZipFile(measure, "w", zipfile.ZIP_STORED) as output:
        with start(output):
            pass
    remaining = target - len(measure.getvalue())
    block = hashlib.shake_256(b"matrixhub synthetic capacity weights v1").digest(1 << 20)
    with zipfile.ZipFile(path, "w", zipfile.ZIP_STORED) as output:
        with start(output) as member:
            while remaining:
                chunk = block[:min(len(block), remaining)]
                member.write(chunk)
                remaining -= len(chunk)
    assert path.stat().st_size == target
    with path.open("rb") as source:
        return hashlib.file_digest(source, "sha256").hexdigest()


def main():
    ROOT.mkdir(parents=True, exist_ok=True)
    api_json("/api/v1alpha1/login", {"username": "admin", "password": "changeme"})
    namespace = "mhlarge" + str(int(time.time()))
    api_json("/api/v1alpha1/projects", {"name": namespace, "type": "PROJECT_TYPE_PUBLIC"})
    token = api_json("/api/v1alpha1/current-user/access-tokens", {"name": "local-large-capacity"})["token"]
    api = HfApi(endpoint=ENDPOINT, token=token)
    samples = []
    stopped = threading.Event()
    def observe():
        while not stopped.is_set():
            run = subprocess.run(["docker", "stats", "--no-stream", "--format", "{{json .}}", *CONTAINERS], capture_output=True, text=True, timeout=15)
            if run.returncode == 0:
                samples.append({"at": time.time(), "containers": [json.loads(line) for line in run.stdout.splitlines()]})
            stopped.wait(0.5)
    observer = threading.Thread(target=observe, daemon=True) if CONTAINERS else None
    if observer:
        observer.start()
    rows = []
    try:
        for mib in [256, 1024]:
            path = ROOT / f"capacity-{mib}.pt"
            digest = fixture(path, mib << 20)
            repo = f"{namespace}/tier{mib}"
            api.create_repo(repo)
            begin = time.monotonic()
            commit = api.upload_file(path_or_fileobj=str(path), path_in_repo="weights.pt", repo_id=repo)
            uploaded = time.monotonic()
            until = uploaded + 900
            while time.monotonic() < until:
                info = api.model_info(repo, revision=commit.oid, securityStatus=True, files_metadata=True)
                report = info.security_repo_status["details"]
                if report["status"] not in {"pending", "scanning", "unscanned"}:
                    break
                time.sleep(0.5)
            else:
                raise AssertionError("capacity scan timeout")
            assert report["status"] == "passed", report
            checked = next(f for f in report["files"] if f["path"] == "weights.pt")
            assert checked["sha256"] == digest and checked["size"] == mib << 20, checked
            assert "fickling/0.1.12" in checked["checks"], checked
            scanned = time.monotonic()
            request = urllib.request.Request(f"{ENDPOINT}/{repo}/resolve/{commit.oid}/weights.pt", headers={"Authorization": "Bearer " + token})
            downloaded = 0
            received_digest = hashlib.sha256()
            with urllib.request.urlopen(request, timeout=60) as response:
                while chunk := response.read(1 << 20):
                    downloaded += len(chunk)
                    received_digest.update(chunk)
            assert downloaded == mib << 20 and received_digest.hexdigest() == digest
            row = {"tier_mib": mib, "bytes": downloaded, "revision": commit.oid, "repo": repo,
                   "sha256": digest, "storage": "lfs" if next(f for f in info.siblings if f.rfilename == "weights.pt").lfs else "git",
                   "upload_seconds": round(uploaded - begin, 3), "scan_after_upload_seconds": round(scanned - uploaded, 3),
                   "download_seconds": round(time.monotonic() - scanned, 3), "status": report["status"], "checks": checked["checks"], "passed": True}
            rows.append(row)
            print(json.dumps(row), flush=True)
            (ROOT / "large-capacity-results.json").write_text(json.dumps({"scope": "synthetic ZIP_STORED containers with small Pickle metadata; single local queue", "cases": rows, "resource_samples": samples}, indent=2) + "\n")
            path.unlink()
    finally:
        stopped.set()
        if observer:
            observer.join(timeout=16)
        (ROOT / "large-capacity-results.json").write_text(json.dumps({"scope": "synthetic ZIP_STORED containers with small Pickle metadata; single local queue", "cases": rows, "resource_samples": samples}, indent=2) + "\n")


if __name__ == "__main__":
    main()
