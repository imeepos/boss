# shadcn/ui 组件速查手册

> 本文件收录 shadcn/ui registry 全部 63 个组件，按用途分类，标注已安装组件。
> 已安装标记：`[installed]` = 已在 `web/admin/src/components/ui/` 下落地。
> 安装命令：`node .agents/skills/self-evolving/scripts/shadcn.mjs add <name>`

---

## 1. 基础交互 (Button / Toggle)

| 组件名 | 用途场景 | 状态 |
|---|---|---|
| `button` | 主操作按钮、次操作按钮、幽灵按钮、危险按钮、链接按钮、图标按钮、加载态按钮 | [installed] |
| `button-group` | 按钮组（工具栏、分段控制器外层） | |
| `toggle` | 单开关切换（加粗/斜体/下划线等工具栏按钮） | |
| `toggle-group` | 多选开关组（对齐方式、列表/网格视图切换） | |

## 2. 表单输入 (Input / Selection)

| 组件名 | 用途场景 | 状态 |
|---|---|---|
| `input` | 单行文本输入（搜索框、账号、邮箱、手机号等） | [installed] |
| `textarea` | 多行文本输入（备注、描述、评论等） | [installed] |
| `checkbox` | 单/多项选择（同意条款、批量选择行） | [installed] |
| `radio-group` | 单选组（性别、支付方式、配送方式等互斥选项） | [installed] |
| `select` | 下拉选择器（国家选择、状态筛选、排序方式） | [installed] |
| `native-select` | 原生 select（移动端场景，使用系统原生选择器） | |
| `switch` | 开关切换（启用/禁用、功能开关、设置项） | [installed] |
| `slider` | 滑块（价格范围、音量、透明度等连续值选择） | |
| `input-group` | 输入框组（前缀/后缀图标、组合输入如金额+单位） | |
| `input-otp` | 一次性密码输入（验证码、MFA 双因素认证） | |
| `label` | 表单标签（与 input/checkbox/radio 配合使用） | [installed] |
| `form` | 表单容器（集成 react-hook-form + zod 校验） | [installed] |
| `field` | 表单字段包装器（label + input + error 一体化） | |

## 3. 数据展示 (Display)

| 组件名 | 用途场景 | 状态 |
|---|---|---|
| `table` | 数据表格（订单列表、用户列表、商品列表等） | [installed] |
| `card` | 卡片容器（信息卡片、统计卡片、设置卡片） | [installed] |
| `badge` | 徽章/标签（状态标签、数量角标、版本标记） | [installed] |
| `avatar` | 头像（用户头像、群组头像、带状态指示） | |
| `calendar` | 日历（日期选择、日程展示、预订系统） | |
| `chart` | 图表（基于 recharts 的折线/柱状/饼图封装） | |
| `empty` | 空状态（无数据、无搜索结果、404 页面） | |
| `skeleton` | 骨架屏（加载占位、内容预占位） | [installed] |
| `separator` | 分隔线（列表项分隔、区域分隔、菜单分组） | [installed] |
| `kbd` | 键盘快捷键显示（⌘C、⌘V 等快捷键提示） | |
| `aspect-ratio` | 固定宽高比容器（图片嵌入、视频播放器） | |
| `scroll-area` | 自定义滚动区域（带自定义滚动条的容器） | |
| `progress` | 进度条（文件上传、任务进度、步骤完成度） | |

## 4. 反馈与提示 (Feedback)

| 组件名 | 用途场景 | 状态 |
|---|---|---|
| `alert` | 警告提示（成功/警告/错误/信息横幅） | [installed] |
| `alert-dialog` | 确认对话框（删除确认、危险操作二次确认） | |
| `dialog` | 对话框（模态弹窗、详情查看、表单弹窗） | [installed] |
| `drawer` | 抽屉（侧边滑出面板、详情抽屉、设置抽屉） | |
| `sheet` | 底部/侧边抽屉（移动端操作面板、筛选面板） | [installed] |
| `popover` | 气泡卡片（信息提示、快速操作面板） | [installed] |
| `tooltip` | 文字提示（按钮说明、截断文本全显） | [installed] |
| `hover-card` | 悬停卡片（用户信息预览、链接预览） | |
| `toast` | 轻量通知（操作成功/失败、后台任务完成） | |
| `sonner` | 现代化 toast 通知（支持 promise、action 按钮） | [installed] |

## 5. 导航与布局 (Navigation)

| 组件名 | 用途场景 | 状态 |
|---|---|---|
| `tabs` | 标签页（页签切换、内容分区） | [installed] |
| `breadcrumb` | 面包屑（层级导航、路径回溯） | |
| `pagination` | 分页器（列表分页、数据翻页） | |
| `menubar` | 菜单栏（应用级菜单、编辑器等工具栏） | |
| `navigation-menu` | 导航菜单（顶部导航、大型菜单下拉） | |
| `sidebar` | 侧边栏（应用主导航、多级菜单） | |
| `collapsible` | 折叠面板（FAQ、设置分组、手风琴） | |
| `accordion` | 手风琴（FAQ 列表、设置项分组展开） | |

## 6. 下拉与命令 (Dropdown / Command)

| 组件名 | 用途场景 | 状态 |
|---|---|---|
| `dropdown-menu` | 下拉菜单（操作菜单、用户菜单、上下文菜单） | [installed] |
| `context-menu` | 右键菜单（表格行操作、文本操作） | |
| `command` | 命令面板（搜索命令、快捷键唤起、VS Code 风格） | [installed] |
| `combobox` | 组合框（可搜索下拉、自动完成、带创建选项） | |

## 7. 高级交互 (Advanced)

| 组件名 | 用途场景 | 状态 |
|---|---|---|
| `carousel` | 轮播图（图片轮播、卡片轮播、横幅广告） | |
| `resizable` | 可拖拽调整大小（面板分割、表格列宽调整） | |
| `attachment` | 文件附件（文件上传、拖拽上传、文件列表） | |
| `item` | 列表项（带操作按钮的列表行、设置项） | |
| `field` | 表单字段（表单字段包装，含校验信息） | |
| `message` | 消息气泡（聊天界面、评论、通知详情） | |
| `message-scroller` | 消息滚动容器（聊天窗口、消息列表） | |
| `bubble` | 聊天气泡（AI 对话、即时通讯） | |
| `marker` | 标记点（地图标注、时间线节点） | |
| `questionnaire` | 问卷（多步骤表单、调查问卷） | |
| `direction` | 方向控制（RTL/LTR 布局切换） | |
| `spinner` | 加载旋转器（按钮加载、页面加载、局部加载） | |

---

## 安装统计

- registry 总计：63 个组件
- 已安装：23 个
- 未安装：40 个

## 安装注意事项

1. 安装后需检查 CSS 变量引用是否与项目 `tokens.css` 兼容
2. 安装后运行门禁：`pnpm typecheck && pnpm test && pnpm build`
3. 双主题验证：light/dark 下分别渲染确认颜色正确
4. 批量安装示例：`node .agents/skills/self-evolving/scripts/shadcn.mjs add avatar breadcrumb --force`
