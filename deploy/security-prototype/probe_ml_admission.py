"""Exercise the public-checkpoint review warning through real HF admission.

Run probe_public_checkpoint.py first in an isolated local instance. No model
loading, unpickling or inference. Tokens remain process-local.
Copyright The MatrixHub Authors. Licensed under Apache-2.0.
"""
import hashlib
import io
import json
import time
import urllib.request

from huggingface_hub import HfApi
from probe import ENDPOINT, ROOT, api_json, deny, wait_report, opener


def main():
    data = (ROOT / "public-checkpoint.bin").read_bytes()
    assert hashlib.sha256(data).hexdigest() == "545d8feae7cdaa752dfcecd8d480928b31a0f7a0b494877c9ab5ddf504906703"
    api_json("/api/v1alpha1/login", {"username": "admin", "password": "changeme"})
    namespace = "mhml" + str(int(time.time()))
    api_json("/api/v1alpha1/projects", {"name": namespace, "type": "PROJECT_TYPE_PUBLIC"})
    token = api_json("/api/v1alpha1/current-user/access-tokens", {"name": "local-ml-admission"})["token"]
    api = HfApi(endpoint=ENDPOINT, token=token)
    repo = namespace + "/checkpoint-review"
    api.create_repo(repo)
    revision = api.upload_file(path_or_fileobj=io.BytesIO(data), path_in_repo="pytorch_model.bin", repo_id=repo).oid
    _, report = wait_report(api, repo, revision)
    assert report["status"] == "warning", report
    file = next(item for item in report["files"] if item["path"] == "pytorch_model.bin")
    assert any(f["rule"] == "ML_CONSTRUCTION_REVIEW" and f["severity"] == "medium" for f in file["findings"]), file
    assert deny(token, repo, revision, "pytorch_model.bin") == "warning"
    rows = [{"scenario": "public-checkpoint-warning-strict-policy-denies", "passed": True, "revision": revision}]
    strict = {"block_severity": "medium", "on_pending": "block", "on_failure": "block"}

    def policy(body):
        request = urllib.request.Request(ENDPOINT + f"/api/security/v1alpha1/models/{repo}/policy",
            data=json.dumps(body).encode(), method="PUT",
            headers={"Content-Type": "application/json", "Authorization": "Bearer " + token})
        with opener.open(request, timeout=10) as response:
            assert json.load(response) == body

    try:
        policy({**strict, "block_severity": "high"})
        with urllib.request.urlopen(ENDPOINT + f"/{repo}/resolve/{revision}/pytorch_model.bin", timeout=10) as response:
            downloaded = response.read(len(data) + 1)
        assert downloaded == data
        rows.append({"scenario": "review-warning-high-threshold-allows-exact-bytes", "passed": True})
        dangerous = b"cos\nsystem\n(S'touch /tmp/MH_PROBE_MUST_NOT_EXECUTE'\ntR."
        bad = api.upload_file(path_or_fileobj=io.BytesIO(dangerous), path_in_repo="extra.pkl", repo_id=repo).oid
        _, report = wait_report(api, repo, bad)
        assert report["status"] == "blocked", report
        assert deny(token, repo, bad, "pytorch_model.bin") == "blocked"
        rows.append({"scenario": "extra-dangerous-pickle-blocks-whole-revision-even-with-high-threshold", "passed": True, "revision": bad})
    finally:
        policy(strict)
    assert deny(token, repo, revision, "pytorch_model.bin") == "warning"
    rows.append({"scenario": "strict-policy-restored-denies-warning-again", "passed": True})
    output = {"repo": repo, "source_sha256": hashlib.sha256(data).hexdigest(), "cases": rows}
    (ROOT / "ml-admission-results.json").write_text(json.dumps(output, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(output, indent=2), flush=True)


if __name__ == "__main__":
    main()
