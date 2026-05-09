# 微信云开发接入说明

当前小程序已经切换为微信云开发模式，不再依赖外部域名和自建 HTTP 后端。

## 1. 创建云环境

1. 用微信开发者工具打开 miniprogram 目录。
2. 点击顶部“云开发”。
3. 创建一个新的云环境，环境地域按你自己的账号选择。
4. 让当前项目绑定到这个云环境。

项目里已经使用 `wx.cloud.DYNAMIC_CURRENT_ENV`，不需要再改代码中的环境 ID。

## 2. 部署云函数

云函数目录已经配置为 `miniprogram/cloudfunctions`。

需要部署的函数：

- `gaokaoApi`

部署步骤：

1. 在开发者工具左侧找到 `cloudfunctions/gaokaoApi`。
2. 右键目录。
3. 先执行“在终端中打开并安装依赖”或等价的 npm 安装。
4. 再执行“上传并部署：云端安装依赖”。

## 3. 云数据库集合

云函数默认读取以下集合，集合名与 PostgreSQL 表名保持一致：

- `college`
- `college_program_group`
- `college_enrollment_plan`
- `college_major`
- `college_major_admission_stat`
- `province_score_line`
- `score_rank`

集合中的字段名也按现有后端表字段导入，例如：

- `college.id`
- `college_enrollment_plan.college_id`
- `college_enrollment_plan.program_group_id`
- `college_major_admission_stat.enrollment_plan_id`

云函数依赖这些数值关联字段做聚合，导入时不要丢掉原始 `id` 字段。

## 4. AI 报告配置

如果需要继续生成 DeepSeek 报告，在云函数环境变量里配置：

- `DEEPSEEK_API_KEY`
- `DEEPSEEK_BASE_URL`

如果不配置，报告页会自动使用本地模板回退，不会阻塞主流程。

## 5. 当前迁移结果

已完成：

- 小程序 `request` 已改为 `wx.cloud.callFunction`
- 首页已移除接口地址配置入口
- 项目已配置云函数根目录
- 推荐、查分、批次线、院校列表、院校详情、AI 报告统一走云函数

仍需要你完成：

1. 创建云环境。
2. 上传云函数。
3. 把数据库数据导入到上述云数据库集合。

完成这三步后，小程序就不再需要配置域名和自建后端。