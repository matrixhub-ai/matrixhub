"""Live bounded-scanner acceptance cases; no unpickling or model execution.

Copyright The MatrixHub Authors. Licensed under Apache-2.0.
"""
import io
import json
import os
import pathlib
import pickle
import subprocess
import struct
import urllib.error
import urllib.request
import zipfile

container = os.environ.get("MH_PROBE_SCANNER_CONTAINER", "mh-probe-scanner")
endpoint = os.environ.get("MH_SCANNER_ENDPOINT")
if not endpoint:
    # The isolated network intentionally has no externally published port.
    network = json.loads(subprocess.check_output(["docker", "inspect", container], text=True))[0]["NetworkSettings"]["Networks"]
    assert len(network) == 1, "scanner must remain on exactly one isolated network"
    address = next(iter(network.values()))["IPAddress"]
    endpoint = "http://" + address + ":8080"


def archive(members):
    buffer = io.BytesIO()
    with zipfile.ZipFile(buffer, "w", zipfile.ZIP_DEFLATED) as output:
        for name, data in members:
            output.writestr(name, data)
    return buffer.getvalue()


safe = pickle.dumps({"safe": [1, 2, 3]}, protocol=4)
danger = b"cos\nsystem\n(S'touch /tmp/MH_PROBE_MUST_NOT_EXECUTE'\ntR."
cases = [
    ("plain.json", b"{}", "passed"),
    ("safe.pkl", safe, "passed"),
    ("danger.pkl", danger, "blocked"),
    ("broken.pkl", b"\x80\x04}", "failed"),
    ("trailing.pkl", safe + danger, "failed"),
    ("safe.pt", archive([("model/data.pkl", safe)]), "passed"),
    ("danger.pt", archive([("model/data.pkl", danger)]), "blocked"),
    ("opaque.pt", archive([("model/readme.txt", b"opaque")]), "failed"),
    ("ratio.zip", archive([("huge.txt", b"0" * (1024 * 1024))]), "failed"),
    ("members.zip", archive([(str(i), b"x") for i in range(101)]), "failed"),
]
cases.extend((f"protocol-{protocol}.pkl", pickle.dumps({"safe": [1, 2, 3]}, protocol=protocol), "passed") for protocol in range(6))
cases.extend([
    ("renamed-danger.dat", danger, "blocked"),
    ("renamed-binary.dat", pickle.dumps({"safe": [1, 2, 3]}, protocol=4), "passed"),
    ("renamed-trailing.dat", pickle.dumps({"safe": [1, 2, 3]}, protocol=0) + danger, "failed"),
    ("renamed-zip.zip", archive([("model/metadata.dat", danger)]), "blocked"),
    ("pickle-too-large.pkl", b"\x80\x04" + b"x" * (8 * 1024 * 1024), "failed"),
])
stored = io.BytesIO()
with zipfile.ZipFile(stored, "w", zipfile.ZIP_STORED) as output:
    output.writestr("model/data.pkl", b"\x80\x04" + b"x" * (8 * 1024 * 1024))
cases.append(("metadata-too-large.pt", stored.getvalue(), "failed"))
# Inert encryption flag fixture; no password or decryption is performed.
encrypted = bytearray(archive([("secret.txt", b"secret")]))
for marker, offset in [(b"PK\x03\x04", 6), (b"PK\x01\x02", 8)]:
    index = encrypted.index(marker)
    flags = struct.unpack_from("<H", encrypted, index + offset)[0]
    struct.pack_into("<H", encrypted, index + offset, flags | 1)
cases.append(("encrypted.zip", bytes(encrypted), "failed"))
eicar = b"X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*"
mixed = io.BytesIO()
with zipfile.ZipFile(mixed, "w", zipfile.ZIP_STORED) as output:
    output.writestr("av-test.txt", eicar)
    output.writestr("model/data.pkl", b"\x80\x04" + b"x" * (8 * 1024 * 1024))
cases.append(("matched-then-incomplete.pt", mixed.getvalue(), "failed"))
cases.extend([
    ("nested.zip", archive([("nested.zip", archive([("data.pkl", danger)]))]), "failed"),
    ("renamed-numpy.dat", b"\x93NUMPY\x01\x00" + danger, "failed"),
    ("danger.joblib", danger, "blocked"),
    ("torch-load.pkl", b"ctorch\nload\n(S'https://example.invalid/not-executed'\ntR.", "blocked"),
    ("nested-load.pkl", b"ctorch.storage\n_load_from_bytes\n(S'not-executed'\ntR.", "blocked"),
    ("ml-mixed-danger.pkl", b"ctorch._utils\n_rebuild_tensor_v2\n0" + danger, "blocked"),
])
reason_codes = {"opaque.pt": "unsupported_serialization", "ratio.zip": "archive_ratio_limit",
                "members.zip": "archive_member_limit", "pickle-too-large.pkl": "pickle_metadata_limit",
                "metadata-too-large.pt": "pickle_metadata_limit", "encrypted.zip": "encrypted_archive",
                "matched-then-incomplete.pt": "pickle_metadata_limit", "nested.zip": "unsupported_serialization",
                "renamed-numpy.dat": "unsupported_serialization"}
results = []
for path, data, expected in cases:
    request = urllib.request.Request(endpoint + "/scan?path=" + path, data=data,
                                     headers={"Content-Type": "application/octet-stream"})
    try:
        response = urllib.request.urlopen(request, timeout=20)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        verdict = json.load(response)
    assert verdict["status"] == expected, (path, verdict)
    if path in reason_codes:
        assert verdict.get("error") == reason_codes[path], (path, verdict)
    if path == "matched-then-incomplete.pt":
        assert any(f["scanner"] == "clamav" for f in verdict["findings"]), verdict
    encoded = json.dumps(verdict)
    assert "MH_PROBE_MUST_NOT_EXECUTE" not in encoded, "payload text leaked into report"
    row = {"case": path, "status": verdict["status"], "expected": expected, "passed": True, "file_type": verdict.get("file_type"), "error": verdict.get("error")}
    results.append(row)
    print(json.dumps(row), flush=True)
marker = subprocess.check_output(["docker", "exec", container, "python", "-c",
                                  "import pathlib; print(pathlib.Path('/tmp/MH_PROBE_MUST_NOT_EXECUTE').exists())"], text=True).strip()
assert marker == "False", "dangerous test bytes executed"
root = pathlib.Path(os.environ.get("MH_PROBE_RESULT_ROOT", ".mh-local/probe"))
root.mkdir(parents=True, exist_ok=True)
(root / "scanner-results.json").write_text(json.dumps({"cases": results, "payload_marker_exists": False}, indent=2), encoding="utf-8")
print(f"PASS: {len(cases)} scanner cases; execution marker absent", flush=True)
