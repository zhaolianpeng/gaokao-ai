# 黑龙江数据核验结论（2026-04-20，齐齐哈尔大学追加批次）

## 本批已核验并可入库

- 齐齐哈尔大学 2025年黑龙江录取分数线——普通本科批
  - 来源：[https://zs.qqhru.edu.cn/info/1128/4676.htm](https://zs.qqhru.edu.cn/info/1128/4676.htm)
  - 已核验：
    - 页面为齐齐哈尔大学招生就业处官网静态表格，可直接提取专业组、专业、最高分、最低分、平均分。
    - 公告同时覆盖历史类、物理类，以及国家专项、地方专项、固边计划、兴林计划、八省区对等协作计划。
    - 公告未提供最低位次，因此本批按页面现状保留 `min_rank=0`。

## 本批入库说明

- `batch` 统一归一为：
  - 本科批, 本科批-八省区对等协作计划, 本科批-兴林计划, 本科批-固边计划, 本科批-国家专项计划, 本科批-地方专项计划
- `admission_type` 保留“专业组-专业名-备注”原始辨识信息，用于区分同专业在不同专业组或计划类别下的记录。
- `avg_score` 按表结构要求四舍五入为整数。

## 实际入库规模

- 齐齐哈尔大学 `college_score_line` 88 条
  - 历史类 23 条
  - 物理类 65 条

## 产出文件

- SQL：[verified_heilongjiang_20260420_qiqihar.sql](verified_heilongjiang_20260420_qiqihar.sql)
- 说明：[verified_heilongjiang_20260420_qiqihar.md](verified_heilongjiang_20260420_qiqihar.md)