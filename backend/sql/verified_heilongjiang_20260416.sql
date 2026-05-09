-- 已核验数据导入脚本
-- 核验时间：2026-04-16
-- 核验来源：牡丹江师范学院本科招生信息网 2025年黑龙江省普通本科批录取公告
-- 来源页面：http://zs.mdjnu.cn/info/1015/1859.htm
-- 说明：
-- 1. 仅导入可从该页面直接读取并逐项核对的数据。
-- 2. 2025 黑龙江省控线仅能从该页间接核验普通本科批省线：历史类 405、物理类 360。
-- 3. 其余用户提供但当前未能稳定核验的数据，不包含在本脚本中。

INSERT INTO province_score_line (province, year, subject, batch, score, source_name, source_url)
VALUES
('黑龙江', 2025, '历史类', '本科批', 405, '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
('黑龙江', 2025, '物理类', '本科批', 360, '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm')
ON CONFLICT (province, year, subject, batch) DO UPDATE SET
    score = EXCLUDED.score,
    source_name = EXCLUDED.source_name,
    source_url = EXCLUDED.source_url,
    updated_at = CURRENT_TIMESTAMP;

WITH college_ref AS (
    INSERT INTO college (name, province, city, level, is_985, is_211, is_double_first, source_name, source_url)
    VALUES ('牡丹江师范学院', '黑龙江', '牡丹江', '本科', FALSE, FALSE, FALSE, '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn')
    ON CONFLICT (name, province) DO UPDATE SET
        city = EXCLUDED.city,
        level = EXCLUDED.level,
        source_name = EXCLUDED.source_name,
        source_url = EXCLUDED.source_url,
        updated_at = CURRENT_TIMESTAMP
    RETURNING id
)
INSERT INTO college_score_line (
    college_id,
    province,
    year,
    subject,
    batch,
    min_score,
    admission_type,
    source_name,
    source_url
)
SELECT
    college_ref.id,
    data.province,
    data.year,
    data.subject,
    data.batch,
    data.min_score,
    data.admission_type,
    data.source_name,
    data.source_url
FROM college_ref
CROSS JOIN (
    VALUES
    ('黑龙江', 2025, '历史类', '普通本科批', 454, '英语(师范类)-专业组018', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '历史类', '普通本科批', 453, '英语(非师范)-专业组018', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '历史类', '普通本科批', 448, '俄语-专业组018', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '历史类', '普通本科批', 448, '日语-专业组018', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '历史类', '普通本科批', 450, '翻译(英语翻译)-专业组018', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '历史类', '普通本科批', 451, '商务英语-专业组018', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '历史类', '普通本科批', 479, '法学-专业组019', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '历史类', '普通本科批', 459, '知识产权-专业组019', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '历史类', '普通本科批', 451, '国际经贸规则-专业组019', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '历史类', '普通本科批', 451, '学前教育(师范类)-专业组019', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '历史类', '普通本科批', 454, '小学教育(师范类)-专业组019', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '历史类', '普通本科批', 475, '汉语言文学(师范类)-专业组019', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '历史类', '普通本科批', 454, '汉语言-专业组019', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '历史类', '普通本科批', 451, '汉语国际教育-专业组019', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '历史类', '普通本科批', 451, '秘书学-专业组019', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '历史类', '普通本科批', 457, '历史学(师范类)-专业组019', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '历史类', '普通本科批', 461, '经济学-专业组020', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '历史类', '普通本科批', 459, '工商管理-专业组020', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '历史类', '普通本科批', 486, '思想政治教育(师范类)-专业组021', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 415, '经济学-专业组052', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 411, '金融数学-专业组052', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 431, '法学-专业组052', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 416, '知识产权-专业组052', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 410, '国际经贸规则-专业组052', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 417, '小学教育(师范类)-专业组052', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 435, '英语(师范类)-专业组052', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 417, '英语(非师范)-专业组052', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 410, '俄语-专业组052', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 410, '日语-专业组052', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 411, '翻译(英语翻译)-专业组052', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 414, '商务英语-专业组052', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 413, '心理学(师范类)-专业组052', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 413, '工商管理-专业组052', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 444, '数学与应用数学(师范类)-专业组053', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 414, '信息与计算科学-专业组053', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 430, '物理学(师范类)-专业组053', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 430, '化学(师范类)-专业组053', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 418, '应用化学-专业组053', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 423, '地理科学(师范类)-专业组053', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 428, '生物科学(师范类)-专业组053', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 414, '生物技术-专业组053', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 414, '材料物理-专业组053', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 421, '电气工程及其自动化-专业组053', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 415, '电子信息科学与技术-专业组053', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 415, '人工智能-专业组053', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 414, '智能测控工程-专业组053', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 424, '计算机科学与技术-专业组053', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 415, '软件工程-专业组053', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm'),
    ('黑龙江', 2025, '物理类', '普通本科批', 415, '数据科学与大数据技术-专业组053', '牡丹江师范学院本科招生信息网', 'http://zs.mdjnu.cn/info/1015/1859.htm')
) AS data(
    province,
    year,
    subject,
    batch,
    min_score,
    admission_type,
    source_name,
    source_url
)
ON CONFLICT (college_id, province, year, subject, batch, admission_type) DO UPDATE SET
    min_score = EXCLUDED.min_score,
    source_name = EXCLUDED.source_name,
    source_url = EXCLUDED.source_url,
    updated_at = CURRENT_TIMESTAMP;