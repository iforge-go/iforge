# iForge API

iForge 后端 API 服务，基于 Go + Fiber 框架构建，提供现代化的 Git 仓库托管平台全部后端能力。

## 技术栈

| 组件 | 技术 |
|------|------|
| 语言 | Go 1.26 |
| Web 框架 | Fiber v2 |
| ORM | GORM |
| 数据库 | SQLite（默认）/ MySQL / PostgreSQL |
| Git 操作 | go-git v5 |
| 认证 | JWT / OIDC / LDAP |
| SSH | 内置 SSH 服务器 |
| API 文档 | Swagger/OpenAPI |

## 项目结构

```
api/
├── cmd/server/
│   └── main.go                 # 应用入口
├── internal/
│   ├── handler/                # HTTP 请求处理层（44 个文件）
│   ├── service/                # 业务逻辑层（45 个文件）
│   ├── model/                  # 数据库模型（19 个文件）
│   ├── router/                 # 路由定义
│   ├── middleware/             # 中间件（认证、指标采集）
│   ├── container/              # 依赖注入容器
│   └── database/               # 数据库初始化与迁移
├── docs/                       # Swagger 文档
├── migrations/                 # 数据库迁移脚本
├── data/                       # 运行时数据目录
│   ├── iforge.db               # SQLite 数据库
│   ├── repositories/           # Git 裸仓库
│   ├── logs/                   # 应用日志
│   └── ssh/                    # SSH 主机密钥
├── go.mod
└── go.sum
```

## 快速启动

```bash
# 启动服务（默认 HTTP :8081, SSH :2022）
cd api/cmd/server
go run .
```

默认管理员账号：`root` / `root`

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `IFORGE_HOME` | `./data` | 数据目录 |
| `IFORGE_SSH_ENABLED` | `true` | 是否启用 SSH 服务 |
| `IFORGE_SSH_PORT` | `2022` | SSH 服务端口 |

HTTP 端口固定为 `8081`。

## API 路由

所有 API 路由位于 `/api/v1` 下，使用 JWT Bearer Token 认证。

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/signin` | 登录 |
| POST | `/register` | 注册 |
| POST | `/signout` | 登出 |
| GET | `/user` | 获取当前用户 |
| PATCH | `/user` | 更新当前用户 |

### 仓库管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/repos` | 仓库列表 |
| POST | `/repos` | 创建仓库 |
| GET | `/repos/:owner/:repo` | 仓库详情 |
| PATCH | `/repos/:owner/:repo` | 更新仓库 |
| DELETE | `/repos/:owner/:repo` | 删除仓库 |
| GET | `/repos/:owner/:repo/forks` | 仓库 Fork 列表 |
| POST | `/repos/:owner/:repo/forks` | Fork 仓库 |
| GET | `/repos/:owner/:repo/stargazers` | Star 用户列表 |
| GET | `/repos/:owner/:repo/watchers` | Watch 用户列表 |

