# gaokao-ai

黑龙江高报小程序与 Go 后端一体仓库。

当前仓库包含三部分：

- `backend/`：Go 后端，提供推荐、院校库、位次、省控线、反馈、支付等接口。
- `miniprogram/`：微信小程序前端。
- `scripts/`：线上生产环境同步、构建、验证与发布脚本。

## 目录说明

```text
gaokao-ai/
  backend/                 Go API 服务
  miniprogram/             微信小程序
  miniprogram.__restore_backup__/
  backend.__restore_backup__/
  scripts/                 线上部署脚本
```

## 本地开发

### 后端

1. 复制环境变量示例：

   `cp backend/.env.example backend/.env`

2. 进入后端目录并启动：

   `cd backend && go run ./cmd/main.go`

默认监听地址为 `:8080`，可通过 `SERVER_ADDR` 覆盖。

### 小程序

使用微信开发者工具打开 `miniprogram/`。

当前默认后端地址已经指向生产反向代理：

- `http://82.156.54.232:80`

## 线上生产环境

当前生产环境约定如下：

- 服务器：`82.156.54.232`
- 登录用户：`ubuntu`
- 远端后端目录：`/home/ubuntu/gaokao-ai-backend/backend`
- 远端 Go：`/home/ubuntu/.local/go/bin/go`
- Nginx：`80/443 -> 127.0.0.1:8080`
- 临时验证端口：`18083`

### 脚本说明

- `scripts/sync_backend_82.sh`
  - 只做源码同步与远端构建
  - 产出远端 `gaokao-api.new`

- `scripts/deploy_backend_82.sh`
  - 做完整发布：同步、远端构建、18083 临时验证、切换到 8080、再做公网验证

- `scripts/watch_sync_backend_82.sh`
  - 监听本地 `backend/` 变化，自动调用同步脚本

### 使用前提

脚本不会在仓库里保存密码。执行前请显式传入：

```bash
export SSHPASS='你的服务器密码'
```

### 常用命令

只同步并在远端构建：

```bash
./scripts/sync_backend_82.sh
```

完整发布到生产：

```bash
./scripts/deploy_backend_82.sh
```

监听本地后端改动并自动同步：

```bash
./scripts/watch_sync_backend_82.sh
```

## 说明

- 仓库默认忽略本地 `.env`、二进制产物、缓存目录。
- 生产环境密钥、支付证书、私有配置不要提交到仓库。
- 备份目录当前保留在仓库中，用于恢复对照；如果后续确认不再需要，可单独清理。