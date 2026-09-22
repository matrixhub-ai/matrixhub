---
sidebar_position: 3
---

# Using Models

MatrixHub supports using models with Transformers, vLLM, and SGLang. On the model details page, click **Use This Model**, select a method, and copy the generated command or code.

## Prerequisites

- A MatrixHub account with access to the target project and model.
- Sufficient compute resources for the model.
- The selected framework and its dependencies, or a working Docker environment.
- Replace `<matrixhub-endpoint>`, `<matrixhub-token>`, `<project-name>`, and `<model-name>` with actual values.

For private projects or models, configure the service endpoint and access token before running the generated command or code:

```bash
export HF_ENDPOINT="https://<matrixhub-endpoint>"
export HF_TOKEN="<matrixhub-token>"
```

## Use Transformers

Use Transformers to load a model and run inference in a Python program.

1. On the model details page, click **Use This Model** and select **Transformers**.
1. Copy the generated code and adjust the input and inference parameters as needed.
1. Run the Python program.

    ```python
    from transformers import pipeline

    pipe = pipeline(
        "text-generation",
        model="<project-name>/<model-name>",
    )

    result = pipe("Hello, MatrixHub!")
    print(result)
    ```

> The generated code varies by model type and task. Use the code shown in the dialog as the source of truth.

## Use vLLM

Use vLLM to serve a model through an OpenAI-compatible API.

1. On the model details page, click **Use This Model** and select **vLLM**.
1. Configure `HF_ENDPOINT` and, for private models, `HF_TOKEN` as described above.
1. Copy and run the generated launch command. The model identifier is typically `<project-name>/<model-name>`.

    ```bash
    vllm serve <project-name>/<model-name>
    ```

1. When the service starts, send inference requests to the endpoint using the request format shown in the dialog.

## Use SGLang

Use SGLang to serve a model for high-performance inference.

1. On the model details page, click **Use This Model** and select **SGLang**.
1. Configure `HF_ENDPOINT` and, for private models, `HF_TOKEN` as described above.
1. Copy and run the generated launch command.

    ```bash
    python -m sglang.launch_server \
      --model-path <project-name>/<model-name>
    ```

1. When the service starts, send inference requests to the endpoint using the request format shown in the dialog.

## Use Models with Docker

When running vLLM or SGLang with Docker, configure the MatrixHub endpoint and token inside the container, and mount the model cache directory. Use the image, launch parameters, and request examples generated in the **Use This Model** dialog.

## Notes

- Use the actual project and model names; do not leave placeholders in the command.
- Private projects and models require a valid access token. Without sufficient permission, the framework cannot download or load the model.
- The first use may download the model files. Reserve sufficient disk space and bandwidth.
- Transformers, vLLM, and SGLang support different model architectures. If startup fails, verify that the model is compatible with the selected framework.
