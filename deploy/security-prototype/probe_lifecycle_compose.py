"""Explicit isolated Compose lifecycle probe. Set server/scanner container names.
Copyright The MatrixHub Authors. Licensed under Apache-2.0.
"""
import io, json, os, pathlib, subprocess, sys, time, urllib.request
sys.path.insert(0, str(pathlib.Path("deploy/security-prototype").resolve()))
from huggingface_hub import HfApi
from probe import api_json, wait_report, deny, ENDPOINT
root=pathlib.Path(os.environ["MH_PROBE_RESULT_ROOT"])
scanner=os.environ["MH_PROBE_SCANNER_CONTAINER"]
server=os.environ["MH_PROBE_SERVER_CONTAINER"]
api_json("/api/v1alpha1/login",{"username":"admin","password":"changeme"})
ns="mhstage3life"+str(int(time.time()))
api_json("/api/v1alpha1/projects",{"name":ns,"type":"PROJECT_TYPE_PUBLIC"})
token=api_json("/api/v1alpha1/current-user/access-tokens",{"name":"stage3-lifecycle"})["token"]
api=HfApi(endpoint=ENDPOINT,token=token)
repo=ns+"/lifecycle"; api.create_repo(repo)
rows=[]
def row(name,**detail):
    rows.append({"scenario":name,"passed":True,**detail}); print(json.dumps(rows[-1]),flush=True)
def upload(data):
    return api.upload_file(path_or_fileobj=io.BytesIO(data),path_in_repo="config.json",repo_id=repo).oid
subprocess.run(["docker","stop",scanner],check=True,stdout=subprocess.DEVNULL)
try:
    failed=upload(b'{"offline":true}')
    _, report=wait_report(api,repo,failed)
    assert report["status"]=="failed" and report["error"]=="scanner identity unavailable", report
    assert deny(token,repo,failed,"config.json")=="failed"
    row("real-scanner-offline-denies",revision=failed)
finally:
    subprocess.run(["docker","start",scanner],check=True,stdout=subprocess.DEVNULL)
# Container startup completion is not scanner readiness. Wait for the local
# identity endpoint before uploading the restored-scanner acceptance revision.
deadline=time.monotonic()+20
while True:
    try:
        subprocess.run(["docker","exec",scanner,"python","-c",
            "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8080/identity',timeout=5).read()"],
            check=True,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
        break
    except subprocess.CalledProcessError:
        if time.monotonic()>deadline: raise
        time.sleep(.25)
good=upload(b'{"restored":true}')
_, report=wait_report(api,repo,good); assert report["status"]=="passed",report
file=next(f for f in report["files"] if f["path"]=="config.json")
assert file["file_type"]=="json-by-name" and file["checked_at"] and file["recommended_action"]=="distribute_by_policy",file
row("restored-scanner-and-complete-file-report",revision=good,file_type=file["file_type"],checked_at=file["checked_at"],recommended_action=file["recommended_action"])
subprocess.run(["docker","restart",server],check=True,stdout=subprocess.DEVNULL)
deadline=time.monotonic()+20
while True:
    try:
        _, report=wait_report(api,repo,failed); break
    except Exception:
        if time.monotonic()>deadline: raise
        time.sleep(.3)
assert report["status"]=="failed",report
assert deny(token,repo,failed,"config.json")=="failed"
_,report=wait_report(api,repo,good); assert report["status"]=="passed",report
old=json.loads((root/"results.json").read_text())
bad=next(r for r in old["results"] if r["status"]=="blocked")
_,report=wait_report(api,old["repo"],bad["revision"]); assert report["status"]=="blocked",report
assert deny(token,old["repo"],bad["revision"],"weights.pkl")=="blocked"
row("real-server-restart-preserves-failed-passed-blocked")
(root/"lifecycle-results.json").write_text(json.dumps({"repo":repo,"cases":rows},indent=2),encoding="utf-8")
