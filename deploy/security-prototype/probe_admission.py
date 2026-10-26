"""Local real-request admission/control acceptance. Never loads model bytes.

Copyright The MatrixHub Authors. Licensed under Apache-2.0.
"""
import hashlib
import io
import json
import os
import pathlib
import pickle
import subprocess
import time
import urllib.error
import urllib.parse
import urllib.request
import zipfile

from huggingface_hub import HfApi
from huggingface_hub.errors import HfHubHTTPError
from probe import ENDPOINT, api_json, wait_report

ROOT = pathlib.Path(os.environ.get("MH_PROBE_RESULT_ROOT", ".mh-local/probe-stage2"))
SCANNER_CONTAINER = os.environ.get("MH_PROBE_SCANNER_CONTAINER", "mh-probe-scanner")

def main():
    ROOT.mkdir(parents=True, exist_ok=True)
    api_json("/api/v1alpha1/login", {"username":"admin","password":"changeme"})
    namespace = "mhgate" + str(int(time.time()))
    api_json("/api/v1alpha1/projects", {"name":namespace,"type":"PROJECT_TYPE_PUBLIC"})
    token = api_json("/api/v1alpha1/current-user/access-tokens", {"name":"local-admission-probe"})["token"]
    api = HfApi(endpoint=ENDPOINT,token=token)
    rows=[]

    def request(path, body=None, method="GET", auth=True, headers=None):
        merged = {"Content-Type":"application/json"}
        if auth: merged["Authorization"]="Bearer "+token
        if headers: merged.update(headers)
        req=urllib.request.Request(ENDPOINT+path, data=None if body is None else json.dumps(body).encode(), method=method,headers=merged)
        try: response=urllib.request.urlopen(req,timeout=25)
        except urllib.error.HTTPError as error: response=error
        with response: return response.status,response.read()

    def record(name, **detail):
        row={"scenario":name,"passed":True,**detail}; rows.append(row); print(json.dumps(row),flush=True)

    def security(repo,action,revision="",body=None,method="GET",auth=True):
        path=f"/api/security/v1alpha1/models/{repo}/{action}"
        if revision: path+="?"+urllib.parse.urlencode({"revision":revision})
        code,data=request(path,body,method,auth)
        return code,json.loads(data)

    def upload(repo,path,data,status="passed",repo_type=None):
        c=api.upload_file(path_or_fileobj=io.BytesIO(data),path_in_repo=path,repo_id=repo,repo_type=repo_type)
        if repo_type != "dataset":
            _,report=wait_report(api,repo,c.oid); assert report["status"]==status,report
        return c.oid

    def batch(repo,revision,data,repo_prefix=""):
        oid=hashlib.sha256(data).hexdigest()
        code,payload=request(f"/{repo_prefix}{repo}.git/info/lfs/objects/batch",
            {"operation":"download","ref":{"name":revision},"objects":[{"oid":oid,"size":len(data)}]},"POST",headers={"Accept":"application/vnd.git-lfs+json"})
        assert code==200,(code,payload)
        return json.loads(payload)["objects"][0]

    safe=namespace+"/safe"; api.create_repo(safe)
    data=pickle.dumps({"safe":[1,2,3]},protocol=4)
    rev=upload(safe,"weights.pkl",data)
    oid=hashlib.sha256(data).hexdigest()
    item=batch(safe,rev,data); link=item["actions"]["download"]
    with urllib.request.urlopen(urllib.request.Request(link["href"],headers=link["header"]),timeout=10) as response:
        assert response.read()==data
    record("lfs-batch-ref-and-scoped-download",revision=rev)
    code,_=request("/objects/"+oid); assert code==403,code
    record("raw-oid-download-denied",http=code)
    other=namespace+"/other"; api.create_repo(other); other_rev=upload(other,"config.json",b"{}")
    code,_=request("/objects/"+oid+"?"+urllib.parse.urlencode({"repo":other,"revision":other_rev})); assert code==403,code
    record("oid-cannot-use-another-repository-context",http=code)

    code,report=security(safe,"report",rev); assert code==200 and report["status"]=="passed",report
    first_attempt=report["attempt"]
    code,_=security(safe,"rescan",rev,{"force":False},"POST"); assert code==202,code
    _,report=wait_report(api,safe,rev)
    assert report["attempt"]==first_attempt+1 and next(f for f in report["files"] if f["path"]=="weights.pkl")["reused"],report
    record("explicit-reuse-rescan",attempt=report["attempt"])
    code,_=security(safe,"rescan",rev,{"force":True},"POST"); assert code==202,code
    _,report=wait_report(api,safe,rev)
    assert not any(f["reused"] for f in report["files"]),report
    record("forced-rescan-bypasses-cache",attempt=report["attempt"])

    # Weight-container shape: bounded Pickle metadata plus a 16 MiB opaque tensor
    # member. It is scanned statically, never imported by a ML framework.
    buffer=io.BytesIO()
    with zipfile.ZipFile(buffer,"w",zipfile.ZIP_STORED) as archive:
        archive.writestr("model/data.pkl",data)
        archive.writestr("model/data/0",os.urandom(16*1024*1024))
    large=buffer.getvalue(); started=time.monotonic()
    large_rev=upload(safe,"weights.pt",large)
    _,large_report=wait_report(api,safe,large_rev)
    f=next(f for f in large_report["files"] if f["path"]=="weights.pt")
    assert f["size"]==len(large) and f["sha256"]==hashlib.sha256(large).hexdigest(),f
    code,downloaded=request(f"/{safe}/resolve/{large_rev}/weights.pt"); assert code==200 and downloaded==large
    record("streamed-16-mib-weight-container-and-exact-download",bytes=len(large),seconds=round(time.monotonic()-started,3))

    env={**os.environ,"GIT_TERMINAL_PROMPT":"0","GIT_LFS_SKIP_SMUDGE":"1"}
    def git(*args):
        return subprocess.run(["git","-c","credential.helper=","-c","http.extraHeader=Authorization: Bearer "+token,*args],capture_output=True,text=True,env=env,timeout=25)
    git("ls-remote",ENDPOINT+"/"+safe+".git") # Queues every reachable revision, including the initial commit.
    for c in api.list_repo_commits(safe):
        _,r=wait_report(api,safe,c.commit_id); assert r["status"]=="passed",r
    clone=git("clone","--no-checkout",ENDPOINT+"/"+safe+".git",str(ROOT/namespace))
    assert clone.returncode==0,clone.stderr
    record("real-git-clone-admits-clean-history")
    danger=b"cos\nsystem\n(S'touch /tmp/MH_PROBE_MUST_NOT_EXECUTE'\ntR."
    bad=upload(safe,"weights.pkl",danger,"blocked")
    item=batch(safe,bad,danger); assert item["error"]["code"]==403,item
    record("lfs-blocked-revision-denied",http=item["error"]["code"])
    fixed=upload(safe,"weights.pkl",data)
    result=git("ls-remote",ENDPOINT+"/"+safe+".git"); assert result.returncode!=0,result.stderr
    code,_=request(f"/{safe}.git/info/refs?service=git-upload-pack"); assert code in (401,403),code
    record("git-denies-dangerous-history-even-after-clean-tip",old_revision=bad,new_revision=fixed)
    code,_=request(f"/{safe}.git/git-upload-pack",{},"POST"); assert code in (401,403),code
    record("direct-git-upload-pack-denied",http=code)

    strict={"block_severity":"medium","on_pending":"block","on_failure":"block"}
    lenient={"block_severity":"high","on_pending":"block","on_failure":"allow"}
    code,p=security(other,"policy",body=lenient,method="PUT"); assert code==200 and p==lenient,(code,p)
    code,p=security(safe,"policy"); assert code==200 and p==lenient,p
    item=batch(safe,bad,danger); assert item["error"]["code"]==403,item
    record("policy-is-project-scoped-high-risk-stays-blocked")
    code,_=security(safe,"policy",body={**strict,"oops":1},method="PUT"); assert code==400,code
    code,_=security(safe,"policy",body=strict,method="PUT"); assert code==200,code

    # Cancel while the scanner is paused; its eventual result must not replace cancellation.
    subprocess.run(["docker","pause",SCANNER_CONTAINER],check=True,stdout=subprocess.DEVNULL)
    try:
        code,pending=security(other,"rescan",other_rev,{"force":True},"POST"); assert code==202,code
        time.sleep(0.3)
        code,cancelled=security(other,"cancel",other_rev,{},"POST"); assert code==202 and cancelled["status"]=="cancelled",(code,cancelled)
    finally: subprocess.run(["docker","unpause",SCANNER_CONTAINER],check=True,stdout=subprocess.DEVNULL)
    time.sleep(0.5)
    code,cancelled=security(other,"report",other_rev); assert code==200 and cancelled["status"]=="cancelled",cancelled
    code,_=request(f"/{other}/resolve/{other_rev}/config.json"); assert code==403,code
    record("cancelled-task-remains-denied-after-worker-returns",attempt=cancelled["attempt"])
    code,_=security(other,"rescan",other_rev,{},"POST"); assert code==202,code
    _,r=wait_report(api,other,other_rev); assert r["status"]=="passed",r
    record("cancelled-task-can-be-rescanned",attempt=r["attempt"])

    dataset=namespace+"/dataset"; api.create_repo(dataset,repo_type="dataset")
    limitations=[]
    try:
        dsrev=upload(dataset,"sample.pkl",data,repo_type="dataset")
    except HfHubHTTPError as error:
        if error.response.status_code != 403 or "pre-receive hook denied" not in str(error): raise
        limitations.append("Upstream model-only pre-receive hook rejects dataset commits; dataset end-to-end case not verified.")
        print("NOT VERIFIED: baseline dataset upload is rejected by the existing pre-receive hook",flush=True)
    else:
        item=batch(dataset,dsrev,data,"datasets/"); link=item["actions"]["download"]
        with urllib.request.urlopen(urllib.request.Request(link["href"],headers=link["header"]),timeout=10) as response: assert response.read()==data
        record("dataset-lfs-download-remains-functional")

    private_ns=namespace+"private"; api_json("/api/v1alpha1/projects",{"name":private_ns,"type":"PROJECT_TYPE_PRIVATE"})
    private=private_ns+"/private"; api.create_repo(private,private=True); privrev=upload(private,"config.json",b"{}")
    for action in ("report","audit","policy"):
        code,_=security(private,action,privrev,auth=False); assert code in (401,403),code
    record("private-report-audit-policy-deny-anonymous")
    code,audit=security(safe,"audit"); assert code==200,audit
    actions={e["action"] for e in audit["events"]}
    assert {"rescan","admission:lfs-object","admission:git-snapshot","policy-updated"} <= actions,actions
    assert "MH_PROBE_MUST_NOT_EXECUTE" not in json.dumps(audit),"audit leaked serialized payload"
    record("audit-captures-control-and-admission-without-payload",events=len(audit["events"]))
    marker=subprocess.check_output(["docker","exec",SCANNER_CONTAINER,"python","-c","import pathlib; print(pathlib.Path('/tmp/MH_PROBE_MUST_NOT_EXECUTE').exists())"],text=True).strip()
    assert marker=="False",marker
    output={"client":"huggingface_hub/1.29.0 + git","repo":safe,"cases":rows,"payload_marker_exists":False,"limitations":limitations}
    (ROOT/"admission-results.json").write_text(json.dumps(output,indent=2),encoding="utf-8")
    print(f"PASS: {len(rows)} real admission/control scenarios",flush=True)

if __name__ == "__main__": main()
