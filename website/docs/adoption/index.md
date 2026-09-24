---
sidebar_position: 7
---

# Adoption

MatrixHub is an open-source, self-hosted, Hugging Face-compatible model registry for enterprise AI inference on Kubernetes.

## Adopters

This list contains organizations that have agreed to be named publicly. If you would like your organization added or removed, open a pull request that updates this page.

**[DaoCloud](https://daocloud.io/)** runs MatrixHub on its Token Factory platform as a private model hub on Kubernetes, with Hugging Face-compatible access, intranet model caching, project-scoped repositories, and operational workflows.

## External integration documentation

[vLLM's MatrixHub documentation](https://docs.vllm.ai/en/latest/models/supported_models/#matrixhub) explains how to point `HF_ENDPOINT` at a MatrixHub instance so that `vllm serve` retrieves model weights over the internal network instead of the public Hugging Face Hub.

[NVIDIA Dynamo's MatrixHub model-loading guide](https://docs.nvidia.com/dynamo/dev/kubernetes/model-deployment/model-loading/matrix-hub) shows how to configure a Kubernetes deployment so that Dynamo frontend and worker components use MatrixHub to load pre-cached model weights.

[llm-d's self-hosted registry guide](https://llm-d.ai/docs/dev/operations/model-loading-and-startup#self-hosted-registry-with-matrixhub) explains how to configure `HF_ENDPOINT` in `modelserver.env` so that llm-d model-server Pods load pre-cached model weights from MatrixHub.

## MatrixHub references

- [Target users and use cases](../overview/)
- [Integration guides](../integrations/)
- [Technical design documents](https://github.com/matrixhub-ai/matrixhub/blob/main/docs/design/README.md)

## Add your organization

To add your organization, open a pull request that updates this page; contributions are welcome.
