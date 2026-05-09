# 黑龙江数据核验结论（2026-04-20）

## 本批已核验并可入库

- 黑龙江大学历年分数查询接口
  - 来源：[https://zsbcx.hlju.edu.cn/static/front/hlju/basic/html_web/lnfs.html](https://zsbcx.hlju.edu.cn/static/front/hlju/basic/html_web/lnfs.html)
  - 已核验：
    - 页面前端通过 csrfToken 流程调用 `f/ajax_lnfs_param` 与 `f/ajax_lnfs`。
    - 已实际抓取黑龙江 2025 物理类、历史类、艺术类全部可见专业明细。
    - 已生成可入库 `college_score_line` 156 条。

- 东北农业大学历年分数查询接口
  - 来源：[https://zsb.neau.edu.cn/static/front/neau/basic/html_web/lnfs.html](https://zsb.neau.edu.cn/static/front/neau/basic/html_web/lnfs.html)
  - 已核验：
    - 页面前端通过 `f/ajax_lnfs_param` 与 `f/ajax_lnfs` 返回黑龙江 2025 分专业明细。
    - 已实际抓取黑龙江 2025 物理类、历史类、艺术类全部可见专业明细。
    - 已生成可入库 `college_score_line` 136 条。

## 本批入库说明

- 为保留接口原始分类信息，本批将接口返回的 `zylx/zycc/zslx` 归入 `batch` 字段。
- 专业明细统一写入 `admission_type`；东北农业大学若存在专业组字段，则拼接为“专业名-专业组”。
- 表结构 `avg_score` 为整数，本批使用接口返回的整数均值字段；无位次的艺术类条目按接口现状保留 `min_rank=0`。

## 实际入库规模

- 黑龙江大学 `college_score_line` 156 条
- 东北农业大学 `college_score_line` 136 条
- 合计 `college_score_line` 292 条

## 产出文件

- SQL：[verified_heilongjiang_20260420.sql](verified_heilongjiang_20260420.sql)
- 说明：[verified_heilongjiang_20260420.md](verified_heilongjiang_20260420.md)
