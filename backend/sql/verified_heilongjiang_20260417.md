# 黑龙江数据核验结论（2026-04-17）

## 本批已核验并可入库

- 2025 年黑龙江省普通高校招生录取控制分数线划定
  - 来源：[https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml](https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml)
  - 已核验：
    - 普通类本科批、专科批、特殊类型招生资格线
    - 体育类本科批/专科批文化课与本科批术科
    - 艺术类本科批/专科批文化课
    - 艺术类本科批专业课各类别分数线

- 齐齐哈尔医学院 2025 年招生计划及 2024 年录取情况
  - 来源：[https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm](https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm)
  - 已核验：
    - 2024 年黑龙江本科提前批、本科批、地方专项、振兴龙江、联合办学录取最低分和位次
    - 主要专业学制与学费
  - 入库修正：页面是“2025 招生计划 + 2024 录取情况”，因此录取线按 2024 年入库。

- 营口理工学院 2025 年录取公告 12（黑龙江）
  - 来源：[https://www.yku.edu.cn/zsxxw/info/1024/2261.htm](https://www.yku.edu.cn/zsxxw/info/1024/2261.htm)
  - 已核验：物理类与历史类各专业最低分、平均分、最高分。

- 哈尔滨学院 2025 年我校黑龙江物理类普通本科录取结束公告
  - 来源：[https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700](https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700)
  - 已核验：物理类各专业最低分、平均分、最高分，以及中外合作办学、协作计划条目。

- 齐齐哈尔大学 2025 年分省招生计划及 2024 年录取分数一览表
  - 来源：[https://zs.qqhru.edu.cn/info/1022/4505.htm](https://zs.qqhru.edu.cn/info/1022/4505.htm)
  - 已核验：黑龙江本科普通批 2024 年最低分，历史类 455、物理类 409。
  - 入库修正：这是 2024 年录取分数，不是 2025 年实际录取分数，因此按 2024 年入库。

- 哈尔滨工业大学录取分数查询接口
  - 来源：[https://zsb.hit.edu.cn/information/score](https://zsb.hit.edu.cn/information/score)
  - 已核验：
    - 页面通过 POST 接口 `/information/score-list` 返回黑龙江 2025 年录取明细。
    - 已实际抓取并入库 57 条黑龙江 2025 录取线。
  - 入库说明：
    - 由于表结构 `avg_score` 为整数，本批将接口返回的小数平均分按四舍五入写入。
    - 为避免同名专业在不同校区冲突，`admission_type` 使用“校区-专业名”。

- 哈尔滨商业大学录取分数查询接口
  - 来源：[https://zsxx.hrbcu.edu.cn/lqfs.html](https://zsxx.hrbcu.edu.cn/lqfs.html)
  - 已核验：
    - 页面通过 POST 接口 `https://zsxx.hrbcu.edu.cn/json/list` 返回黑龙江 2025 年分专业录取最低分。
    - 已实际抓取并入库 111 条黑龙江 2025 录取线。
  - 结论：
    - 接口返回的真实数据与用户原始提供的“物理类005组 497、会计学583”等并不一致。
    - 本次以官网接口实际返回值为准入库。

## 已核验但未按原始写法入库

- 齐齐哈尔大学公费师范生、国家专项、地方专项等条目中的 `111`、`68`、`35`
  - 相关来源：
    - [https://zs.qqhru.edu.cn/info/1022/4505.htm](https://zs.qqhru.edu.cn/info/1022/4505.htm)
    - [https://zs.qqhru.edu.cn/info/1022/4507.htm](https://zs.qqhru.edu.cn/info/1022/4507.htm)
  - 结论：这些是招生计划数，不是录取最低分，不能写入 `college_score_line.min_score`。

## 当前仍未纳入本批入库

- 黑龙江大学、东北农业大学、哈尔滨理工大学等条目
  - 当前环境下：
    - `https://zs.hlju.edu.cn` 连接被重置，HTTP 版本返回 500。
    - `https://zsb.neau.edu.cn` 与 HTTP 版本都返回连接重置。
  - 现阶段未拿到可稳定访问的公开明细页或接口，暂不入库。

## 实际入库规模

- 第二批静态页面入库：
  - `province_score_line` 22 条
  - `college_score_line` 72 条
  - `college_major` 11 条
- 第三批接口数据入库：
  - 哈尔滨工业大学 `college_score_line` 57 条
  - 哈尔滨商业大学 `college_score_line` 111 条

## 产出文件

- 第二批 SQL：[verified_heilongjiang_20260417.sql](verified_heilongjiang_20260417.sql)