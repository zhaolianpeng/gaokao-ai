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

### 真机发布 / 微信开发者工具上传

1. 用微信开发者工具打开 `miniprogram/`，确认 AppID 使用项目内现有配置。
2. 在“详情”里确认本地设置与项目配置一致，尤其是：
  - 不要误切到其他云环境
  - 确认请求地址仍指向 `http://82.156.54.232:80`
3. 先做一次真机预览：
  - 点击“预览”
  - 用测试微信扫码
  - 重点验证首页、院校库分页、排序切换、推荐页、报告页、关于我们里的后端健康检查
4. 真机验证通过后再上传：
  - 点击“上传”
  - 填写版本号和更新说明
  - 在微信公众平台提交审核或直接发布
5. 如果线上后端刚更新，建议上传前先手动访问以下地址确认服务正常：
  - `http://82.156.54.232:80/healthz`
  - `http://82.156.54.232:80/api/colleges?province=黑龙江&subject=历史&year=2025&page=1&limit=3`

### 真机回归清单

上传前至少按下面顺序在真机过一遍：

1. 分页
  - 进入院校库
  - 触底后确认会继续加载下一页，而不是直接提示“已经到底了”
  - 顶部“已加载第 N 页”提示要跟随变化
2. 排序
  - 在院校库切换“按学校层次排序”和“按录取位次排序”
  - 确认两种排序的首屏结果明显不同
3. 推荐
  - 首页输入分数、位次、意向专业
  - 生成推荐后确认冲、稳、保三组都有结果或有合理空态提示
4. 报告
  - 在推荐页触发 AI 报告
  - 确认报告能正常生成、打开、复制，并能继续跳转到家长沟通卡片
5. 支付
  - 进入 VIP 中心
  - 确认登录态正确、商品展示正常、支付入口可触发
  - 正式发布前至少做一次完整支付链路联调或沙箱验证
6. 健康检查
  - 在关于我们页面执行 `/healthz` 检测
  - 确认当前后端地址仍是生产地址，网络诊断页没有新的异常记录

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

- `scripts/validate_backend_82.sh`
  - 只做候选版本验证
  - 在 `18083` 拉起 `gaokao-api.new`
  - 校验 `healthz` 与示例院校库接口

- `scripts/deploy_backend_82.sh`
  - 只做正式发布
  - 把已经验证过的 `gaokao-api.new` 切换到 `8080`
  - 再做公网健康检查

- `scripts/rollback_backend_82.sh`
  - 只做生产回滚
  - 自动选择最近一个 `gaokao-api.bak.*` 备份
  - 回滚后重启 `8080` 并校验公网接口

- `scripts/watch_sync_backend_82.sh`
  - 监听本地 `backend/` 变化，自动调用同步脚本

### 使用前提

脚本不会在仓库里保存密码。执行前请显式传入：

```bash
export SSHPASS='你的服务器密码'
```

### 常用命令

第一步，只同步并在远端构建候选版本：

```bash
./scripts/sync_backend_82.sh
```

第二步，只验证候选版本：

```bash
./scripts/validate_backend_82.sh
```

第三步，正式切换生产版本：

```bash
./scripts/deploy_backend_82.sh
```

如果正式发布后需要快速回滚：

```bash
./scripts/rollback_backend_82.sh
```

监听本地后端改动并自动同步：

```bash
./scripts/watch_sync_backend_82.sh
```

推荐发布顺序：

```bash
export SSHPASS='你的服务器密码'
./scripts/sync_backend_82.sh
./scripts/validate_backend_82.sh
./scripts/deploy_backend_82.sh
```

如果上线后发现异常，推荐回滚命令：

```bash
export SSHPASS='你的服务器密码'
./scripts/rollback_backend_82.sh
```

## 说明

- 仓库默认忽略本地 `.env`、二进制产物、缓存目录。
- 生产环境密钥、支付证书、私有配置不要提交到仓库。
- 备份目录当前保留在仓库中，用于恢复对照；如果后续确认不再需要，可单独清理。