"""Real-request web-session acceptance; no session secrets are saved or printed.
Copyright The MatrixHub Authors. Licensed under Apache-2.0.
"""
import json
import os
import pathlib
import urllib.error
import urllib.parse
import urllib.request

from probe import ENDPOINT, api_json, opener

def main():
    api_json('/api/v1alpha1/login', {'username':'admin','password':'changeme'})
    root=pathlib.Path(os.environ.get('MH_PROBE_RESULT_ROOT', '.mh-local/probe-stage2'))
    repo=json.loads((root/'admission-results.json').read_text())['repo']
    base='/api/security/v1alpha1/models/'+repo
    rows=[]
    def call(action, method='GET', body=None, origin=None, credential=None):
        headers={'Content-Type':'application/json'}
        if origin is not None: headers['Origin']=origin
        if credential: headers['Authorization']=credential
        req=urllib.request.Request(ENDPOINT+base+'/'+action, data=None if body is None else json.dumps(body).encode(), headers=headers, method=method)
        try: response=opener.open(req,timeout=15)
        except urllib.error.HTTPError as error: response=error
        with response: return response.status,response.read()
    def check(name, code, expected):
        assert code==expected,(name,code,expected)
        rows.append({'scenario':name,'http':code,'passed':True})
    code,data=call('policy'); check('cookie-authenticated-policy-read',code,200)
    policy=json.loads(data)
    code,_=call('policy','PUT',policy,ENDPOINT);check('same-origin-cookie-policy-write',code,200)
    code,_=call('policy','PUT',policy);check('cookie-write-without-origin-denied',code,403)
    code,_=call('policy','PUT',policy,'http://evil.invalid');check('cross-origin-cookie-write-denied',code,403)
    code,_=call('policy','PUT',policy,ENDPOINT,'Bearer forged');check('explicit-invalid-credential-cannot-fall-back-to-cookie',code,403)
    (root/'web-session-results.json').write_text(json.dumps({'cases':rows},indent=2)+'\n')
    print(json.dumps(rows,indent=2))

if __name__=='__main__': main()
