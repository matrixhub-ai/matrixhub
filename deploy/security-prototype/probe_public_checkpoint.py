"""Characterize a small public PyTorch checkpoint without loading it.
Copyright The MatrixHub Authors. Licensed under Apache-2.0.
"""
import hashlib
import json
import os
import pathlib
import subprocess
import urllib.error
import urllib.request
from probe import ROOT

source = "hf-internal-testing/tiny-random-BertModel"
revision = "fc08ad9cc33be9aef4f55cc80e16ef5ae3d5981c"
url = f"https://huggingface.co/{source}/resolve/{revision}/pytorch_model.bin"
with urllib.request.urlopen(url, timeout=30) as response:
    data = response.read((1 << 20) + 1)
assert len(data) <= 1 << 20, "public fixture exceeds this probe's download budget"
assert hashlib.sha256(data).hexdigest() == "545d8feae7cdaa752dfcecd8d480928b31a0f7a0b494877c9ab5ddf504906703", "public fixture digest changed"
(ROOT / "public-checkpoint.bin").write_bytes(data)
container = os.environ.get("MH_PROBE_SCANNER_CONTAINER", "mh-security-local-scanner-1")
network = json.loads(subprocess.check_output(["docker", "inspect", container], text=True))[0]["NetworkSettings"]["Networks"]
assert len(network) == 1
endpoint = "http://" + next(iter(network.values()))["IPAddress"] + ":8080"
request = urllib.request.Request(endpoint + "/scan?path=pytorch_model.bin", data=data, headers={"Content-Type": "application/octet-stream"})
try:
    response = urllib.request.urlopen(request, timeout=30)
except urllib.error.HTTPError as error:
    response = error
with response:
    result = json.load(response)
assert result["status"] == "warning" and any(f["rule"] == "ML_CONSTRUCTION_REVIEW" for f in result["findings"]), result
output = {"scope": "one public Hugging Face test checkpoint; characterization, no model loading or inference", "source": source,
          "source_url": url, "revision": revision, "sha256": hashlib.sha256(data).hexdigest(), "bytes": len(data), "result": result}
(ROOT / "public-checkpoint-results.json").write_text(json.dumps(output, indent=2) + "\n")
print(json.dumps(output, indent=2), flush=True)
