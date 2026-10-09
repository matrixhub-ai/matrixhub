# MatrixHub 模型制品安全扫描与准入原型


> 当前设计、容量边界及验证汇总见 [artifact-security-admission.md](../../docs/design/artifact-security-admission.md)。


这是基于 MatrixHub `6890726abe272767d26ce1a6b0580b0718f232c0` 开发的独立参赛工作目录。启用扫描后，文件更新按不可变提交排队，HF 下载、Git HTTP 整仓下载、LFS 批量与对象直取受准入策略控制。网页“安全”页提供文件依据、操作记录、复扫、取消和项目策略。

扫描使用 ClamAV 与 Fickling 0.1.12；不加载模型，不调用 Pickle 反序列化，不执行上传脚本。未完成或失败的扫描默认拒绝下载。高风险匹配不能通过调整失败策略放行。

实测汇总与脱敏证据索引见 `docs/design/artifact-security-admission.md`。

## Docker Compose 部署

要求 Linux/WSL、Docker Engine 与 Compose v2，可访问镜像仓库和依赖源。构建阶段需要下载 Go 工具链、锁定的前端依赖及扫描器依赖。约需 4 GiB 可用内存，另留源码、构建缓存、模型存储空间。Compose 绑定主机回环地址，仅供本地演示。

在项目根目录执行：

```bash
bash deploy/security-prototype/prepare-runtime.sh
docker compose -f deploy/security-prototype/compose.yaml up -d --build
docker compose -f deploy/security-prototype/compose.yaml ps
```

等 `clamav` 与 `scanner` 显示 healthy 后访问 `http://127.0.0.1:13872`。本地初始化管理员为 `admin` / `changeme`，登录后修改密码。正式部署应另行完成 HTTPS、账户初始化及持久化备份配置。

本地 SSH 端口为 `13873`，服务端使用非特权端口 2222。先在账户设置添加自己的公钥；私钥保留在本机。

构建脚本使用本目录代码，复制二进制、数据库迁移和前端产物到 `.mh-local/runtime-bundle`，不会复制测试数据库、Cookie、API 令牌或模型样本。运行数据使用 Compose 的独立命名卷；`down` 不删除数据，不要加 `-v`，除非确认要清空此演示实例。

```bash
docker compose -f deploy/security-prototype/compose.yaml logs --tail=50 matrixhub scanner clamav
docker compose -f deploy/security-prototype/compose.yaml down
```

扫描器只接入 internal 网络；ClamAV 额外接入更新网络下载签名。扫描器配置只读根目录、32 MiB /tmp 与持久 spool 卷中的匿名临时文件、1 CPU、512 MiB 内存、32 个进程、无 Linux capabilities；不对主机发布扫描器端口。

## 可复现验证

在运行的、独立测试实例上执行，勿对已有业务仓库执行测试。脚本会创建测试项目、仓库和测试令牌；不会把令牌写入结果文件。Python 依赖为 `huggingface_hub==1.29.0`，建议安装在独立虚拟环境。

```bash
export HF_HUB_DISABLE_XET=1
export MH_PROBE_ENDPOINT=http://127.0.0.1:13871
python3 deploy/security-prototype/probe.py
python3 deploy/security-prototype/probe_admission.py
python3 deploy/security-prototype/probe_web_session.py
```

Compose 隔离实例的验证命令（脚本会暂停并恢复这一个测试扫描器）：

```bash
export HF_HUB_DISABLE_XET=1
export MH_PROBE_ENDPOINT=http://127.0.0.1:13872
export MH_PROBE_RESULT_ROOT=.mh-local/probe-stage2/compose
export MH_PROBE_SCANNER_CONTAINER=mh-security-local-scanner-1
export MH_PROBE_SERVER_CONTAINER=mh-security-local-matrixhub-1
python3 deploy/security-prototype/probe.py
python3 deploy/security-prototype/probe_admission.py
python3 deploy/security-prototype/probe_web_session.py
python3 deploy/security-prototype/probe_ssh.py
python3 deploy/security-prototype/probe_scanner.py
python3 deploy/security-prototype/probe_large_capacity.py
python3 deploy/security-prototype/probe_public_checkpoint.py
python3 deploy/security-prototype/probe_ml_admission.py
python3 deploy/security-prototype/probe_lifecycle_compose.py
```

