## 核心用例集

<table style="width: 100%; min-width: 1680px; border-collapse: collapse; table-layout: fixed; font-size: 14px; line-height: 1.45;">
  <colgroup>
    <col style="width: 4%;" />
    <col style="width: 10%;" />
    <col style="width: 10%;" />
    <col style="width: 14%;" />
    <col style="width: 16%;" />
    <col style="width: 23%;" />
    <col style="width: 23%;" />
  </colgroup>
  <thead>
    <tr>
      <th style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">编号</th>
      <th style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">模块</th>
      <th style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">测试点</th>
      <th style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">测试标题</th>
      <th style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">前提条件</th>
      <th style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">测试步骤</th>
      <th style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">预期结果</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">登录登出</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">登录</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">使用正确的信息登录</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已经存在用户test1（密码：test123）<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 输入matrixhub访问地址<br>2. 输入用户名：test1，密码：test123，点击登录<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 成功进入matrixhub登录页面<br>2. 用户登录成功，进入matrixhub首页</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">2</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">登录登出</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">登录</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">使用错误的信息登录</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已经存在用户test1（密码：test123）<br>2. 系统中不存在用户test111<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 输入matrixhub访问地址<br>2. 输入用户名：test1，密码：123456，点击登录<br>3. 输入用户名：test111，输入密码：test123，点击登录<br>4. 用户名输入框留空，输入密码：test123，点击登录<br>5. 输入用户名：test1，密码输入框留空，点击登录</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 成功进入matrixhub登录页面<br>2. 系统提示“用户名或密码错误”，登录失败，停留在登录页面，错误字段输入框高亮<br>3. 系统提示 “用户名或密码错误”，登录失败，停留在登录页，错误字段输入框高亮<br>4. 系统提示 “用户名不能为空”，登录失败，停留在登录页，错误字段输入框高亮<br>5. 系统提示 “密码不能为空”，登录失败，停留在登录页，错误字段输入框高亮</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">3</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">登录登出</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">登录</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">回车登录<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;"></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 输入 matrixhub 访问地址<br>2. 输入用户名：test1，密码：test123，按下回车键</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 成功进入matrixhub登录页面<br>2. 用户登录成功，进入 matrixhub 首页</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">4</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">登录登出</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">登录</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">密码显示/隐藏<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已经存在用户test1（密码：test123）​</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 输入 matrixhub 访问地址<br>2. 输入密码：test123，点击 “显示密码” 图标<br>3. 点击“隐藏密码”图标<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 成功进入matrixhub登录页面<br>2. 密码从密文变为明文显示<br>3. 密码恢复为密文显示</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">5</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">登录登出</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">登出</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">手动退出登录</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已经存在用户test1<br>2. 用户test1已成功登录<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 输入 matrixhub 访问地址<br>2. 输入用户名 test1、密码 test123，点击登录<br>3. 点击页面上的 “退出登录” 按钮</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 成功进入matrixhub登录页面<br>2. 用户登录成功，进入 matrixhub 首页<br>3. 系统提示 “确认退出登录”。确认后用户退出登录，返回 matrixhub 登录页面</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">6</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">登录登出</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">记住登录状态</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">未勾选「记住我的登录状态」—— 浏览器关闭后会话失效</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 存在用户 test1（密码：test123）<br>2. 浏览器无 test1 登录缓存<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 输入 matrixhub 访问地址<br>2. 输入 test1/test123，不勾选「记住我的登录状态」，点击登录<br>3. 登录成功后，完全关闭浏览器（所有标签页 / 窗口）<br>4. 重新打开浏览器，输入 matrixhub 访问地址</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 成功进入登录页面<br>2. 登录成功进入首页<br>3. 浏览器完全关闭<br>4. 需重新输入账号密码登录，会话已失效</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">7</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">登录登出</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">记住登录状态</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">勾选「记住我的登录状态」——30 天绝对上限强制失效</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;"></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 输入 matrixhub 访问地址<br>2. 输入 test1/test123，勾选「记住我的登录状态」，点击登录（记为第 1 天）<br>3. 每天执行至少 1 次有效操作（如刷新页面、点击菜单），持续至第 30 天<br>4. 第 31 天，打开浏览器访问 matrixhub</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 成功进入登录页面<br>2. 登录成功进入首页<br>3. 持续保持登录状态，每次操作后会话续期 7 天，但未超过 30 天绝对上限<br>4. 自动退出登录，提示「会话已过期，请重新登录」，跳转至登录页面（触发 30 天绝对上限）</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">8</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">登录登出</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">记住登录状态</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">统一强制失效场景 —— 管理员重置用户密码</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 存在用户 test1（密码：test123）<br>2. 存在管理员 admin1，可重置 test1 密码<br>3. 用户 test1 已勾选「记住我的登录状态」并登录成功</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 切换至管理员 admin1 账号，重置 test1 密码为 Resettest123<br>2. 切换回 test1 已登录的浏览器窗口，刷新页面或访问任意功能页面</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 管理员重置密码成功<br>2. test1 会话立即失效，自动退出登录，提示「会话已失效，请重新登录」，跳转至登录页面，需使用新密码 Resettest123 登录</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">9</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建模型或数据集</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建模型<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">合法的模型名称</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1<br>2. 存在项目 project01（公开）<br>3. 已存在模型名称test111<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入模型仓库页面<br>3. 进入创建模型页面<br>4. 输入模型名称 test-model-01，提交创建<br>5. 输入模型名称 t1，提交创建<br>6. 输入模型名称 1test2，提交创建<br>7. 输入模型名称 test.project，提交创建<br>8. 输入模型名称 test，提交创建<br>9. 输入模型名称 12，提交创建<br>10. 输入模型名称 12_，提交创建</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入模型仓库页面<br>3. 成功进入创建模型页面<br>4. 模型（test-model-01）创建成功<br>5. 模型（t1）创建成功<br>6. 模型（1test2）创建成功<br>7. 模型（test.project）创建成功<br>8. 模型（test）创建成功<br>9. 模型（12）创建成功<br>10. 模型（12_）创建成功</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">10</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建模型或数据集</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建模型<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">不符合规范/重复的模型名称<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;"></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入模型仓库页面<br>3. 进入创建模型页面<br>4. 输入模型名称 _test，提交创建<br>5. 输入模型名称 -test，提交创建<br>6. 输入模型名称 test 01，提交创建<br>7. 输入模型名称 test%123，提交创建<br>8. 输入模型名称 test*123，提交创建<br>9. 输入模型名称 test~01，提交创建<br>10. 输入模型名称 test@#$^&amp;*()+=01，提交创建<br>11. 输入模型名称 test111，提交创建</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入模型仓库页面<br>3. 成功进入创建模型页面<br>4. 模型创建失败，提示 “模型名称只能以数字、字母开头，且仅支持数字、字母、下划线、短横线和点”<br>5. 模型创建失败，提示 “模型名称只能以数字、字母开头，且仅支持数字、字母、下划线、短横线和点”<br>6. 模型创建失败，提示 “模型名称只能以数字、字母开头，且仅支持数字、字母、下划线、短横线和点”<br>7. 模型创建失败，提示 “模型名称只能以数字、字母开头，且仅支持数字、字母、下划线、短横线和点”<br>8. 模型创建失败，提示 “模型名称只能以数字、字母开头，且仅支持数字、字母、下划线、短横线和点”<br>9. 模型创建失败，提示 “模型名称只能以数字、字母开头，且仅支持数字、字母、下划线、短横线和点”<br>10. 模型创建失败，提示 “模型名称只能以数字、字母开头，且仅支持数字、字母、下划线、短横线和点”<br>11. 模型创建失败，提示 “模型已存在”</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">11</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建模型或数据集</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建模型<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">所属项目为私有项目</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1<br>2. 存在项目 project02（私有）<br>3. 当前用户对该项目有操作权限</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入模型仓库页面<br>3. 进入创建模型页面<br>4. 输入模型名称 test-model-01，所属项目选择 projec02，License 选择 Apache License 2.0，提交创建</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入模型仓库页面<br>3. 成功进入创建模型页面<br>4. 模型创建成功，该模型默认权限为 “私有”，仅项目成员可见</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">12</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建模型或数据集</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建模型<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">所属项目为公开项目</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1<br>2. 存在项目 project01（公开）</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入模型仓库页面<br>3. 进入创建模型页面<br>4. 输入模型名称 test-model-01，所属项目选择 projec01，License 选择 Apache License 2.0，提交创建</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入模型仓库页面<br>3. 成功进入创建模型页面<br>4. 模型创建成功，该模型默认权限为 “公开”，所有用户可见</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">13</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建模型或数据集</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建数据集</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">合法的数据集名称<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1<br>2. 存在项目 project01（公开）<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入数据集页面<br>3. 进入创建数据集页面<br>4. 输入数据集名称 `test-dataset-01`，提交创建<br>5. 输入数据集名称 `t1`，提交创建<br>6. 输入数据集名称 `1test2`，提交创建<br>7. 输入数据集名称 `test.dataset`，提交创建<br>8. 输入数据集名称 `test`，提交创建<br>9. 输入数据集名称 `12`，提交创建<br>10. 输入数据集名称 `12_`，提交创建</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入数据集页面<br>3. 成功进入创建数据集页面<br>4. 数据集（test-dataset-01）创建成功<br>5. 数据集（t1）创建成功<br>6. 数据集（1test2）创建成功<br>7. 数据集（test.dataset）创建成功<br>8. 数据集（test）创建成功<br>9. 数据集（12）创建成功<br>10. 数据集（12_）创建成功</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">14</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建模型或数据集</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建数据集</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">不符合规范的数据集名称<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;"></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入数据集页面<br>3. 进入创建数据集页面<br>4. 输入数据集名称 `_test`，提交创建<br>5. 输入数据集名称 `-test`，提交创建<br>6. 输入数据集名称 `test 01`，提交创建<br>7. 输入数据集名称 `test%123`，提交创建<br>8. 输入数据集名称 `test*123`，提交创建<br>9. 输入数据集名称 `test~01`，提交创建<br>10. 输入数据集名称 `test@#$^&amp;*()+=01`，提交创建<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入数据集页面<br>3. 成功进入创建数据集页面<br>4. 数据集创建失败，提示 “数据集名称只能以数字、字母开头，且仅支持数字、字母、下划线、短横线和点”<br>5. 数据集创建失败，提示 “数据集名称只能以数字、字母开头，且仅支持数字、字母、下划线、短横线和点”<br>6. 数据集创建失败，提示 “数据集名称只能以数字、字母开头，且仅支持数字、字母、下划线、短横线和点”<br>7. 数据集创建失败，提示 “数据集名称只能以数字、字母开头，且仅支持数字、字母、下划线、短横线和点”<br>8. 数据集创建失败，提示 “数据集名称只能以数字、字母开头，且仅支持数字、字母、下划线、短横线和点”<br>9. 数据集创建失败，提示 “数据集名称只能以数字、字母开头，且仅支持数字、字母、下划线、短横线和点”<br>10. 数据集创建失败，提示 “数据集名称只能以数字、字母开头，且仅支持数字、字母、下划线、短横线和点”</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">15</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建模型或数据集</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建数据集</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">所属项目为私有项目</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1<br>2. 存在项目 project02（私有）<br>3. 当前用户对该项目有操作权限</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入数据集页面<br>3. 进入创建数据集页面<br>4. 输入数据集名称 test-dataset-01，所属项目选择 projec02，License 选择 Apache License 2.0，提交创建</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入数据集页面<br>3. 成功进入创建数据集页面<br>4. 数据集创建成功，该数据集默认权限为 “私有”，仅项目成员可见</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">16</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建模型或数据集</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建数据集</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">所属项目为公开项目<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1<br>2. 存在项目 project01（公开）</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入数据集页面<br>3. 进入创建数据集页面<br>4. 输入数据集名称 test-dataset-01，所属项目选择 projec01，License 选择 Apache License 2.0，提交创建</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入数据集页面<br>3. 成功进入创建数据集页面<br>4. 数据集创建成功，该数据集默认权限为 “公开”，所有用户可见</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">17</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">模型列表</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">数据展示</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">正常加载模型列表<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1<br>2. 系统中存在多个模型<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入 “模型仓库” 页面<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入 “模型仓库” 页面。页面正常加载，显示 “热门模型” 和 “全部模型” 两个区域。模型卡片按设计展示，包含名称、标签、大小、项目、更新时间等信息。仅呈现用户有权限的模型。显示热门模型区域和全部模型区域，全部模型区域有“查看更多”按钮</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">18</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">模型列表</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">搜索</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">搜索存在的项目<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1<br>2. 存在项目 project01、project02、project03、dataProj01、dataProj02</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入 “模型仓库” 页面<br>3. 在项目名称搜索框输入 “project”<br>4. 在项目名称搜索框输入 “dataProj”<br>5. 在项目名称搜索框输入 “project01”<br>6. 在项目名称搜索框输入 “pro”</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 列表记录默认展示<br>3. 列表展示 3 条记录：project01、project02、project03，仅呈现用户有权限的项目，呈现顺序按照项目名称优先<br>4. 列表展示 2 条记录：dataProj01、dataProj02，仅呈现用户有权限的项目，呈现顺序按照项目名称优先<br>5. 列表展示 1 条记录：project01，仅呈现用户有权限的项目，呈现顺序按照项目名称优先<br>6. 列表展示 5 条记录：project01、project02、project03、dataProj01、dataProj02，仅呈现用户有权限的项目，呈现顺序按照项目名称优先</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">19</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">模型列表</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">搜索</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">搜索存在的模型<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1<br>2. 存在模型 test-model-01、test-model-02、test-model-03、data01、data02<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入 “模型仓库” 页面<br>3. 在模型名称搜索框输入 “test-model”<br>4. 在模型名称搜索框输入 “data”<br>5. 在模型名称搜索框输入 “test-model-01”<br>6. 在模型名称搜索框输入 “test”</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 列表记录默认展示<br>3. 列表展示 3 条记录：test-model-01、test-model-02、test-model-03，仅呈现用户有权限的模型，呈现顺序按照模型名称优先<br>4. 列表展示 2 条记录：data01、data02，仅呈现用户有权限的模型，呈现顺序按照模型名称优先<br>5. 列表展示 1 条记录：test-model-01，仅呈现用户有权限的模型，呈现顺序按照模型名称优先<br>6. 列表展示 5 条记录：test-model-01、test-model-02、test-model-03、data01、data02，仅呈现用户有权限的模型，呈现顺序按照模型名称优先</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">20</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">模型列表</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">筛选</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">按任务筛选<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1<br>2. 存在不同任务类型的模型：文本生成、图像文本到文本、图像到图像、文字转语音、文本转语音</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入模型仓库页面<br>3. 在左侧筛选栏选择 “任务”→“文本生成”<br>4. 在左侧筛选栏选择 “任务”→“图像文本到文本”<br>5. 在左侧筛选栏选择 “任务”→“图像到图像”<br>6. 在左侧筛选栏选择 “任务”→“文字转语音”<br>7. 在左侧筛选栏选择 “任务”→“文本转语音”</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入模型仓库页面<br>3. 列表仅展示任务类型为 “文本生成” 的模型，仅呈现用户有权限的模型<br>4. 列表仅展示任务类型为 “图像文本到文本” 的模型，仅呈现用户有权限的模型<br>5. 列表仅展示任务类型为 “图像到图像” 的模型，仅呈现用户有权限的模型<br>6. 列表仅展示任务类型为 “文字转语音” 的模型，仅呈现用户有权限的模型<br>7. 列表仅展示任务类型为 “文本转语音” 的模型，仅呈现用户有权限的模型</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">21</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">模型列表</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">权限控制</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">公开项目对所有用户可见</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1、test2（非 project01 项目成员）<br>2. 存在公开项目 project01（公开），其中包含模型 doubao</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 未登录状态下，进入模型仓库页面<br>2. 查看是否展示 doubao<br>3. 使用用户 test2 登录平台<br>4. 进入模型仓库页面<br>5. 查看是否展示 doubao</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 成功进入模型仓库页面<br>2. 未登录用户可看到 doubao<br>3. 用户 test2 登录成功<br>4. 成功进入模型仓库页面<br>5. 非项目成员 test2 可看到 doubao</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">22</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">数据集列表<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">数据展示</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">正常加载数据集列表</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1<br>2. 系统中存在多个数据集<br>3. 用户已登录</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入 “数据集仓库” 页面<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入 “数据集仓库” 页面。页面正常加载，数据集卡片按设计展示，包含名称、标签、样本数量、项目、更新时间等信息。仅呈现用户有权限的模型</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">23</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">数据集列表<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">搜索</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">搜索存在的数据集</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1<br>2. 存在数据集：test-dataset-01、test-dataset-02、test-dataset-03、data01、data02<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入 “数据集仓库” 页面<br>3. 在数据集名称搜索框输入 “test-dataset”<br>4. 在数据集名称搜索框输入 “data”<br>5. 在数据集名称搜索框输入 “test-dataset-01”<br>6. 在数据集名称搜索框输入 “test”<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 列表记录默认展示<br>3. 列表展示 3 条记录：test-dataset-01、test-dataset-02、test-dataset-03，仅呈现用户有权限的数据集，呈现顺序按照数据集名称优先<br>4. 列表展示 2 条记录：data01、data02，仅呈现用户有权限的数据集，呈现顺序按照数据集名称优先<br>5. 列表展示 1 条记录：test-dataset-01，仅呈现用户有权限的数据集，呈现顺序按照数据集名称优先<br>6. 列表展示 5 条记录：test-dataset-01、test-dataset-02、test-dataset-03、data01、data02，仅呈现用户有权限的数据集，呈现顺序按照数据集名称优先</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">24</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">数据集列表<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">权限控制</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">公开项目对所有用户可见</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1、test2（非 project01 项目成员）<br>2. 存在公开项目 project01（公开），其中包含数据集 Qwen/DeepPlanning</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 未登录状态下，进入数据集仓库页面<br>2. 查看是否展示 Qwen/DeepPlanning<br>3. 使用用户 test2 登录平台<br>4. 进入数据集仓库页面<br>5. 查看是否展示 Qwen/DeepPlanning</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 成功进入数据集仓库页面<br>2. 未登录用户可看到 Qwen/DeepPlanning<br>3. 用户 test2 登录成功<br>4. 成功进入数据集仓库页面<br>5. 非项目成员 test2 可看到 Qwen/DeepPlanning</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">25</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">模型详情<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">数据展示</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">正常加载模型详情页</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1<br>2. 存在模型 project 01 / Kimi-K2.5</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 从模型列表点击进入该模型详情页</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入模型详情页，页面正常加载，显示模型名称、标签、大小、项目、更新时间等信息</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">26</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">模型文件<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">下载</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">下载单个文件</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1<br>2. 模型中存在文件 README.md<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入 “模型文件” Tab<br>3. 在文件列表中找到 README.md,点击 “下载” 按钮</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入 “模型文件” Tab<br>3. 成功通过浏览器下载该文件。可以正常打开文件，文件内容正常</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">27</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">模型文件<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">下载</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">下载整个模型<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1<br>2. 存在模型 project 01 / Kimi-K2.5</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入 “模型文件” Tab<br>3. 点击 “下载模型” 按钮，在弹出的抽屉中复制下载命令<br>4. 打开命令行，运行复制的命令</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入 “模型文件” Tab<br>3. 弹出抽屉，显示下载命令指引。可复制命令，指引文档链接可跳转<br>4. 成功下载模型</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">28</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">模型文件<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">版本管理</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">切换版本</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1<br>2. 模型存在分支 main与分支 dev</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入 “模型文件” Tab<br>3. 从下拉框选择不同分支 / 版本</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入 “模型文件” Tab<br>3. 成功切换分支 / 版本。文件列表更新为对应版本的文件</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">29</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">模型文件<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">删除模型<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">正常删除模型</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1<br>2. test1对模型有删除权限<br>3. 存在模型，路径为：moonshotai / Kimi-K2.5</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入 “模型设置” Tab，在输入框输入完整模型路径 moonshotai / Kimi-K2.5<br>3. 点击 “删除模型” 按钮<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入 “模型设置” Tab<br>3. 操作成功，成功删除模型<br></td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">30</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">数据集详情<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">数据展示</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">正常加载数据集详情页</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1<br>2. 存在数据集 Qwen/DeepPlanning</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 从数据集列表点击进入该数据集详情页</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入数据集详情页，页面正常加载，显示数据集名称、标签、大小、项目、更新时间等信息</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">31</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">数据集详情<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">数据展示</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">下载整个数据集<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1<br>2. 存在数据集 Qwen/DeepPlanning</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入 “数据集文件” Tab<br>3. 点击 “下载数据集” 按钮</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入 “数据集文件” Tab<br>3. 弹出抽屉，显示下载命令指引。可复制命令，指引文档链接可跳转。复制的下载命令可用于下载。</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">32</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">数据集设置<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">删除数据集</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">正常删除数据集</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1<br>2. test1 对数据集有删除权限<br>3. 存在数据集，路径为：Qwen/DeepPlanning<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入 “数据集设置” Tab，在输入框输入完整数据集路径 Qwen/DeepPlanning<br>3. 点击 “删除数据集” 按钮</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入 “数据集设置” Tab<br>3. 操作成功，成功删除数据集<br></td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">33</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">平台设置-用户管理</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建用户-不设为管理员<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户 test1<br>2. 用户 test1 拥有创建用户的权限<br>3. 用户 test1 拥有设置用户为管理员的权限<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户test1登录平台<br>2. 进入 “用户管理” Tab<br>3. 点击 “创建用户” 按钮<br>4. 输入用户名 “test2”、密码 “123456”、确认密码 “123456”。不勾选设为管理员。点击 “确定”<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户test1登录成功<br>2. 成功进入用户管理页面<br>3. 出现创建用户的弹窗<br>4. 弹窗关闭，创建成功，用户列表新增用户 “test2”，平台管理员列显示 “否”<br></td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">34</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">平台设置-用户管理</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建用户-密码与确认密码不一致</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户 test1<br>2. 用户 test1 拥有创建用户的权限</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户test1登录平台<br>2. 进入 “用户管理” Tab<br>3. 点击 “创建用户” 按钮<br>4. 输入用户名 “test4”、密码 “123456”、确认密码 “123”，不勾选设为管理员，点击 “确定”</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户test1登录成功<br>2. 成功进入用户管理页面<br>3. 出现创建用户的弹窗<br>4. 操作失败，提示 “两次输入的密码不一致”，用户列表无新增</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">35</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">平台设置-用户管理</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">批量删除</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户 test1，拥有删除用户的权限<br>2. 存在用户 user01、user02<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户test1登录平台<br>2. 进入 “用户管理” Tab<br>3. 勾选用户 user01，user02，点击 “批量删除”。在确认弹窗中点击 “确定”<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户test1登录成功<br>2. 成功进入用户管理页面<br>3. 操作成功，用户列表中 user01 和 user02 的记录均消失，页面刷新后无该用户信息。<br></td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">36</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">平台设置-用户管理</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">成员管理<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">添加成员</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户 test1，拥有管理用户组成员的权限<br>2. 存在用户组 userGroup01，当前成员为 user1、user2、user3<br>3. 存在未加入该组的平台用户 user4</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入「用户组管理」Tab<br>3. 找到用户组 userGroup01，点击进入详情页，切换到「成员管理」区域<br>4. 点击「添加用户」按钮<br>5. 在添加用户弹窗中，选择 / 搜索用户 user4，确认添加，关闭弹窗，查看成员列表</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入用户组管理页面<br>3. 成功进入 userGroup01 详情页，成员管理区域展示当前成员 user1、user2、user3<br>4. 弹出添加用户弹窗，可搜索 / 选择平台内用户<br>5. 弹窗关闭，添加成功，成员列表新增用户 user4，来源、创建时间等信息正常展示，「基本信息」区域的「成员数量」从 3 更新为 4</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">37</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">平台设置-用户管理</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">成员管理<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">删除成员<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户 test1，拥有管理用户组成员的权限<br>2. 存在用户组 userGroup01，成员为 user1、user2、user3、user4</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入「用户组管理」Tab<br>3. 找到用户组 userGroup01，点击进入详情页，切换到「成员管理」区域<br>4. 在成员列表中找到用户 user4，点击操作列的「删除」按钮<br>5. 在确认弹窗中点击「确定」</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入用户组管理页面<br>3. 成功进入 userGroup01 详情页，成员管理区域展示当前成员<br>4. 弹出删除确认弹窗，提示「确定要将用户 user4 从该用户组移除吗？」<br>5. 成员列表中 user4 记录消失，「基本信息」区域的「成员数量」从 4 更新为 3</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">38</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">平台设置-用户管理</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">同步</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">手动执行同步规则</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户test1<br>2. 存在状态为 “启用” 的手动同步规则</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入 “远程同步” 页面<br>3. 找到启用状态的同步规则，点击 “同步” 按钮</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入远程同步页面<br>3. 系统触发同步任务，显示同步中状态，同步完成后提示同步成功（或失败原因）</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">39</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">平台设置-用户管理</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">搜索</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">搜索存在的同步规则<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;"></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 选择 “平台设置 - 远程同步” 菜单<br>3. 在搜索框输入 “sync”<br>4. 在搜索框输入 “test”<br>5. 在搜索框输入 “test-sync”<br>6. 在搜索框输入 “-01”<br>7. 在搜索框输入 “push”</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入远程同步页面，列表数据分页展示<br>3. 列表展示 4 条记录：kangaroo-sync、test-sync-01、test-sync-02、sync-push-demo<br>4. 列表展示 2 条记录：test-sync-01、test-sync-02<br>5. 列表展示 2 条记录：test-sync-01、test-sync-02<br>6. 列表展示 1 条记录：test-sync-01<br>7. 列表展示 1 条记录：sync-push-demo</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">40</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">平台设置-用户管理</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">停止任务<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">停止任务<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已存在用户：test1<br>2. 存在进行中状态的远程同步任务：task-running-01（ID：3013），当前用户具备停止操作权限</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户 test1 登录平台<br>2. 进入 “平台设置 - 远程同步 - 任务” 页面<br>3. 找到进行中任务 task-running-01，点击操作列的 “停止任务” 按钮<br>4. 确认停止操作<br>5. 查看任务状态变化</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户 test1 登录成功<br>2. 成功进入 “平台设置 - 远程同步 - 任务” 页面<br>3. 系统提示 “确认停止任务”<br>4. 确认后，任务状态由 “进行中” 变更为 “已停止”<br>5. 任务列表中该任务状态同步更新为 “已停止”</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">41</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">平台设置-用户管理</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">停止任务<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">私有项目</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;"></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户test1登录平台<br>2. 选择项目管理菜单<br>3. 点击“创建项目”，输入项目名称（test-project1），选择类型（不勾选），点击“确定”<br>4. 使用用户test2登录平台<br>5. 选择项目管理菜单</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户test1登录成功<br>2. 进入项目管理列表页面<br>3. 私有项目（test-project1）创建成功，列表出现该条记录<br>4. 用户test2登录成功<br>5. 进入项目管理列表页面，无法查看到私有项目（test-project1）</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">42</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">项目管理</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">合法的项目名称<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已经存在用户test1<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户test1登录平台<br>2. 选择项目管理菜单<br>3. 点击“创建项目”，输入项目名称（test-project1），选择类型，点击“确定”<br>4. 点击“创建项目”，输入项目名称（t1），选择类型，点击“确定”<br>5. 点击“创建项目”，输入项目名称（1test2），选择类型，点击“确定”<br>6. 点击“创建项目”，输入项目名称（test-project），选择类型，点击“确定”<br>7. 点击“创建项目”，输入项目名称（test），选择类型，点击“确定”<br>8. 点击“创建项目”，输入项目名称（12），选择类型，点击“确定”</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户test1登录成功<br>2. 进入项目管理列表页面<br>3. 项目（test-project1）创建成功，列表出现该条记录<br>4. 项目（t1）创建成功，列表出现该条记录<br>5. 项目（1test2）创建成功，列表出现该条记录<br>6. 项目（test-project）创建成功，列表出现该条记录<br>7. 项目（test）创建成功，列表出现该条记录<br>8. 项目（12）创建成功，列表出现该条记录</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">43</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">项目管理</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">不符合规范的项目名称</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;"></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户test1登录平台<br>2. 选择项目管理菜单<br>3. 点击“创建项目”，输入项目名称（t），选择类型，点击“确定”<br>4. 点击“创建项目”，输入项目名称（-test），选择类型，点击“确定”<br>5. 点击“创建项目”，输入项目名称（test 01），选择类型，点击“确定”<br>6. 点击“创建项目”，输入项目名称（test%123），选择类型，点击“确定”<br>7. 点击“创建项目”，输入项目名称（test*123），选择类型，点击“确定”<br>8. 点击“创建项目”，输入项目名称（test~01），选择类型，点击“确定”<br>9. 点击“创建项目”，输入项目名称（test@#$^&amp;*()+=01），选择类型，点击“确定”<br>10. 点击“创建项目”，输入项目名称（1test-），选择类型，点击“确定”<br>11. 点击“创建项目”，输入项目名称（Test），选择类型，点击“确定”<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户test1登录成功<br>2. 进入项目管理列表页面<br>3. 项目创建失败，出现提示信息 “项目名称由小写字符、数字和连字符组成且至少2个字符并以字符或者数字开头和结尾。”<br>4. 项目创建失败，出现提示信息 “项目名称由小写字符、数字和连字符组成且至少2个字符并以字符或者数字开头和结尾。”<br>5. 项目创建失败，出现提示信息 “项目名称由小写字符、数字和连字符组成且至少2个字符并以字符或者数字开头和结尾。”<br>6. 项目创建失败，出现提示信息 “项目名称由小写字符、数字和连字符组成且至少2个字符并以字符或者数字开头和结尾。”<br>7. 项目创建失败，出现提示信息 “项目名称由小写字符、数字和连字符组成且至少2个字符并以字符或者数字开头和结尾。”<br>8. 项目创建失败，出现提示信息 “项目名称由小写字符、数字和连字符组成且至少2个字符并以字符或者数字开头和结尾。”<br>9. 项目创建失败，出现提示信息 “项目名称由小写字符、数字和连字符组成且至少2个字符并以字符或者数字开头和结尾。”<br>10. 项目创建失败，出现提示信息 “项目名称由小写字符、数字和连字符组成且至少2个字符并以字符或者数字开头和结尾。”<br>11. 项目创建失败，出现提示信息 “项目名称由小写字符、数字和连字符组成且至少2个字符并以字符或者数字开头和结尾。”</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">44</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">项目管理</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">公开项目</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已经存在用户test1，test2</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户test1登录平台<br>2. 选择项目管理菜单<br>3. 点击“创建项目”，输入项目名称（test-project1），选择类型（勾选），点击“确定”<br>4. 使用用户test2登录平台<br>5. 选择项目管理菜单</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户test1登录成功<br>2. 进入项目管理列表页面<br>3. 公开项目（test-project1）创建成功，列表出现该条记录<br>4. 用户test2登录成功<br>5. 进入项目管理列表页面，可以查看到公开项目（test-project1），信息显示正确</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">45</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">项目管理</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">代理项目</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已经存在用户test1（平台管理员）<br>2. 已经存在目标仓库（HF）<br>3. 存在公开模型 matrixhub/test123<br>4. 存在私有模型 matrixhub/test-mn</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户test1登录平台<br>2. 选择项目管理菜单<br>3. 点击“创建项目”，输入项目名称（test-project1），选择类型（不勾选），勾选“代理”，选择目标仓库（HF），输入Org（matrixhub），点击“确定”<br>4. 点击“创建项目”，输入项目名称（test-project2），选择类型（不勾选），勾选“代理”，选择目标仓库（HF），输入用户名（testuser），点击“确定”<br>5. 输入命令下载公开模型 matrixhub/test123 （`hf download matrixhub/test123`）<br>6. 输入命令下载私有模型 matrixhub/test-mn （`hf download matrixhub/test-mn`）</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户test1登录成功<br>2. 进入项目管理列表页面<br>3. 项目（test-project1）创建成功，列表出现该条记录<br>4. 项目（test-project2）创建成功，列表出现该条记录<br>5. 公开模型下载成功<br>6. 无法下载私有模型</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">46</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">项目管理</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">删除</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">删除项目</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;"></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户test1登录平台<br>2. 选择项目管理菜单<br>3. 选择项目（test-project1），点击“删除”<br>4. 点击“确定”</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户test1登录成功<br>2. 进入项目管理列表页面<br>3. 出现二次删除项目提示弹框<br>4. 项目（test-project1）删除成功，列表无该条记录信息</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">47</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">个人中心<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">访问密钥<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建永久密钥</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已经存在用户（test1 / test123）<br>2. 存在私有模型 matrixhub666/test-mn<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户test1登录平台<br>2. 选择“个人中心”<br>3. 点击“访问密钥”<br>4. 点击“创建密钥”<br>5. 输入名称（test-11111），选择“永久”有效期，点击“确定”<br>6. 复制生成的密钥信息<br>7. 使用生成的密钥登录私有项目进行模型下载<br>    1. 打开控制台，输入命令： `hf auth login`<br>    2. 输入生成的密钥<br>    3. 配置访问地址 `export HF_ENDPOINT="matrixhub.url"`<br>    4. 输入下载命令： `hf download matrixhub666/test-mn`<br>    5. 输入上传命令： `hf upload  matrixhub666/test-mn123`<br>    6. 退出登录 `hf auth logout`<br>    7. 再次输入下载命令：`hf download matrixhub666/test-mn`</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户test1登录成功<br>2. 进入个人中心页面，存在两个页面：安全设置以及访问密钥<br>3. 进入访问密钥页面<br>4. 进入创建密钥页面<br>5. 密钥创建成功<br>6. 密钥复制成功<br>7. 模型下载成功<br>    1. 命令正常输入，提示输入密钥信息<br>    2. 登录成功<br>    3. 地址配置成功<br>    4. 模型下载成功<br>    5. 模型上传成功<br>    6. 退出登录成功<br>    7. 出现提示 “make sure you are authenticated ”</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">48</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">个人中心<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">SSH公钥</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">创建永久SSH公钥</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已经存在用户（test1 / test123）<br>2. 存在私有模型 matrixhub666/test-mn<br>3. 存在公共模型matrixhub666/test-public-model<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用用户test1登录平台<br>2. 选择“个人中心”<br>3. 点击“SSH公钥”<br>4. 点击“导入SSH公钥”<br>5. 输入名称（test-11111），导入公钥，选择“永久”有效期，点击“确定 （参考文档：https://docs.daocloud.io/ghippo/user-guide/personal-center/ssh-key/#1-ssh）<br>6. 对私有项目进行模型下载<br>    1. 打开控制台，输入命令： `git clone `[`git@matrixhub.com`](mailto:git@matrixhub.com)`:matrixhub666/test-mn`<br>    2. 修改当前项目，增加一个test文件后使用如下命令： <br>        1. `git checkout -b add-file`<br>        2. `git add *`<br>        3. `git commit -m "add text file"`<br>        4. `git push --set-upstream origin add-file`</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 用户test1登录成功<br>2. 进入个人中心页面，存在三个页面：安全设置、访问密钥以及SSH公钥<br>3. 进入SSH公钥页面<br>4. 进入导入SSH公钥弹框页面<br>5. 公钥导入成功<br>6. 模型下载成功<br>    1. git clone成功，正常下载模型项目文件<br>    2. 结果如下<br>        1. 创建分支成功<br>        2. 文件暂存成功<br>        3. 暂存区的修改正式提交到本地仓库的历史记录中<br>        4. 推送成功</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">49</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">机器人账号</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">项目 - 机器人账号</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">使用机器人账号进行hf操作 - 正确的令牌<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 已经存在私有模型 `matrixhub666/test-mn`以及 `matrixhub666/test-mn123`<br>2. 已经存在机器人账号`robot$test-robot`<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用机器人账号 `robot$test-robot` 登录对私有项目进行以下操作<br>    1. 打开控制台，输入命令： `hf auth login`<br>    2. 输入生成的令牌信息<br>    3. 配置访问地址 `export HF_ENDPOINT="matrixhub.url"`<br>    4. 输入下载命令： `hf download matrixhub666/test-mn`<br>    5. 输入上传命令： `hf upload  matrixhub666/test-mn123`<br>    6. 退出登录 `hf auth logout`<br>    7. 再次输入下载命令：`hf download matrixhub666/test-mn`</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 结果如下<br>    1. 命令正常输入，提示输入密钥信息<br>    2. 登录成功<br>    3. 地址配置成功<br>    4. 模型下载成功<br>    5. 模型上传成功<br>    6. 退出登录成功<br>    7. 出现提示 “make sure you are authenticated ”</td>
    </tr>
    <tr>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">50</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">机器人账号</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">项目 - 机器人账号</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">使用机器人账号进行git 操作- 正确的令牌<br></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;"></td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 使用机器人账号 `robot$test-robot` 登录对私有项目进行以下操作<br>    1. 使用 git clone 下载 `matrixhub666/test-mn` 模型仓库 （git clone 'http://robot$test-robot:&lt;令牌信息&gt;@x.x.x.x:3001/matrixhub666/test-mn.git'）<br>    2. 使用git push 推送 （git push http://robot$test-robot:&lt;令牌信息&gt;@x.x.x.x:3001/matrixhub666/test-mn.git main） <br>    3. 使用git pull 拉取 （git pull http://robot$test-robot:&lt;令牌信息&gt;@x.x.x.x:3001/matrixhub666/test-mn.git）</td>
      <td style="border: 1px solid rgba(127, 127, 127, 0.5); vertical-align: top; padding: 6px 8px; overflow-wrap: anywhere; word-break: normal;">1. 结果如下<br>    1. git clone成功<br>    2. git push 成功<br>    3. git pull 成功</td>
    </tr>
  </tbody>
</table>
