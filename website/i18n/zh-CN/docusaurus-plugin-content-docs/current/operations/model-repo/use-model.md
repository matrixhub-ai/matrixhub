---
sidebar_position: 3
---

# 使用模型

MatrixHub 支持通过 Transformers、vLLM 和 SGLang 使用模型。进入模型详情页，点击右上角 **使用此模型**，选择使用方式后，复制弹窗中的命令或代码即可开始使用。

## 前置条件

- 已登录 MatrixHub，并拥有目标项目和模型的访问权限。
- 已准备运行模型所需的计算资源。
- 已安装所选框架及其依赖，或已准备可用的 Docker 环境。
- 已将示例中的 `<matrixhub-endpoint>`和`<matrixhub-token>` 替换为实际值。

## 使用 Transformers

Transformers 适合在 Python 程序中加载模型并进行推理。使用步骤如下：

1. 在模型详情页点击 **使用此模型**，选择 **Transformers**。
1. 复制页面生成的代码，并根据实际任务修改输入内容和推理参数。
1. 在运行代码前，配置 MatrixHub 服务端点和访问凭证。

    ```bash
    export HF_ENDPOINT="https://<matrixhub-endpoint>"
    export HF_TOKEN="<matrixhub-token>"
    ```

1. 在 Python 程序中使用模型。

    ```python
    from transformers import pipeline

    pipe = pipeline(
        "text-generation",
        model="<project-name>/<model-name>",
    )

    result = pipe("Hello, MatrixHub!")
    print(result)
    ```

> 页面生成的代码会根据模型类型提供相应的任务和参数。请使用页面中的代码作为实际运行示例。

## 使用 vLLM

vLLM 适合启动模型服务，并通过 OpenAI 兼容接口提供推理能力。

1. 在模型详情页点击 **使用此模型**，选择 **vLLM**。
1. 配置 MatrixHub 服务端点。

    ```bash
    export HF_ENDPOINT="https://<matrixhub-endpoint>"
    ```

1. 复制页面生成的启动命令并执行。命令中的模型标识通常为 `<project-name>/<model-name>`。

    ```bash
    vllm serve <project-name>/<model-name>
    ```

1. 服务启动后，使用页面提供的接口地址和请求示例发送推理请求。

## 使用 SGLang

SGLang 适合启动高性能模型服务。使用步骤如下：

1. 在模型详情页点击 **使用此模型**，选择 **SGLang**。
1. 配置 MatrixHub 服务端点。

    ```bash
    export HF_ENDPOINT="https://<matrixhub-endpoint>"
    ```

1. 复制页面生成的启动命令并执行。

    ```bash
    python -m sglang.launch_server \
      --model-path <project-name>/<model-name>
    ```

1. 服务启动后，使用页面提供的接口地址和请求示例发送推理请求。

## 通过 Docker 使用模型

如果使用 Docker 运行 vLLM 或 SGLang，请在容器中配置 MatrixHub 服务端点，并挂载模型缓存目录。具体镜像、启动参数和请求示例以 **使用此模型** 弹窗中生成的内容为准。

## 注意事项

- 模型标识必须使用实际的项目名和模型名，不能直接保留占位符。
- 私有项目或私有模型需要有效的访问凭证；权限不足时，模型无法下载或加载。
- 首次使用模型时，框架可能需要先下载模型文件，请根据模型大小预留磁盘空间和网络带宽。
- Transformers、vLLM 和 SGLang 对模型架构的支持范围不同。若启动失败，请先确认模型与所选框架兼容。
