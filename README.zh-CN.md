# OIDC 通用认证平台

当前版本：**v1.12**

一个自托管的、面向开发者的通用身份认证平台。在标准 OpenID Connect / OAuth 2.0 之上，引入「账号置信等级（Trust Level）」模型：用户在本平台通过邮箱注册，再绑定 GitHub、Gitee、GitLab、Discord、Google、Microsoft、Apple、Telegram、QQ、微信、手机号等第三方账号来提升自己的置信等级；其他业务系统只需要对接本平台一次，通过设置「准入置信等级 + 附加条件」即可获得低成本的风控与准入控制能力。

> 一句话定位：**用 OIDC 给你的业务系统装一个开箱即用的风控前置层。**

英文说明 / English version: [README.md](./README.md)。

## 设计目标

- **对开发者友好的标准接口**：完整实现 OIDC（Authorization Code、Refresh Token、ID Token）和 OAuth 2.0，业务系统按标准协议接入即可，无需理解平台内部模型。
- **置信等级集中管理**：用户在本平台一次绑定，所有接入系统共享同一份置信视图。
- **业务侧零风控成本**：业务系统只声明「我要求 Lv3 + 已绑定手机号 + GitHub 账龄 ≥ 30 天」，剩下的判断、拒绝、提示由平台完成。
- **滥用反馈闭环**：业务系统可以将滥用账号标记上报，平台据此降低该用户置信等级，并把其绑定账号列入风控名单，对其他接入方同步生效。
- **限制集中在本平台**：白名单、别名限制、邮箱域名限制、IP/地域风控等都在本平台配置，避免每个业务系统重复造轮子。

## 核心概念

### 置信等级（Trust Level）
每个用户拥有一个动态的置信等级（例如 Lv0 ~ Lv5），由「已绑定的第三方账号种类、绑定时长、是否手机/邮箱已验证、是否启用 MFA、风控记录」共同计算得出。后台可为每一级配置门槛规则，例如：

- **Lv1**：邮箱已验证
- **Lv2**：Lv1 + 绑定任意一个社交账号
- **Lv3**：Lv2 + 绑定手机号 + 启用 TOTP
- **Lv4**：Lv3 + GitHub 账龄 ≥ 90 天 或 绑定微信/Apple
- **Lv5**：Lv4 + 人工审核通过

### 准入策略（Access Rule）
业务系统在「我的应用」中声明：

- 最低置信等级
- 必须包含的绑定（如：必须绑定手机号）
- 附加条件（如：GitHub 账龄、邮箱域名、地域限制、别名黑/白名单）

用户登录时，平台先校验上述策略，未达标会引导用户去补充绑定，达标后才下发 ID Token。

### 风控反馈
业务系统可通过 API 上报滥用：`POST /api/v1/risk/report`。平台收到上报后会：

- 降低该用户置信等级
- 将其绑定的账号（如 GitHub 用户名、手机号 hash、设备指纹）写入风控库
- 其他接入系统在评估同一用户时自动感知

### 一键登录扩展
除标准 OIDC 外，还提供「绑定即登录」体验：用户在 A 系统绑定过 GitHub，下次在 B 系统点 GitHub 登录，平台会自动复用同一身份并按目标系统的准入策略放行/拒绝。

## 已支持的绑定渠道

| 类别 | 渠道 |
| ---- | ---- |
| 全球 | Google、GitHub、GitLab、Microsoft、Apple、Discord、Telegram |
| 中国 | Gitee、QQ、微信、Linux DO |
| 通用 | 邮箱、手机号（短信验证）、TOTP/MFA、Passkey |

> 每个渠道都是可插拔的，后台可独立启用/禁用并填写凭据，不必一次配齐。

## 主要功能模块

- **用户端账号中心**：邮箱注册/登录、密码找回、邮箱验证、第三方账号绑定与解绑、会话管理、授权应用列表、近期操作、安全等级查看、TOTP/MFA、Passkey 管理。
- **开发者门户**：自助创建应用、管理 redirect URI 与密钥、配置准入策略、管理授权用户、拉黑应用用户、提交滥用举报。
- **后台管理控制台**：用户管理、应用管理、授权用户管理、社交渠道配置、单渠道登录/注册开关、签名密钥轮换、审计日志、风控策略、风控名单、安全规则、系统设置、版本与更新检测、别名/白名单/邮箱域名限制。
- **OIDC / OAuth 服务端**：基于 [ory/fosite](https://github.com/ory/fosite) 实现，提供 `/.well-known/openid-configuration`、`/authorize`、`/token`、`/userinfo`、`/jwks.json` 等标准端点。
- **风控与安全**：登录失败次数锁定、Cloudflare Turnstile / hCaptcha 人机验证、平台级风控阻断、限流、请求审计、密码强度策略、Passkey、签名密钥定期轮换。

## 技术栈

- **后端**：Go 1.23+，[chi](https://github.com/go-chi/chi) 路由，[ory/fosite](https://github.com/ory/fosite) OIDC 引擎，[viper](https://github.com/spf13/viper) 配置，pgx / modernc.org/sqlite。
- **存储**：SQLite（默认，零依赖单文件部署）或 PostgreSQL + Redis（生产）。
- **前端**：Vue 3 + Vite + TypeScript + Pinia，单包同时承载用户中心 / 开发者门户 / 后台管理。
- **部署**：推荐直接拉取 GHCR Docker 镜像部署，也支持单二进制 + 前端 dist 的本地/手动部署方式。

## 路线图

- [x] 邮箱注册 / 登录 / 找回
- [x] OIDC / OAuth 2.0 标准端点
- [x] GitHub / Google / GitLab / Gitee / Linux DO / Microsoft / Discord / Apple / Telegram / QQ / 微信 / 手机号 绑定
- [x] 多级置信等级模型
- [x] 应用准入策略（最低等级 + 必须绑定 + 附加条件）
- [x] 别名 / 邮箱域名 / IP / 地域 限制
- [x] 滥用上报、管理员审核与共享风控库
- [x] 平台级风控策略与阻断控制
- [x] Cloudflare Turnstile / hCaptcha 人机验证
- [x] WebAuthn / Passkey 管理
- [x] 用户近期操作与后台审计追踪
- [x] 系统版本展示与 Release 更新检测
- [x] Docker 镜像工作流与 GHCR 镜像部署
- [ ] 多租户隔离
- [ ] SDK：Go / Node / Python 一键接入示例

部署、配置与开发命令请参考英文 [README.md](./README.md)。

## License

Private project，保留所有权利。
