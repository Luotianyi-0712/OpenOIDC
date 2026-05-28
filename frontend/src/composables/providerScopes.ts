// OAuth2 Provider Scopes 智能联想数据
// 支持中英文搜索

export interface ScopeOption {
  value: string
  label: string
  description: string
  keywords: string[] // 中文关键词用于搜索
}

export const providerScopes: Record<string, ScopeOption[]> = {
  github: [
    // 用户权限
    { value: 'user', label: 'user', description: '读写用户资料（包含 user:email 和 user:follow）', keywords: ['读写', '用户', '资料', '完全'] },
    { value: 'read:user', label: 'read:user', description: '读取用户基本信息', keywords: ['读取', '用户', '基本', '信息'] },
    { value: 'user:email', label: 'user:email', description: '读取用户邮箱地址', keywords: ['读取', '邮箱', '邮件', '地址'] },
    { value: 'user:follow', label: 'user:follow', description: '关注和取消关注用户', keywords: ['关注', '取消', '用户'] },
    // 仓库权限
    { value: 'repo', label: 'repo', description: '完全访问公开和私有仓库（包含代码、提交状态、协作者、部署、webhook）', keywords: ['仓库', '完全', '访问', '公开', '私有', '读写'] },
    { value: 'repo:status', label: 'repo:status', description: '读写提交状态', keywords: ['提交', '状态', '读写'] },
    { value: 'repo_deployment', label: 'repo_deployment', description: '访问部署状态', keywords: ['部署', '状态', '访问'] },
    { value: 'public_repo', label: 'public_repo', description: '访问公开仓库（包含 star）', keywords: ['公开', '仓库', '访问', 'star'] },
    { value: 'repo:invite', label: 'repo:invite', description: '接受/拒绝仓库协作邀请', keywords: ['仓库', '邀请', '协作'] },
    { value: 'security_events', label: 'security_events', description: '读写代码扫描安全事件', keywords: ['安全', '事件', '读写', '代码扫描'] },
    // 仓库 Webhook
    { value: 'admin:repo_hook', label: 'admin:repo_hook', description: '完全管理仓库 webhook（读、写、ping、删除）', keywords: ['仓库', '钩子', 'webhook', '完全', '管理'] },
    { value: 'write:repo_hook', label: 'write:repo_hook', description: '读写和 ping 仓库 webhook', keywords: ['仓库', '钩子', 'webhook', '读写'] },
    { value: 'read:repo_hook', label: 'read:repo_hook', description: '读取和 ping 仓库 webhook', keywords: ['仓库', '钩子', 'webhook', '读取'] },
    // 组织权限
    { value: 'admin:org', label: 'admin:org', description: '完全管理组织（团队、项目、成员）', keywords: ['组织', '管理', '完全', '团队'] },
    { value: 'write:org', label: 'write:org', description: '读写组织成员和项目', keywords: ['组织', '成员', '项目', '读写'] },
    { value: 'read:org', label: 'read:org', description: '只读组织成员、项目和团队', keywords: ['组织', '成员', '只读'] },
    // 组织 Webhook
    { value: 'admin:org_hook', label: 'admin:org_hook', description: '完全管理组织 webhook', keywords: ['组织', '钩子', 'webhook', '管理'] },
    // 公钥管理
    { value: 'admin:public_key', label: 'admin:public_key', description: '完全管理公钥', keywords: ['公钥', '管理', '完全'] },
    { value: 'write:public_key', label: 'write:public_key', description: '创建、列出和查看公钥', keywords: ['公钥', '创建', '列出', '查看'] },
    { value: 'read:public_key', label: 'read:public_key', description: '列出和查看公钥', keywords: ['公钥', '列出', '查看'] },
    // GPG 密钥
    { value: 'admin:gpg_key', label: 'admin:gpg_key', description: '完全管理 GPG 密钥', keywords: ['gpg', '密钥', '管理', '完全'] },
    { value: 'write:gpg_key', label: 'write:gpg_key', description: '创建、列出和查看 GPG 密钥', keywords: ['gpg', '密钥', '创建', '列出', '查看'] },
    { value: 'read:gpg_key', label: 'read:gpg_key', description: '列出和查看 GPG 密钥', keywords: ['gpg', '密钥', '列出', '查看'] },
    // 其他
    { value: 'gist', label: 'gist', description: '写入 gists 代码片段', keywords: ['gist', '代码片段', '写入'] },
    { value: 'notifications', label: 'notifications', description: '访问通知（读取、标记已读、watch/unwatch）', keywords: ['通知', '访问', '读取'] },
    { value: 'delete_repo', label: 'delete_repo', description: '删除可管理的仓库', keywords: ['删除', '仓库'] },
    // 项目
    { value: 'project', label: 'project', description: '读写用户和组织项目', keywords: ['项目', '读写', 'project'] },
    { value: 'read:project', label: 'read:project', description: '只读用户和组织项目', keywords: ['项目', '只读', 'project'] },
    // Packages
    { value: 'write:packages', label: 'write:packages', description: '上传包到 GitHub Packages', keywords: ['包', '上传', 'packages'] },
    { value: 'read:packages', label: 'read:packages', description: '下载包从 GitHub Packages', keywords: ['包', '下载', 'packages'] },
    { value: 'delete:packages', label: 'delete:packages', description: '删除包从 GitHub Packages', keywords: ['包', '删除', 'packages'] },
    // Codespaces
    { value: 'codespace', label: 'codespace', description: '创建和管理 Codespaces', keywords: ['codespace', '创建', '管理'] },
    // Workflow
    { value: 'workflow', label: 'workflow', description: '添加和更新 GitHub Actions 工作流文件', keywords: ['工作流', 'workflow', 'actions', '更新'] },
    // 审计日志
    { value: 'read:audit_log', label: 'read:audit_log', description: '读取审计日志数据', keywords: ['审计', '日志', '读取'] },
  ],

  discord: [
    // 用户信息
    { value: 'identify', label: 'identify', description: '读取用户基本信息（ID、用户名、头像等，不含邮箱）', keywords: ['读取', '用户', '基本', '信息', '用户名', '头像'] },
    { value: 'email', label: 'email', description: '读取用户邮箱地址', keywords: ['读取', '邮箱', '邮件', '地址'] },
    { value: 'identify.premium', label: 'identify.premium', description: '读取用户 Nitro 订阅类型（需审批）', keywords: ['读取', 'nitro', '订阅', '会员'] },
    { value: 'connections', label: 'connections', description: '读取用户连接的第三方账号（Twitch、Steam 等）', keywords: ['读取', '连接', '第三方', '账号'] },
    // 服务器相关
    { value: 'guilds', label: 'guilds', description: '读取用户加入的服务器列表', keywords: ['读取', '服务器', '列表', '公会'] },
    { value: 'guilds.join', label: 'guilds.join', description: '将用户加入服务器', keywords: ['加入', '服务器', '公会'] },
    { value: 'guilds.members.read', label: 'guilds.members.read', description: '读取用户在服务器中的成员信息', keywords: ['读取', '服务器', '成员', '信息'] },
    { value: 'bot', label: 'bot', description: '将机器人添加到服务器', keywords: ['机器人', '添加', '服务器', 'bot'] },
    // 应用命令
    { value: 'applications.commands', label: 'applications.commands', description: '向服务器添加斜杠命令（bot scope 默认包含）', keywords: ['应用', '命令', '创建', '斜杠'] },
    { value: 'applications.commands.update', label: 'applications.commands.update', description: '使用 Bearer token 更新命令（仅客户端凭证）', keywords: ['应用', '命令', '更新'] },
    { value: 'applications.commands.permissions.update', label: 'applications.commands.permissions.update', description: '更新服务器中的命令权限', keywords: ['命令', '权限', '更新'] },
    // 应用相关
    { value: 'applications.builds.read', label: 'applications.builds.read', description: '读取应用构建数据', keywords: ['应用', '构建', '读取'] },
    { value: 'applications.builds.upload', label: 'applications.builds.upload', description: '上传/更新应用构建（需审批）', keywords: ['应用', '构建', '上传'] },
    { value: 'applications.entitlements', label: 'applications.entitlements', description: '读取应用权益', keywords: ['应用', '权益', '读取'] },
    { value: 'applications.store.update', label: 'applications.store.update', description: '读写应用商店数据（SKU、成就等）', keywords: ['应用', '商店', '更新'] },
    // 活动
    { value: 'activities.read', label: 'activities.read', description: '读取"正在玩/最近玩过"列表（暂不可用）', keywords: ['活动', '读取', '正在玩'] },
    { value: 'activities.write', label: 'activities.write', description: '更新用户活动状态（暂不可用，GameSDK 不需要）', keywords: ['活动', '更新', '状态'] },
    // 消息和频道
    { value: 'messages.read', label: 'messages.read', description: '读取所有频道消息（本地 RPC，需审批）', keywords: ['读取', '消息', '历史'] },
    { value: 'dm_channels.read', label: 'dm_channels.read', description: '查看用户的私信和群组私信（需审批）', keywords: ['私信', '查看', '群组'] },
    { value: 'gdm.join', label: 'gdm.join', description: '将用户加入群组私信', keywords: ['加入', '群组', '私信'] },
    // 社交
    { value: 'relationships.read', label: 'relationships.read', description: '读取好友列表、待处理请求和屏蔽用户（Social SDK）', keywords: ['好友', '读取', '社交'] },
    // 角色连接
    { value: 'role_connections.write', label: 'role_connections.write', description: '更新用户的角色连接和元数据', keywords: ['写入', '角色', '连接', '元数据'] },
    // Webhook
    { value: 'webhook.incoming', label: 'webhook.incoming', description: '创建 Webhook（OAuth 响应中返回）', keywords: ['创建', 'webhook', '钩子'] },
    // RPC（本地客户端控制，需审批）
    { value: 'rpc', label: 'rpc', description: '控制用户本地 Discord 客户端（需审批）', keywords: ['rpc', '控制', '客户端'] },
    { value: 'rpc.activities.write', label: 'rpc.activities.write', description: '更新用户活动状态（需审批）', keywords: ['更新', '活动', '状态', 'rpc'] },
    { value: 'rpc.notifications.read', label: 'rpc.notifications.read', description: '接收推送通知（需审批）', keywords: ['接收', '通知', 'rpc'] },
    { value: 'rpc.voice.read', label: 'rpc.voice.read', description: '读取语音设置和事件（需审批）', keywords: ['读取', '语音', '状态', 'rpc'] },
    { value: 'rpc.voice.write', label: 'rpc.voice.write', description: '更新语音设置（需审批）', keywords: ['控制', '语音', '状态', 'rpc'] },
    // 语音
    { value: 'voice', label: 'voice', description: '代表用户连接语音并查看成员（需审批）', keywords: ['语音', '连接', '查看'] },
  ],

  google: [
    { value: 'openid', label: 'openid', description: 'OpenID Connect 身份验证', keywords: ['openid', '身份', '验证', '认证'] },
    { value: 'profile', label: 'profile', description: '查看用户基本资料信息', keywords: ['查看', '用户', '基本', '资料', '信息'] },
    { value: 'email', label: 'email', description: '查看用户邮箱地址', keywords: ['查看', '邮箱', '邮件', '地址'] },
    { value: 'https://www.googleapis.com/auth/userinfo.profile', label: 'userinfo.profile', description: '查看用户个人信息', keywords: ['查看', '用户', '个人', '信息'] },
    { value: 'https://www.googleapis.com/auth/userinfo.email', label: 'userinfo.email', description: '查看用户邮箱', keywords: ['查看', '用户', '邮箱'] },
    { value: 'https://www.googleapis.com/auth/drive', label: 'drive', description: '查看和管理 Google Drive 文件', keywords: ['查看', '管理', 'drive', '云盘', '文件'] },
    { value: 'https://www.googleapis.com/auth/drive.readonly', label: 'drive.readonly', description: '只读访问 Google Drive', keywords: ['只读', '访问', 'drive', '云盘'] },
    { value: 'https://www.googleapis.com/auth/drive.file', label: 'drive.file', description: '访问应用创建的文件', keywords: ['访问', '应用', '创建', '文件', 'drive'] },
    { value: 'https://www.googleapis.com/auth/calendar', label: 'calendar', description: '查看和编辑日历', keywords: ['查看', '编辑', '日历'] },
    { value: 'https://www.googleapis.com/auth/calendar.readonly', label: 'calendar.readonly', description: '只读访问日历', keywords: ['只读', '访问', '日历'] },
    { value: 'https://www.googleapis.com/auth/gmail.readonly', label: 'gmail.readonly', description: '只读访问 Gmail', keywords: ['只读', '访问', 'gmail', '邮件'] },
    { value: 'https://www.googleapis.com/auth/gmail.send', label: 'gmail.send', description: '发送邮件', keywords: ['发送', '邮件', 'gmail'] },
    { value: 'https://www.googleapis.com/auth/contacts.readonly', label: 'contacts.readonly', description: '只读访问联系人', keywords: ['只读', '访问', '联系人'] },
    { value: 'https://www.googleapis.com/auth/youtube.readonly', label: 'youtube.readonly', description: '只读访问 YouTube', keywords: ['只读', '访问', 'youtube', '视频'] },
  ],

  microsoft: [
    { value: 'openid', label: 'openid', description: 'OpenID Connect 身份验证', keywords: ['openid', '身份', '验证', '认证'] },
    { value: 'profile', label: 'profile', description: '查看用户基本资料', keywords: ['查看', '用户', '基本', '资料'] },
    { value: 'email', label: 'email', description: '查看用户邮箱地址', keywords: ['查看', '邮箱', '邮件', '地址'] },
    { value: 'User.Read', label: 'User.Read', description: '读取用户个人资料', keywords: ['读取', '用户', '个人', '资料'] },
    { value: 'User.ReadWrite', label: 'User.ReadWrite', description: '读写用户个人资料', keywords: ['读写', '用户', '个人', '资料'] },
    { value: 'User.ReadBasic.All', label: 'User.ReadBasic.All', description: '读取所有用户基本资料', keywords: ['读取', '所有', '用户', '基本', '资料'] },
    { value: 'Mail.Read', label: 'Mail.Read', description: '读取用户邮件', keywords: ['读取', '用户', '邮件'] },
    { value: 'Mail.ReadWrite', label: 'Mail.ReadWrite', description: '读写用户邮件', keywords: ['读写', '用户', '邮件'] },
    { value: 'Mail.Send', label: 'Mail.Send', description: '以用户身份发送邮件', keywords: ['发送', '邮件', '用户'] },
    { value: 'Calendars.Read', label: 'Calendars.Read', description: '读取用户日历', keywords: ['读取', '用户', '日历'] },
    { value: 'Calendars.ReadWrite', label: 'Calendars.ReadWrite', description: '读写用户日历', keywords: ['读写', '用户', '日历'] },
    { value: 'Contacts.Read', label: 'Contacts.Read', description: '读取用户联系人', keywords: ['读取', '用户', '联系人'] },
    { value: 'Contacts.ReadWrite', label: 'Contacts.ReadWrite', description: '读写用户联系人', keywords: ['读写', '用户', '联系人'] },
    { value: 'Files.Read', label: 'Files.Read', description: '读取用户文件', keywords: ['读取', '用户', '文件', 'onedrive'] },
    { value: 'Files.ReadWrite', label: 'Files.ReadWrite', description: '读写用户文件', keywords: ['读写', '用户', '文件', 'onedrive'] },
    { value: 'offline_access', label: 'offline_access', description: '离线访问（刷新令牌）', keywords: ['离线', '访问', '刷新', '令牌'] },
  ],

  gitlab: [
    // 用户信息
    { value: 'read_user', label: 'read_user', description: '只读访问用户信息（/user 端点、用户名、邮箱、全名）', keywords: ['只读', '访问', '用户', '信息'] },
    // API 访问
    { value: 'api', label: 'api', description: '完全读写访问 API（包含所有组、项目、容器镜像、依赖代理、包）', keywords: ['完全', '访问', 'api', '读写'] },
    { value: 'read_api', label: 'read_api', description: '只读访问 API（包含所有组、项目、容器镜像、包）', keywords: ['只读', '访问', 'api'] },
    // 仓库
    { value: 'read_repository', label: 'read_repository', description: '只读访问私有项目仓库（Git-over-HTTP 或 Repository Files API）', keywords: ['只读', '访问', '仓库'] },
    { value: 'write_repository', label: 'write_repository', description: '读写私有项目仓库（Git-over-HTTP，不含 API）', keywords: ['读写', '仓库'] },
    // 容器镜像仓库
    { value: 'read_registry', label: 'read_registry', description: '只读访问私有项目容器镜像', keywords: ['只读', '访问', '容器', '镜像', '仓库', 'registry'] },
    { value: 'write_registry', label: 'write_registry', description: '读写私有项目容器镜像', keywords: ['读写', '容器', '镜像', '仓库', 'registry'] },
    // Runner
    { value: 'create_runner', label: 'create_runner', description: '创建 Runner', keywords: ['创建', 'runner', '运行器'] },
    { value: 'manage_runner', label: 'manage_runner', description: '管理 Runner', keywords: ['管理', 'runner', '运行器'] },
    // Kubernetes
    { value: 'k8s_proxy', label: 'k8s_proxy', description: '使用 Kubernetes Agent 执行 API 调用', keywords: ['kubernetes', 'k8s', '代理'] },
    // 管理员
    { value: 'sudo', label: 'sudo', description: '以任意用户身份执行 API 操作（需管理员权限）', keywords: ['管理员', '执行', 'api', '操作', 'sudo'] },
    { value: 'admin_mode', label: 'admin_mode', description: '启用管理员模式', keywords: ['启用', '管理员', '模式'] },
    // OpenID Connect
    { value: 'openid', label: 'openid', description: 'OpenID Connect 身份验证（包含用户资料和组成员）', keywords: ['openid', '身份', '验证', '认证'] },
    { value: 'profile', label: 'profile', description: '只读访问用户资料（OpenID Connect）', keywords: ['访问', '用户', '资料'] },
    { value: 'email', label: 'email', description: '只读访问用户邮箱（OpenID Connect）', keywords: ['访问', '用户', '邮箱'] },
  ],

  gitee: [
    { value: 'user_info', label: 'user_info', description: '访问用户基本信息', keywords: ['访问', '用户', '基本', '信息'] },
    { value: 'projects', label: 'projects', description: '访问仓库信息', keywords: ['访问', '仓库', '信息', '项目'] },
    { value: 'pull_requests', label: 'pull_requests', description: '访问 Pull Request', keywords: ['访问', 'pr', 'pull', 'request', '合并请求'] },
    { value: 'issues', label: 'issues', description: '访问 Issues', keywords: ['访问', 'issue', '问题'] },
    { value: 'notes', label: 'notes', description: '访问评论', keywords: ['访问', '评论', '备注'] },
    { value: 'keys', label: 'keys', description: '访问公钥', keywords: ['访问', '公钥', '密钥'] },
    { value: 'hook', label: 'hook', description: '访问 Webhook', keywords: ['访问', 'webhook', '钩子'] },
    { value: 'groups', label: 'groups', description: '访问组织信息', keywords: ['访问', '组织', '信息', '团队'] },
    { value: 'gists', label: 'gists', description: '访问代码片段', keywords: ['访问', '代码', '片段', 'gist'] },
    { value: 'enterprises', label: 'enterprises', description: '访问企业信息', keywords: ['访问', '企业', '信息'] },
    { value: 'emails', label: 'emails', description: '访问邮箱列表', keywords: ['访问', '邮箱', '列表'] },
  ],

  linuxdo: [
    { value: 'user', label: 'user', description: '访问用户基本信息', keywords: ['访问', '用户', '基本', '信息'] },
    { value: 'email', label: 'email', description: '访问用户邮箱', keywords: ['访问', '用户', '邮箱'] },
    { value: 'profile', label: 'profile', description: '访问用户资料', keywords: ['访问', '用户', '资料'] },
  ],

  qq: [
    { value: 'get_user_info', label: 'get_user_info', description: '获取登录用户的昵称、头像、性别', keywords: ['获取', '用户', '信息', '昵称', '头像'] },
    { value: 'get_vip_info', label: 'get_vip_info', description: '获取 QQ 会员基本信息', keywords: ['获取', '会员', '信息', 'vip'] },
    { value: 'get_vip_rich_info', label: 'get_vip_rich_info', description: '获取 QQ 会员高级信息', keywords: ['获取', '会员', '详细', '信息', 'vip'] },
    { value: 'list_album', label: 'list_album', description: '获取用户 QQ 空间相册列表', keywords: ['获取', '相册', '列表', '空间'] },
    { value: 'upload_pic', label: 'upload_pic', description: '上传照片到 QQ 空间相册', keywords: ['上传', '图片', '照片', '相册'] },
    { value: 'add_album', label: 'add_album', description: '在 QQ 空间创建新相册', keywords: ['创建', '相册', '空间'] },
    { value: 'list_photo', label: 'list_photo', description: '获取 QQ 空间相册中的照片列表', keywords: ['获取', '照片', '列表', '相册'] },
    { value: 'get_tenpay_addr', label: 'get_tenpay_addr', description: '获取财付通用户的收货地址', keywords: ['获取', '财付通', '地址', '收货'] },
  ],

  apple: [
    { value: 'name', label: 'name', description: '访问用户姓名', keywords: ['访问', '用户', '姓名', '名字'] },
    { value: 'email', label: 'email', description: '访问用户邮箱', keywords: ['访问', '用户', '邮箱'] },
  ],
}

/**
 * 搜索 scopes，支持中英文
 */
export function searchScopes(provider: string, query: string): ScopeOption[] {
  const scopes = providerScopes[provider] || []
  if (!query || !query.trim()) {
    return scopes
  }

  const q = query.toLowerCase().trim()
  return scopes.filter(scope => {
    // 匹配 value
    if (scope.value.toLowerCase().includes(q)) return true
    // 匹配 label
    if (scope.label.toLowerCase().includes(q)) return true
    // 匹配 description
    if (scope.description.includes(q)) return true
    // 匹配中文关键词
    return scope.keywords.some(keyword => keyword.includes(q))
  })
}

/**
 * 获取 provider 的默认 scopes
 */
export function getDefaultScopes(provider: string): string[] {
  const defaults: Record<string, string[]> = {
    github: ['read:user', 'user:email'],
    discord: ['identify', 'email'],
    google: ['openid', 'profile', 'email'],
    microsoft: ['openid', 'profile', 'email', 'User.Read', 'offline_access'],
    gitlab: ['read_user'],
    gitee: ['user_info'],
    linuxdo: ['user'],
    qq: ['get_user_info'],
    apple: ['name', 'email'],
  }
  return defaults[provider] || []
}