`probe_lifecycle_compose.py` 依赖先前 `probe.py` 生成的 results.json；它在显式指定的测试容器上检查离线拒绝、恢复及重启后的状态保留。容器名称来自本文件 Compose 项目，改过项目名时先用 `docker compose -f deploy/security-prototype/compose.yaml ps` 核对。

设置 HF_HUB_DISABLE_XET=1 使用本原型已实现的 HTTP/LFS 上传路径；Xet 上传未实现。大容量脚本会分别上传 256 MiB 和 1 GiB 的受控容器并校验下载 SHA-256，需预留至少 4 GiB 磁盘空间；不要反复运行。公开检查点脚本访问 Hugging Face 上的一个固定小型测试模型，仅进行静态扫描，不加载模型。脚本使用测试管理员；修改过密码时先调整本地测试账户配置。`probe_scanner.py` 在 Compose 上须指定上述扫描器容器名；所有探针均只用于独立测试实例。

结果保存在 `.mh-local/`；仓库中的证据目录是脱敏快照。

## 当前处理预算

| 项目 | 配置 |
| --- | --- |
| 单文件传输/扫描 | 1 GiB；256 MiB 与 1 GiB 受控容器实测 |
| Pickle 元数据 | 单段 8 MiB；一文件累计 32 MiB；静态子进程最多 10 秒 |
| 单版本文件数 | 256 |
| Git 快照 | 256 refs / 256 可达提交；128 MiB pack；30 秒创建，超限拒绝 |
| ZIP 成员/总解压体积/膨胀比 | 100 / 1280 MiB / 200 |
| 单版本处理预算 | Compose 900 秒；未配置时默认 600 秒 |
| 扫描器 HTTP / worker / ClamAV | 600 秒 / 540 秒 / 480 秒 |
| 调度 | 一个执行循环；不同上传任务可排队 |

超出预算、格式不支持、扫描器离线等情况记录为未完成，严格策略拒绝下载。超过 1 GiB 的文件拒绝；本轮没有扩大 Git pack 预算。

## 授权与交付边界

MatrixHub 与新增适配代码按 Apache-2.0；Fickling 0.1.12 为外部 LGPL-3.0 依赖，ClamAV 为外部 GPL-2.0 扫描器。组件版本和来源见 THIRD_PARTY.md；发布容器时保留组件的许可证、版本与源码获取说明；参赛代码不复制它们的实现。

接口与数据库表采用 opt-in 原型契约。迁移文件/API 契约已提供，Git 并发写入边界已用固定快照及回归处理；正式社区交付仍需契约评审、更多规模测量和维护者认可。数据集上传受上游已有 model-only 接收钩子影响，未列入通过案例。社区设计与集成评审仍需完成。

## 格式与复核

纯数据 Pickle 协议 0—5、ClamAV 标准测试字符串、损坏/尾随数据、重命名序列化与压缩预算已有受控回归。PyTorch 张量字节识别须先获得元数据的有限模型重建警告，不能只凭 data.pkl 与 data/数字文件名跳过检查；数字成员仍检查可识别的 Pickle 调用，整容器继续经过 ClamAV。重复 ZIP 名称返回失败。

已命中的高风险始终阻断，即使该文件其他检查失败、项目允许失败下载。失败/规则变化/保存失败不抹掉已返回的风险证据。最新结果见 docs/design/artifact-security-admission.md。边界复现入口为 probe_guard_edges.py，须先执行 probe_public_checkpoint.py 准备固定公开元数据样本；只针对独立测试实例运行。

仅含选定张量重建函数和 FloatStorage/LongStorage、OrderedDict 的有限静态轮廓会得到中风险 ML_CONSTRUCTION_REVIEW，严格策略阻止；项目负责人复核运行依赖及来源后可使用高风险阈值策略。torch.load、torch.storage._load_from_bytes 与任意额外危险调用不进入这一轮廓。

嵌套压缩、加密 ZIP、NumPy 序列化、超预算或无法完整静态分析记录失败，不伪装成通过。Safetensors 目前只做通用 ClamAV 检查及按文件名分类，尚无结构专用验证。扫描通过表示这些配置检查完成，不保证权重行为、模型后门、训练数据或运行时依赖安全。
