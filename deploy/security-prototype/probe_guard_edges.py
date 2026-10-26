"""Reproduce layout spoofing and mixed risk/failure through local requests.

Only for an isolated test instance. Run probe_public_checkpoint.py first.
Dangerous bytes are never deserialized.
Copyright The MatrixHub Authors. Licensed under Apache-2.0.
"""
import argparse
import hashlib
import io
import json
import os
import pickle
import subprocess
import time
import urllib.error
import urllib.request
import zipfile

from huggingface_hub import HfApi
from probe import ENDPOINT, ROOT, api_json, deny, wait_report


def archive(members):
    buffer = io.BytesIO()
    with zipfile.ZipFile(buffer, "w", zipfile.ZIP_STORED) as output:
        for name, content in members:
            output.writestr(name, content)
    return buffer.getvalue()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--characterize", action="store_true", help="capture the pre-fix behavior without asserting success")
    options = parser.parse_args()
    container = os.environ.get("MH_PROBE_SCANNER_CONTAINER", "mh-security-local-scanner-1")
    networks = json.loads(subprocess.check_output(["docker", "inspect", container], text=True))[0]["NetworkSettings"]["Networks"]
    assert len(networks) == 1
    endpoint = "http://" + next(iter(networks.values()))["IPAddress"] + ":8080"
    safe = pickle.dumps({"safe": [1, 2, 3]}, protocol=4)
    danger = b"cos\nsystem\n(S'touch /tmp/MH_PROBE_MUST_NOT_EXECUTE'\ntR."
    checkpoint = (ROOT / "public-checkpoint.bin").read_bytes()
    assert hashlib.sha256(checkpoint).hexdigest() == "545d8feae7cdaa752dfcecd8d480928b31a0f7a0b494877c9ab5ddf504906703"
    with zipfile.ZipFile(io.BytesIO(checkpoint)) as model:
        ml = model.read(next(name for name in model.namelist() if name.endswith("/data.pkl")))
    rows = []
    fixtures = [
        ("spoof-layout.pt", archive([("model/data.pkl", safe), ("model/data/0", danger)])),
        ("spoof-layout.zip", archive([("model/data.pkl", safe), ("model/data/0", danger)])),
        ("spoof-before-metadata.pt", archive([("model/data/0", danger), ("model/data.pkl", safe)])),
        ("spoof-broken-metadata.pt", archive([("model/data.pkl", b"\x80\x04}"), ("model/data/0", danger)])),
        ("spoof-ml-layout.pt", archive([("model/data.pkl", ml), ("model/data/0", danger)])),
        ("spoof-ml-layout.zip", archive([("model/data.pkl", ml), ("model/data/0", danger)])),
        ("duplicate-metadata.pt", archive([("model/data.pkl", safe), ("model/data.pkl", ml), ("model/data/0", danger)])),
    ]
    for name, payload in fixtures:
        request = urllib.request.Request(endpoint + "/scan?path=" + name, data=payload,
            headers={"Content-Type": "application/octet-stream"})
        try:
            response = urllib.request.urlopen(request, timeout=20)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            result = json.load(response)
        accepted = result["status"] in {"blocked", "failed"}
        if not options.characterize:
            assert accepted, (name, result)
            if name == "spoof-ml-layout.pt":
                assert any(item["rule"] == "ML_CONSTRUCTION_REVIEW" for item in result["findings"]), result
        rows.append({"scenario": name, "status": result["status"], "not_clean": accepted,
            "rules": [item["rule"] for item in result.get("findings", [])]})

    api_json("/api/v1alpha1/login", {"username": "admin", "password": "changeme"})
    namespace = "mhguard" + str(int(time.time()))
    api_json("/api/v1alpha1/projects", {"name": namespace, "type": "PROJECT_TYPE_PUBLIC"})
    token = api_json("/api/v1alpha1/current-user/access-tokens", {"name": "local-guard-edges"})["token"]
    api = HfApi(endpoint=ENDPOINT, token=token)
    repo = namespace + "/mixed-failure"
    api.create_repo(repo)
    eicar = b"X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*"
    payload = archive([("av-test.txt", eicar), ("model/data.pkl", b"\x80\x04" + b"x" * (8 << 20))])
    revision = api.upload_file(path_or_fileobj=io.BytesIO(payload), path_in_repo="weights.pt", repo_id=repo).oid
    _, report = wait_report(api, repo, revision)
    assert report["status"] == "failed", report
    file = next(item for item in report["files"] if item["path"] == "weights.pt")
    assert any(item["severity"] == "high" for item in file["findings"]), file
    if not options.characterize:
        assert file["recommended_action"] == "quarantine_and_replace", file
    strict = {"block_severity": "medium", "on_pending": "block", "on_failure": "block"}

    def policy(value):
        request = urllib.request.Request(ENDPOINT + f"/api/security/v1alpha1/models/{repo}/policy",
            data=json.dumps(value).encode(), method="PUT",
            headers={"Content-Type": "application/json", "Authorization": "Bearer " + token})
        with urllib.request.urlopen(request, timeout=10) as response:
            assert json.load(response) == value

    try:
        policy({**strict, "block_severity": "high", "on_failure": "allow"})
        request = urllib.request.Request(ENDPOINT + f"/{repo}/resolve/{revision}/weights.pt",
            headers={"Authorization": "Bearer " + token})
        try:
            response = urllib.request.urlopen(request, timeout=15)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            code = response.status
            # Do not execute, load or save the downloaded container.
            if code == 403:
                decision = json.load(response)
                assert decision["scanStatus"] == "failed", decision
        if not options.characterize:
            assert code == 403, "known high-risk findings were allowed by failure policy"
        rows.append({"scenario": "high-finding-in-failed-file-with-failure-allow", "http": code,
            "passed": code == 403, "revision": revision})
    finally:
        policy(strict)
    assert deny(token, repo, revision, "weights.pt") == "failed"
    marker = subprocess.check_output(["docker", "exec", container, "python", "-c",
        "from pathlib import Path; print(Path('/tmp/MH_PROBE_MUST_NOT_EXECUTE').exists())"], text=True).strip()
    assert marker == "False"
    output = {"phase": "before" if options.characterize else "after", "repo": repo,
        "cases": rows, "payload_marker_exists": False}
    filename = "guard-edges-before.json" if options.characterize else "guard-edges-results.json"
    (ROOT / filename).write_text(json.dumps(output, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(output, indent=2), flush=True)


if __name__ == "__main__":
    main()
