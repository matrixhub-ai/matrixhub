# 第三方组件与复现版本

2026-10-10；新增适配源码保留 MatrixHub 的 Apache-2.0 许可证。源码包不包含第三方扫描器实现、镜像、私钥、数据库或构建依赖缓存。

| 组件 | 验证/锁定版本 | 许可与来源 |
| --- | --- | --- |
| MatrixHub 基线 | 6890726abe272767d26ce1a6b0580b0718f232c0 | Apache-2.0；根目录 LICENSE；https://github.com/matrixhub-ai/matrixhub |
| Fickling | 0.1.12，Dockerfile 固定 pip 版本 | LGPL-3.0-or-later；已安装包 classifier 与许可文本；https://github.com/trailofbits/fickling |
| ClamAV | 1.5.4，当前扫描协议记录签名 28148 | GPL-2.0；外部服务；https://github.com/Cisco-Talos/clamav |
| ClamAV 容器 | sha256:ebec5bc138401b36ae987caa1a3fa3c3b2a21ed3d51f0bfa5852825e663e67b0 | compose.yaml 固定镜像摘要；保留官方镜像内许可证与组件说明 |
| Go 工具链 | 1.26.9；golang.org/x/net 0.60.0 | https://go.dev/LICENSE；依赖见 go.mod/go.sum |
| Node / pnpm | Node 24 / pnpm 10.17.0 | 构建工具；前端依赖见 ui/package.json、ui/pnpm-lock.yaml |
| 官方客户端 | huggingface_hub 1.29.0 | Apache-2.0；验证依赖；https://github.com/huggingface/huggingface_hub |

扫描器隔离为外部进程/服务，新增文件没有复制其实现。若分发构建镜像，随镜像保留外部组件的完整许可与相应源码获取方式；本表不是整个上游依赖树的许可证审计，锁文件及组件原许可继续适用。

规则身份含实际 ClamAV 协议版本、Fickling 版本、预算、worker/analyzer 与 ClamAV 配置 SHA-256。ClamAV 磁盘签名更新与运行中 daemon 的加载时间可能不同；以报告实际使用的身份记录为准。
