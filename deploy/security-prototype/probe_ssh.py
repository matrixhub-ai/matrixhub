"""Real SSH Git admission checks on a disposable local instance.
Copyright The MatrixHub Authors. Licensed under Apache-2.0.
"""
import io
import json
import os
import pathlib
import shlex
import subprocess
import time

from huggingface_hub import HfApi
from probe import ENDPOINT, api_json, wait_report

def main():
    root=pathlib.Path(os.environ.get('MH_PROBE_RESULT_ROOT','.mh-local/probe-stage2/compose'))
    root.mkdir(parents=True,exist_ok=True)
    keydir=root/('ssh-private-'+str(int(time.time())))
    keydir.mkdir(mode=0o700)
    key=keydir/'key'
    subprocess.run(['ssh-keygen','-q','-t','ed25519','-N','','-f',str(key)],check=True)
    api_json('/api/v1alpha1/login',{'username':'admin','password':'changeme'})
    api_json('/api/v1alpha1/current-user/ssh-keys',{'name':'local-ssh-probe','publicKey':key.with_suffix('.pub').read_text()})
    token=api_json('/api/v1alpha1/current-user/access-tokens',{'name':'local-ssh-probe'})['token']
    api=HfApi(endpoint=ENDPOINT,token=token)
    namespace='mhssh'+str(int(time.time()))
    api_json('/api/v1alpha1/projects',{'name':namespace,'type':'PROJECT_TYPE_PUBLIC'})
    safe=namespace+'/safe';api.create_repo(safe)
    commit=api.upload_file(repo_id=safe,path_in_repo='config.json',path_or_fileobj=io.BytesIO(b'{}'))
    _,report=wait_report(api,safe,commit.oid);assert report['status']=='passed'
    blocked=json.loads((root/'admission-results.json').read_text())['repo']
    port=os.environ.get('MH_PROBE_SSH_PORT','13873')
    env=dict(os.environ,GIT_SSH_COMMAND='ssh -o BatchMode=yes -o StrictHostKeyChecking=accept-new -o UserKnownHostsFile='+shlex.quote(str(keydir/'known_hosts'))+' -i '+shlex.quote(str(key))+' -p '+port)
    rows=[]
    def git(repo,allowed):
        # The first Git request can enqueue an unscanned empty initialization
        # commit. Wait for that history scan; do not weaken admission policy.
        deadline=time.monotonic()+20
        while True:
            run=subprocess.run(['git','ls-remote','ssh://git@127.0.0.1:'+port+'/'+repo+'.git'],env=env,capture_output=True,text=True,timeout=25)
            if not allowed or run.returncode == 0 or time.monotonic() >= deadline: break
            time.sleep(.25)
        assert (run.returncode==0)==allowed,(run.returncode,run.stderr)
        if not allowed: assert 'repository snapshot is not admitted' in run.stderr.lower(),run.stderr
        rows.append({'scenario':'ssh-clean-history-allowed' if allowed else 'ssh-dangerous-history-denied','exit_code':run.returncode,'passed':True})
    git(safe,True);git(blocked,False)
    (root/'ssh-results.json').write_text(json.dumps({'cases':rows},indent=2)+'\n')
    print(json.dumps(rows,indent=2))

if __name__=='__main__':main()