### Git 操作

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/repos/:owner/:repo/files` | 文件列表 |
| GET | `/repos/:owner/:repo/file` | 文件内容 |
| POST | `/repos/:owner/:repo/files` | 创建文件 |
| PUT | `/repos/:owner/:repo/files` | 更新文件 |
| DELETE | `/repos/:owner/:repo/files` | 删除文件 |
| GET | `/repos/:owner/:repo/raw/:ref/*` | 原始文件下载 |
| GET | `/repos/:owner/:repo/commits` | 提交历史 |
| GET | `/repos/:owner/:repo/branches` | 分支列表 |
| POST | `/repos/:owner/:repo/branches` | 创建分支 |
| GET | `/repos/:owner/:repo/tags` | 标签列表 |
| GET | `/repos/:owner/:repo/archive/*` | 归档下载 |
| GET | `/repos/:owner/:repo/search` | 代码搜索 |

### Git HTTP 协议

| 路径 | 说明 |
|------|------|
| `/:owner/:repo.git/*` | Smart HTTP Git 协议 |
| `/:owner/:repo/info/refs` | 引用发现 |
| `/:owner/:repo/git-upload-pack` | 拉取操作 |
| `/:owner/:repo/git-receive-pack` | 推送操作 |

### Issue 跟踪

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST | `/repos/:owner/:repo/issues` | Issue 列表/创建 |
| GET/PATCH/DELETE | `/repos/:owner/:repo/issues/:id` | Issue 详情/更新/删除 |
| GET/POST | `/repos/:owner/:repo/issues/:id/comments` | 评论 |
| GET/POST | `/repos/:owner/:repo/issues/:id/labels` | 标签关联 |

### Merge Request

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST | `/repos/:owner/:repo/pulls` | MR 列表/创建 |
| POST | `/repos/:owner/:repo/pulls/:id/merge` | 合并 |
| GET/POST | `/repos/:owner/:repo/pulls/:id/reviews` | 代码评审 |
| GET | `/repos/:owner/:repo/pulls/:id/commits` | 提交列表 |
| GET | `/repos/:owner/:repo/pulls/:id/files` | 文件变更 |

### 项目管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST | `/repos/:owner/:repo/projects` | 项目看板 |
| GET/POST | `/projects/:id/columns` | 看板列 |
| GET/POST | `/projects/columns/:id/cards` | 看板卡片 |

### Wiki

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST | `/repos/:owner/:repo/wiki` | Wiki 页面 |
| GET/PATCH/DELETE | `/repos/:owner/:repo/wiki/:slug` | Wiki 操作 |

### 发布管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST | `/repos/:owner/:repo/releases` | 发布列表/创建 |
| GET/PATCH/DELETE | `/repos/:owner/:repo/releases/:tag` | 发布操作 |
| POST | `/repos/:owner/:repo/releases/:tag/assets` | 上传附件 |

### 用户管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/users/:username` | 用户信息 |
| GET | `/users/:username/repos` | 用户仓库 |
| GET/POST/DELETE | `/user/keys` | SSH 密钥管理 |
| GET/POST/DELETE | `/user/gpg_keys` | GPG 密钥管理 |
| GET/POST/DELETE | `/user/tokens` | Access Token 管理 |
| GET/POST/DELETE | `/user/emails` | 邮箱管理 |

### 管理后台

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST/DELETE | `/admin/users` | 用户管理 |
| GET | `/admin/repos` | 仓库管理 |
| GET/PATCH | `/admin/settings` | 系统设置 |
| GET | `/admin/system-info` | 系统信息 |
| GET | `/admin/plugins` | 插件管理 |
| GET | `/admin/database` | 数据库查看 |

### 系统设置

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/settings/ssh` | SSH 公开配置 |
| GET/PATCH | `/settings` | 用户偏好设置 |

## 核心模型

### 用户与权限
- **Account** — 用户/组织账户，支持管理员标志
- **OrganizationMember** — 组织成员关系
- **Collaborator** — 仓库协作者及权限角色
- **SSHKey / GPGKey** — 加密密钥
- **AccessToken** — 个人访问令牌

### 仓库
- **Repository** — Git 仓库，支持公开/私有、Fork、镜像
- **ProtectedBranch** — 分支保护规则
- **DeployKey** — 部署密钥
- **RepositoryMirror** — 仓库镜像配置
- **RepositoryStar / RepositoryWatch** — 用户互动

### Issue 与 MR
- **Issue** — Issue 和 MR 统一模型
- **IssueComment** — 评论
- **Label / Milestone / Priority** — 分类与排期
- **CustomField** — 自定义字段
- **MergeRequest** — 合并请求，支持 Draft 模式
- **Review / ReviewComment** — 代码评审

### 协作
- **WikiPage** — Wiki 文档
- **Project / ProjectColumn / ProjectCard** — 看板项目管理
- **Notification** — 通知系统
- **Activity** — 动态信息流
- **Webhook** — 外部集成

### 发布
- **ReleaseTag** — 版本发布
- **ReleaseAsset** — 发布附件

### 扩展
- **Plugin / PluginEvent / PluginHook** — 插件系统
- **SystemSetting** — 全局配置
- **CommitStatus** — CI/CD 状态集成

## 架构设计

```
HTTP Request
    │
    ▼
┌──────────┐     ┌──────────────┐     ┌──────────┐
│  Router   │────▶│   Handler    │────▶│ Service  │
│ (Fiber)   │     │  (REST API)  │     │(Business) │
└──────────┘     └──────────────┘     └──────────┘
                                            │
                                     ┌──────┴──────┐
                                     ▼             ▼
                              ┌──────────┐  ┌──────────┐
                              │  Model   │  │ Git Repo │
                              │ (GORM)   │  │ (go-git) │
                              └──────────┘  └──────────┘
```

- **Container** — 依赖注入容器，懒加载所有 Service 和 Handler
- **Middleware** — JWT 认证、请求指标采集
- **Router** — 按功能模块分组注册路由（40+ 路由组）
