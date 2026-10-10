"""Recheck retained capacity artifacts under the final scanner rules.
Copyright The MatrixHub Authors. Licensed under Apache-2.0.
"""
import hashlib
import json
import pathlib
import time
import urllib.request
from huggingface_hub import HfApi
from probe import ENDPOINT, ROOT

api = HfApi(endpoint=ENDPOINT)
baseline = json.loads((ROOT / "large-capacity-results.json").read_text())
rows = []
for previous in baseline["cases"]:
    begin = time.monotonic()
    until = begin + 900
    while time.monotonic() < until:
        info = api.model_info(previous["repo"], revision=previous["revision"], securityStatus=True)
        report = info.security_repo_status["details"]
        if report["status"] not in {"pending", "scanning", "unscanned"}:
            break
        time.sleep(0.5)
    else:
        raise AssertionError("final rules recheck timeout")
    assert report["status"] == "passed", report
    file = next(f for f in report["files"] if f["path"] == "weights.pt")
    assert file["size"] == previous["bytes"] and file["sha256"] == previous["sha256"]
    digest = hashlib.sha256()
    size = 0
    with urllib.request.urlopen(f"{ENDPOINT}/{previous['repo']}/resolve/{previous['revision']}/weights.pt", timeout=60) as response:
        while chunk := response.read(1 << 20):
            size += len(chunk)
            digest.update(chunk)
    assert size == previous["bytes"] and digest.hexdigest() == previous["sha256"]
    row = {"tier_mib": previous["tier_mib"], "passed": True, "status": report["status"], "ruleset": report["ruleset"], "attempt": report["attempt"], "checks": file["checks"], "seconds": round(time.monotonic() - begin, 3)}
    rows.append(row)
    print(json.dumps(row), flush=True)
    (ROOT / "large-final-rules-results.json").write_text(json.dumps({"cases": rows}, indent=2) + "\n")
