# 微信云开发索引清单

这份清单按当前云函数 gaokaoApi 的实际查询路径整理，适用于微信云开发数据库控制台中的“索引管理”。

## 使用说明

- 这不是 PostgreSQL SQL，不能在“高级操作”窗口里执行。
- 应在每个集合的“索引管理”里手工新建索引。
- 复合索引字段顺序不要随意改，默认按下面顺序创建。
- 没有写“降序”的字段，统一用升序。
- 云开发没有 PostgreSQL 的 pg_trgm 模糊索引，院校名/专业名“包含关键词”无法 1:1 迁移。
- 除非下面明确写“唯一索引”，否则都按“非唯一索引”创建。

## 唯一性选择原则

- 唯一索引：只用于业务上天然唯一、且字段组合完整的键，适合防止重复导入。
- 非唯一索引：只用于查询加速，不承担数据去重职责，适合列表、详情、推荐这类高频读路径。
- 如果你的云开发集合里已经存在重复数据，不要直接把某个索引改成唯一，否则会创建失败。

## 需要创建的索引

### 1. college

用途：院校详情按 id 查询；院校列表结果集回表。

索引 1：college_id_lookup
- 类型：建议唯一索引
- id 升序

说明：
- 命中 /api/colleges/:id
- 命中 fetchByIds(COLLECTIONS.COLLEGE, 'id', ...)
- 这里的 id 对应原 PostgreSQL 主键，业务上应当唯一

### 2. college_program_group

用途：院校列表、推荐、院校详情页的核心集合。

索引 1：program_group_list_main
- 类型：建议非唯一索引
- province 升序
- year 升序
- subject 升序
- group_min_rank 升序
- group_min_score 降序

说明：
- 命中 /api/colleges
- 命中 /api/recommend
- 这两个接口都会先按 province + year + subject 过滤，再按 group_min_rank / group_min_score 排序拉取

索引 2：program_group_detail_main
- 类型：建议非唯一索引
- college_id 升序
- province 升序
- year 升序
- subject 升序
- group_min_rank 升序
- group_code 升序

说明：
- 命中 /api/colleges/:id
- 详情页会按 college_id + province + year + subject 过滤，并按 min_rank、group_code 排序

### 3. college_enrollment_plan

用途：概览统计、院校列表关键词补充匹配、院校详情、推荐结果专业聚合。

索引 1：enrollment_plan_overview_main
- 类型：建议非唯一索引
- province 升序
- year 升序
- subject 升序

说明：
- 命中 /api/dashboard/overview
- 命中不带 college_id 的基础拉取

索引 2：enrollment_plan_detail_main
- 类型：建议非唯一索引
- college_id 升序
- province 升序
- year 升序
- subject 升序
- program_group_id 升序

说明：
- 命中 /api/colleges/:id
- 详情页按 college_id + province + year + subject 查专业计划

索引 3：enrollment_plan_group_join_main
- 类型：建议非唯一索引
- program_group_id 升序
- province 升序
- year 升序
- subject 升序

说明：
- 命中 /api/colleges 中按 program_group_id 批量回查专业
- 命中 /api/recommend 中按 candidate group 批量回查专业

可选索引 4：enrollment_plan_college_only
- 类型：建议非唯一索引
- college_id 升序

说明：
- 如果详情页数据量继续变大，可补这个单列索引作为兜底

### 4. college_major

用途：院校详情页补充专业强项、硕博点信息。

索引 1：college_major_by_college
- 类型：建议非唯一索引
- college_id 升序

说明：
- 命中 /api/colleges/:id
- 当前查询只按 college_id 过滤

### 5. college_major_admission_stat

用途：概览统计、院校详情页专业历年录取数据。

索引 1：major_stat_by_plan
- 类型：建议非唯一索引
- enrollment_plan_id 升序

说明：
- 命中 /api/dashboard/overview
- 命中 /api/colleges/:id
- 当前主要是按 enrollment_plan_id 批量 in 查询
- 如果后面你要补“防重复导入”能力，可再单独增加一个唯一索引：enrollment_plan_id + stat_year

### 6. province_score_line

用途：省控线查询。

索引 1：province_score_line_main
- 类型：建议唯一索引
- province 升序
- year 升序
- subject 升序
- score 降序
- batch 升序

说明：
- 命中 /api/province-lines
- 云函数里会先按 province + year + subject 过滤，再按 score/batch 输出
- 这一组字段在业务上应唯一对应一条省控线记录

### 7. score_rank

用途：一分一段、位次换算。

索引 1：score_rank_main
- 类型：建议唯一索引
- province 升序
- year 升序
- subject 升序
- score 降序

说明：
- 命中 /api/score-rank
- 当前代码先拉全量再在内存里找最优匹配，这个索引主要用于缩小扫描范围
- 这一组字段在业务上应唯一对应一个分数档位

## 创建优先级

如果你想先建最关键的一批，优先顺序如下：

1. college_program_group.program_group_list_main
2. college_program_group.program_group_detail_main
3. college_enrollment_plan.enrollment_plan_group_join_main
4. college_enrollment_plan.enrollment_plan_detail_main
5. province_score_line.province_score_line_main
6. score_rank.score_rank_main
7. college_major_admission_stat.major_stat_by_plan
8. college.id
9. college_major.college_major_by_college

## 不建议照搬的 PostgreSQL 索引

下面两类 PostgreSQL 索引在微信云开发里没有等价能力，不要尝试照搬：

- college.name 的 pg_trgm 模糊索引
- college_enrollment_plan.major_name 的 pg_trgm 模糊索引

如果后续要提升关键词检索性能，建议改数据模型：

- 给院校增加 search_tokens 数组字段
- 给专业增加 search_tokens 数组字段
- 查询时先按 province + year + subject 缩小范围，再命中 tokens

## 对应代码位置

当前索引清单基于以下查询路径整理：

- gaokao-ai/miniprogram/cloudfunctions/gaokaoApi/index.js 中的 /api/dashboard/overview
- gaokao-ai/miniprogram/cloudfunctions/gaokaoApi/index.js 中的 /api/province-lines
- gaokao-ai/miniprogram/cloudfunctions/gaokaoApi/index.js 中的 /api/score-rank
- gaokao-ai/miniprogram/cloudfunctions/gaokaoApi/index.js 中的 /api/colleges
- gaokao-ai/miniprogram/cloudfunctions/gaokaoApi/index.js 中的 /api/colleges/:id
- gaokao-ai/miniprogram/cloudfunctions/gaokaoApi/index.js 中的 /api/recommend