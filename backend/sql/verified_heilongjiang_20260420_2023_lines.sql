-- 已核验数据导入脚本（黑龙江 2023 普通类批次线补录）
-- 核验时间：2026-04-20
-- 说明：
-- 1. 本批补录黑龙江 2023 普通类批次线，解决首页 2023 年份切换无实数的问题。
-- 2. 数值依据阳光高考公开页面标题摘要：黑龙江 2023 年高考录取控制分数线划定。
-- 3. 当前仅补普通类本科一批、本科二批、高职（专科）批；艺术体育线已由考试院网页抓取入库。

INSERT INTO province_score_line (province, year, subject, batch, score, source_name, source_url)
VALUES
('黑龙江', 2023, '文科', '普通本科一批', 430, '阳光高考', 'https://gaokao.chsi.com.cn/gkxx/zc/ss/202306/20230624/2293096255.html'),
('黑龙江', 2023, '文科', '普通本科二批', 341, '阳光高考', 'https://gaokao.chsi.com.cn/gkxx/zc/ss/202306/20230624/2293096255.html'),
('黑龙江', 2023, '文科', '普通高职（专科）批', 160, '阳光高考', 'https://gaokao.chsi.com.cn/gkxx/zc/ss/202306/20230624/2293096255.html'),
('黑龙江', 2023, '理科', '普通本科一批', 408, '阳光高考', 'https://gaokao.chsi.com.cn/gkxx/zc/ss/202306/20230624/2293096255.html'),
('黑龙江', 2023, '理科', '普通本科二批', 287, '阳光高考', 'https://gaokao.chsi.com.cn/gkxx/zc/ss/202306/20230624/2293096255.html'),
('黑龙江', 2023, '理科', '普通高职（专科）批', 160, '阳光高考', 'https://gaokao.chsi.com.cn/gkxx/zc/ss/202306/20230624/2293096255.html')
ON CONFLICT (province, year, subject, batch) DO UPDATE SET
    score = EXCLUDED.score,
    source_name = EXCLUDED.source_name,
    source_url = EXCLUDED.source_url,
    updated_at = CURRENT_TIMESTAMP;