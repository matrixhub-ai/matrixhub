# 这是 case.md —— 业务测试流程的源头文件，一行一个 Case，按执行顺序排列。
# 你只需要维护这个文件（加 environment.yaml、可选 preparation.md/截图），AI 会生成其他运行资产。

# 第一列是 Case ID（英文，对应 cases/<ID>.yaml），用空格与描述分隔。
# # 开头的行（像本行）和空行会被忽略，可作为注释。
# 用 ${VAR} 引用 environment.yaml 里的变量，不要把地址/账号/资源名写死。

# 用户登录
login_success 访问 ${BASE_URL}；确认成功进入 MatrixHub 登录页面；输入用户名 ${USERNAME}、密码 ${PASSWORD}，点击“登录”；确认用户登录成功并进入 MatrixHub 首页


# 创建 MatrixHub 项目的 Case（公开 / 私有）
create_public_project 在当前已登录会话中选择“项目管理”菜单并进入项目管理列表页面；点击“创建项目”，项目名称输入 ${PUBLIC_PROJECT_NAME}，勾选项目类型选项，点击“确定”；确认公开项目 ${PUBLIC_PROJECT_NAME} 创建成功、列表中出现该项目，且项目类型及信息显示正确
create_private_project 在当前已登录会话中选择“项目管理”菜单并进入项目管理列表页面；点击“创建项目”，项目名称输入 ${PRIVATE_PROJECT_NAME}，不勾选项目类型选项，点击“确定”；确认私有项目 ${PRIVATE_PROJECT_NAME} 创建成功、列表中出现该项目，且项目类型及信息显示正确
