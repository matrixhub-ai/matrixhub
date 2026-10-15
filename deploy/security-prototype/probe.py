"""Local API acceptance probe. Never deserializes uploaded samples.

Copyright The MatrixHub Authors. Licensed under Apache-2.0.
"""
import hashlib
import http.cookiejar
import io
import json
import os
import pathlib
import pickle
import time
import urllib.error
import urllib.request

from huggingface_hub import HfApi, hf_hub_download

ENDPOINT = os.environ.get("MH_PROBE_ENDPOINT", "http://127.0.0.1:13871")
ROOT = pathlib.Path(os.environ.get("MH_PROBE_RESULT_ROOT", ".mh-local/probe"))
ROOT.mkdir(parents=True, exist_ok=True)
opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))


def api_json(path, body):
    request = urllib.request.Request(ENDPOINT + path, data=json.dumps(body).encode(),
                                     headers={"Content-Type": "application/json"})
    with opener.open(request, timeout=10) as response:
        return json.load(response)


def wait_report(api, repo, revision):
    until = time.monotonic() + 75
    while time.monotonic() < until:
        info = api.model_info(repo, revision=revision, securityStatus=True, files_metadata=True)
        assert info.sha == revision, "metadata must bind to requested immutable revision"
        report = info.security_repo_status["details"]
        assert report["revision"] == revision
        if report["status"] not in {"pending", "scanning", "unscanned"}:
            return info, report
        time.sleep(0.2)
    raise AssertionError("scan did not finish within probe budget")


def deny(token, repo, revision, filename):
    # Use the actual resolve endpoint, bypassing client-side cache.
    request = urllib.request.Request(f"{ENDPOINT}/{repo}/resolve/{revision}/{filename}",
                                     headers={"Authorization": "Bearer " + token})
    try:
        with urllib.request.urlopen(request, timeout=10) as response:
            response.read()
    except urllib.error.HTTPError as error:
        assert error.code == 403, error.code
        result = json.load(error)
        assert result["error"] == "RevisionBlocked", result
        assert result["revision"] == revision, result
        return result["scanStatus"]
    raise AssertionError("unsafe revision was downloadable")


def main():
    api_json("/api/v1alpha1/login", {"username": "admin", "password": "changeme"})
    namespace = "mhprobe" + str(int(time.time()))
    api_json("/api/v1alpha1/projects", {"name": namespace, "type": "PROJECT_TYPE_PUBLIC"})
    token = api_json("/api/v1alpha1/current-user/access-tokens", {"name": "local-core-probe"})["token"]
    api = HfApi(endpoint=ENDPOINT, token=token)
    repo = namespace + "/acceptance"
    api.create_repo(repo)
    rows = []

    def upload(name, data, expected):
        started = time.monotonic()
        commit = api.upload_file(path_or_fileobj=io.BytesIO(data), path_in_repo=name, repo_id=repo)
        info, report = wait_report(api, repo, commit.oid)
        assert report["status"] == expected, report
        file_result = next(item for item in report["files"] if item["path"] == name)
        assert file_result["sha256"] == hashlib.sha256(data).hexdigest(), file_result
        assert any(item.rfilename == name and item.size == len(data) for item in info.siblings), info.siblings
        if expected == "passed":
            downloaded = hf_hub_download(repo, filename=name, revision=commit.oid, endpoint=ENDPOINT,
                                          token=token, force_download=True, local_dir=ROOT / commit.oid)
            assert pathlib.Path(downloaded).read_bytes() == data
        else:
            assert deny(token, repo, commit.oid, name) == expected
            try:
                hf_hub_download(repo, filename=name, revision=commit.oid, endpoint=ENDPOINT,
                                token=token, force_download=True, local_dir=ROOT / commit.oid)
            except Exception as error:
                cause = error
                while cause is not None and not hasattr(cause, "response"):
                    cause = cause.__cause__
                assert cause is not None and cause.response.status_code == 403, type(error).__name__
            else:
                raise AssertionError("official client downloaded unsafe content")
        row = {"scenario": name, "revision": commit.oid, "status": report["status"],
               "seconds": round(time.monotonic() - started, 3), "download": "allowed" if expected == "passed" else "403",
               "sha256": file_result["sha256"], "findings": file_result["findings"]}
        sibling = next(item for item in info.siblings if item.rfilename == name)
        row["storage"] = "lfs" if sibling.lfs is not None else "git"
        rows.append(row)
        print(json.dumps(row), flush=True)
        return commit.oid

    normal = upload("config.json", b'{"model_type":"probe","test":true}\n', "passed")
    upload("weights.pkl", pickle.dumps({"safe": [1, 2, 3]}, protocol=4), "passed")
    # An inert test string instructs os.system IF unpickled. The scanners only parse bytes.
    danger = b"cos\nsystem\n(S'touch /tmp/MH_PROBE_MUST_NOT_EXECUTE'\ntR."
    bad = upload("weights.pkl", danger, "blocked")
    assert deny(token, repo, bad, "config.json") == "blocked", "admission must cover whole revision"
    upload("weights.pkl", b"\x80\x04}", "failed")
    fixed = upload("weights.pkl", pickle.dumps({"safe": "fixed"}, protocol=4), "passed")
    _, old = wait_report(api, repo, bad)
    assert old["status"] == "blocked"
    assert deny(token, repo, bad, "weights.pkl") == "blocked"
    _, original = wait_report(api, repo, normal)
    assert original["status"] == "passed"
    rows.append({"scenario": "historical-revision-isolation", "old_revision": bad,
                 "new_revision": fixed, "status": "passed", "old_download": "403"})
    # EICAR is the standard antivirus test string, not malware.
    eicar = b"X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*"
    upload("av-test.txt", eicar, "blocked")
    output = {"client": "huggingface_hub/1.29.0", "repo": repo, "endpoint": ENDPOINT, "results": rows}
    (ROOT / "results.json").write_text(json.dumps(output, indent=2), encoding="utf-8")
    print("PASS: official client upload, security metadata, downloads and revision isolation", flush=True)


if __name__ == "__main__":
    main()
