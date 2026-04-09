# MatrixHub 文档编写指南（Agent用）

## 任务：创建操作文档

当用户要求编写操作文档时，按以下流程执行：

---

## 1. 目录结构规范

```
docs/
├── intro.md
├── getting-started/
└── operations/              ← 新内容放这里
    ├── _category_.json
    ├── overview.md
    ├── projects/
    ├── assets/
    └── developer/
```

---

## 2. 文件模板

**Markdown Frontmatter 模板：**
```yaml
---
sidebar_position: 1
---
```

**分类配置模板（`_category_.json`）：**
```json
{
  "label": "操作文档",
  "position": 3
}
```

---

## 3. Sidebars 更新规则

在 `sidebars.ts` 的 `tutorialSidebar` 数组中添加：

```typescript
{
  type: 'category',
  label: '操作文档',
  items: [
    'operations/overview',
    'operations/projects/create-delete',
    'operations/projects/members',
    'operations/assets/manage',
    'operations/assets/upload-download',
    'operations/assets/popular',
    'operations/repository',
    'operations/developer/access-token',
  ],
}
```

---

## 4. 多语言支持

- **英文原文**：保持现有 `docs/` 结构不变
- **中文翻译**：放 `i18n/zh/docusaurus-plugin-content-docs/current/`，结构与 `docs/` 一一对应

**中文翻译文件示例：**
```markdown
---
sidebar_position: 1
---

# 介绍

（中文内容）
```

---

## 5. 界面元素翻译

主题文案在 `i18n/zh/docusaurus-theme-classic/`：
- `footer.json` - 页脚导航链接标签
- `navbar.json` - 导航栏菜单

---

## 6. 文档写作风格规范（参考 d.run 规范）

### 6.1 整体结构

- 页面顶部可选 frontmatter（`---` 块），用于 `sidebar_position` 等控制，非必须时不加
- 每个文档必须从 `#` 一级标题开始
- 标题层级：`#` 页面标题 → `##` 二级章节 → `###` 三级小节，**不超过三级**
- 固定章节模板顺序：**前提条件** → **操作步骤** → **配置参数说明**（按需）

### 6.2 操作步骤（有序列表）

- 序号统一写 `1.`，Docusaurus 自动递增，不手动写 `2.` `3.`
- **禁止** `### 1.` 或把序号与标题混用
- 序号下的图片、代码块、表格等附属内容，用 **4 空格缩进** 挂在该步骤下
- 序号下有多行续行文本，同样 **4 空格缩进**，不用 bullet points 嵌套

```markdown
1. 登录 MatrixHub 平台，选择 **模型仓库** ，点击 **上传模型** 。

    ![上传界面](./images/upload.png)

    | 参数 | 说明 |
    |-----|------|
    | 模型名称 | 模型的唯一标识符 |

1. 点击 **确定** ，完成上传。
```

### 6.3 无序列表

- 列表项之间**不空行**（紧凑排列）
- 行内关键字用粗体 `**词语**` 标注
- 列表项内有冒号说明时，直接接文字，不额外套子列表

```markdown
- **私有模型托管:** 安全存储微调后的模型权重，支持版本控制
- **高速缓存分发:** 通过 P2P 技术加速模型分发，降低带宽成本
```

### 6.4 粗体渲染（重要）

粗体 `**词语**` **前后必须各加一个空格**，否则中文环境下渲染可能异常。

```markdown
✅ 点击 **确定** ，完成实例创建。
✅ 选择 **模型仓库** -> **上传模型**
❌ 点击**确定**，完成实例创建。
```

### 6.5 图片

- 路径用相对路径：`./images/xxx.png` 或 `images/xxx.png`
- 文字行与图片之间必须空一行
- 图片在步骤内时，整行 4 空格缩进

```markdown
1. 点击列表右侧的 **┇** ，在下拉列表中选择 **删除**

    ![删除确认](./images/delete.png)
```

- 独立段落下的图片（非步骤内），顶格写，上方空一行：

```markdown
这是 GPU 缓存的工作流程。

![缓存架构](./images/cache-flow.png)
```

### 6.6 Admonition 提示块（Docusaurus 语法）

Docusaurus 使用 `:::` 语法而非 MkDocs 的 `!!!`：

```markdown
:::note

- 15 天未访问的模型文件可能会被清理缓存。
- 关机后 GPU 资源不会预留。

:::

:::tip

购买预留实例时长越长，享受的价格折扣通常越大。

:::

:::warning

删除操作不可逆，请提前备份重要数据。

:::
```

- `:::` 后接类型（note/tip/warning/danger/info），下一行空行 + 4 空格缩进写内容
- note/tip/warning 内可用 bullet list 或普通段落
- 不在提示块内嵌套标题

### 6.7 表格

- 表格在步骤内时 4 空格缩进挂在该步骤下
- 单元格内换行用 `<br/>`，多个选项用 `<br/>- **选项名:** 说明` 格式
- 表头简洁：`参数 | 说明` 或 `名称 | 说明`

```markdown
| 参数 | 说明 |
|------|------|
| 存储类型 | **标准存储:** 适合长期归档<br/>**高性能存储:** 适合频繁访问 |
| 访问权限 | **私有:** 仅自己可见<br/>**公开:** 所有人可下载 |
```

### 6.8 Docusaurus Tabs（标签页）

Tab 页内部不嵌套任何标题（`##` `###` 均不允许）：

```markdown
import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

<Tabs>
<TabItem value="docker" label="Docker 部署">

- 安装 Docker 20.10+
- 运行部署命令

```bash
docker-compose up -d
```

</TabItem>
<TabItem value="k8s" label="Kubernetes 部署">

- 安装 Helm 3+
- 添加 Chart 仓库

```bash
helm install matrixhub ./chart
```

</TabItem>
</Tabs>
```

### 6.9 分隔线

- `---` 仅用于 frontmatter 的开闭，正文中不使用 `---` 作为视觉分隔
- 章节间用标题层级区分，不用水平线

### 6.10 文字风格

- 产品名、按钮名、UI 元素用粗体 `**XX**`，前后加空格
- UI 路径用 `->` 连接：`**模型仓库** -> **上传模型**`
- 数字与中文之间加空格：`30 GB`、`50 GB`、`7 天`
- 链接与前后中文之间不加空格：`请参见[快速入门](./quickstart.md)文档`
