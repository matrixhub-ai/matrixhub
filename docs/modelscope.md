# ModelScope model downloads

MatrixHub supports ModelScope as an upstream model registry and provides
ModelScope-compatible model download endpoints. The existing Hugging Face and
Git interfaces continue to access the same repositories and LFS objects.

## Configure an upstream

In **Registry Management → Create Registry**, choose **ModelScope**, use
`https://www.modelscope.cn` as the base URL, and leave certificate verification
enabled. Public models need no credentials. For private upstream models, enter
the ModelScope username and access token. Testing a connection with a token
validates it with ModelScope's login API. Pulls exchange that SDK token for
ModelScope's separate Git token; file downloads continue to use the SDK token.

Select this registry in a pull synchronization rule or when creating a proxy
project. An upstream namespace such as `sshleifer` maps to the project's local
namespace. Model discovery supports paginated `namespace/*` rules. This
integration covers **models**; ModelScope dataset discovery and download APIs
are not implemented.

The pull fetches Git branches and tags, downloads their LFS files through
ModelScope's file API, and validates file sizes and SHA256 hashes before
reporting success. Failed or truncated transfers are not installed in the LFS
content store. Credentials are passed to Git through per-process configuration,
not persisted in repository configuration or included in command arguments.

## Download from MatrixHub

Tested with `modelscope==1.40.2` and `modelscope-hub==0.6.1`:

```bash
pip install 'modelscope==1.40.2' 'modelscope-hub==0.6.1'
export MODELSCOPE_ENDPOINT='http://127.0.0.1:3001'
ms-hub download team/model --revision main --local-dir ./model
```

The current `modelscope` CLI also exposes the same download command. For a
private project, supply a **MatrixHub** access token, not the upstream ModelScope
token:

```bash
export MODELSCOPE_API_TOKEN='<your-matrixhub-token>'
ms-hub download team/private-model --local-dir ./private-model
```

Python snapshot and single-file downloads:

```python
import os
from modelscope import snapshot_download
from modelscope.hub.file_download import model_file_download

endpoint = 'http://127.0.0.1:3001'
token = os.getenv('MODELSCOPE_API_TOKEN')

directory = snapshot_download(
    'team/model', endpoint=endpoint, token=token, revision='main',
)
config = model_file_download(
    'team/model', 'config.json', endpoint=endpoint, token=token,
    revision='main',
)
```

Use the local project/model path, which can differ from the upstream namespace.
Branches, tags and commit hashes are supported. When `master` does not exist,
the ModelScope download interface treats it as an alias for the repository's
default branch, allowing the SDK's default revision to work with `main`.
Pin a commit hash when downloads must remain stable across upstream updates.

The interface supports recursive file listings, SHA256 metadata, normal Git
files, LFS files, HEAD and HTTP Range requests. Existing project permissions
apply equally to HF and ModelScope requests; bearer tokens and ModelScope's
`m_session_id` access-token cookie are supported. ModelScope upload, repository
management and login APIs on MatrixHub are outside this download integration.

## Validation

The Go tests cover protocol responses, permissions, file ranges, invalid paths,
private Git authentication, discovery pagination, token validation, and rejection
of corrupt LFS content. Real SDK/CLI and real upstream checks are opt-in:

```bash
MATRIXHUB_MODELSCOPE_PYTHON=/path/to/venv/bin/python \
  go test -count=1 -run ModelScope ./internal/apiserver/handler/hf

MATRIXHUB_MODELSCOPE_MODEL=sshleifer/tiny-gpt2 \
  go test -count=1 -run TestModelScope ./internal/repo
```

The second command downloads a public upstream model into temporary test
storage. Neither test modifies an existing deployment.
