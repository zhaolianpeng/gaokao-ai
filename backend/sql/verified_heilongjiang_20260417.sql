-- 已核验数据导入脚本（第二批）
-- 核验时间：2026-04-17
-- 说明：
-- 1. 仅导入已从公开页面直接核验的数据。
-- 2. 部分页面标题为“2025年招生计划及2024年录取情况”，此类数据按 2024 年录取数据入库。
-- 3. 不使用用户提供的占位 college_id，统一以学校名称实时解析真实主键。

INSERT INTO college (name, province, city, level, is_985, is_211, is_double_first, source_name, source_url)
VALUES
('齐齐哈尔医学院', '黑龙江', '齐齐哈尔', '本科', FALSE, FALSE, FALSE, '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn'),
('齐齐哈尔大学', '黑龙江', '齐齐哈尔', '本科', FALSE, FALSE, FALSE, '齐齐哈尔大学招生就业处', 'https://zs.qqhru.edu.cn'),
('哈尔滨学院', '黑龙江', '哈尔滨', '本科', FALSE, FALSE, FALSE, '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn'),
('营口理工学院', '辽宁', '营口', '本科', FALSE, FALSE, FALSE, '营口理工学院招生与就业处', 'https://www.yku.edu.cn/zsxxw')
ON CONFLICT (name, province) DO UPDATE SET
    city = EXCLUDED.city,
    level = EXCLUDED.level,
    source_name = EXCLUDED.source_name,
    source_url = EXCLUDED.source_url,
    updated_at = CURRENT_TIMESTAMP;

INSERT INTO province_score_line (province, year, subject, batch, score, source_name, source_url)
VALUES
('黑龙江', 2025, '历史类', '本科批', 405, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml'),
('黑龙江', 2025, '历史类', '高职（专科）批', 160, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml'),
('黑龙江', 2025, '历史类', '特殊类型招生资格线', 480, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml'),
('黑龙江', 2025, '物理类', '本科批', 360, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml'),
('黑龙江', 2025, '物理类', '高职（专科）批', 160, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml'),
('黑龙江', 2025, '物理类', '特殊类型招生资格线', 472, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml'),
('黑龙江', 2025, '体育（历史类）', '体育类本科批文化课', 283, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml'),
('黑龙江', 2025, '体育（物理类）', '体育类本科批文化课', 252, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml'),
('黑龙江', 2025, '体育（历史类）', '体育类高职（专科）批文化课', 150, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml'),
('黑龙江', 2025, '体育（物理类）', '体育类高职（专科）批文化课', 150, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml'),
('黑龙江', 2025, '体育（历史类）', '体育类本科批术科', 72, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml'),
('黑龙江', 2025, '体育（物理类）', '体育类本科批术科', 72, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml'),
('黑龙江', 2025, '艺术类', '艺术类本科批文化课', 270, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml'),
('黑龙江', 2025, '戏曲类', '艺术类本科批文化课', 180, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml'),
('黑龙江', 2025, '艺术类', '艺术类高职（专科）批文化课', 150, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml'),
('黑龙江', 2025, '艺术类-美术与设计类', '艺术类本科批专业课', 150, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml'),
('黑龙江', 2025, '艺术类-音乐类', '艺术类本科批专业课', 140, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml'),
('黑龙江', 2025, '艺术类-舞蹈类', '艺术类本科批专业课', 180, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml'),
('黑龙江', 2025, '艺术类-播音与主持类', '艺术类本科批专业课', 180, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml'),
('黑龙江', 2025, '艺术类-书法类', '艺术类本科批专业课', 180, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml'),
('黑龙江', 2025, '艺术类-表（导）演类', '艺术类本科批专业课', 170, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml'),
('黑龙江', 2025, '艺术类-戏曲类', '艺术类本科批专业课', 180, '哈尔滨市人民政府网站', 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml')
ON CONFLICT (province, year, subject, batch) DO UPDATE SET
    score = EXCLUDED.score,
    source_name = EXCLUDED.source_name,
    source_url = EXCLUDED.source_url,
    updated_at = CURRENT_TIMESTAMP;

INSERT INTO college_major (college_id, major_catalog_id, major_name, education_level, study_years, tuition_fee, source_name, source_url)
VALUES
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), NULL, '临床医学', '本科', '5', '6500元/年', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), NULL, '精神医学', '本科', '5', '6500元/年', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), NULL, '口腔医学', '本科', '5', '6500元/年', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), NULL, '医学影像学', '本科', '5', '6500元/年', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), NULL, '医学检验技术', '本科', '4', '6500元/年', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), NULL, '预防医学', '本科', '5', '6500元/年', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), NULL, '临床药学', '本科', '5', '5000元/年', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), NULL, '药学', '本科', '4', '6500元/年', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), NULL, '护理学', '本科', '4', '6500元/年', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), NULL, '应用心理学', '本科', '4', '6500元/年', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), NULL, '中药学', '本科', '4', '6500元/年', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm')
ON CONFLICT (college_id, major_name) DO UPDATE SET
    education_level = EXCLUDED.education_level,
    study_years = EXCLUDED.study_years,
    tuition_fee = EXCLUDED.tuition_fee,
    source_name = EXCLUDED.source_name,
    source_url = EXCLUDED.source_url,
    updated_at = CURRENT_TIMESTAMP;

INSERT INTO college_score_line (college_id, province, year, subject, batch, min_score, min_rank, admission_type, source_name, source_url)
VALUES
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), '黑龙江', 2024, '物理类', '本科提前批', 479, 41569, '临床医学(免费医学定向)', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), '黑龙江', 2024, '物理类', '本科批', 540, 22562, '精神医学', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), '黑龙江', 2024, '物理类', '本科批', 538, 23122, '口腔医学', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), '黑龙江', 2024, '物理类', '本科批', 497, 35514, '临床医学', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), '黑龙江', 2024, '物理类', '本科批', 513, 30452, '医学影像学', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), '黑龙江', 2024, '物理类', '本科批', 466, 46141, '预防医学', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), '黑龙江', 2024, '物理类', '本科批', 467, 45792, '临床药学', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), '黑龙江', 2024, '物理类', '本科批', 457, 49383, '药学', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), '黑龙江', 2024, '物理类', '本科批', 454, 50494, '护理学', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), '黑龙江', 2024, '历史类', '本科批', 481, 11698, '应用心理学', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), '黑龙江', 2024, '物理类', '本科批', 505, 32966, '临床医学(地方专项)', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), '黑龙江', 2024, '物理类', '本科批', 512, 30782, '精神医学(地方专项)', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), '黑龙江', 2024, '物理类', '本科批', 512, 30782, '医学影像学(地方专项)', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), '黑龙江', 2024, '物理类', '本科批', 487, 38734, '临床医学(振兴龙江)', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), '黑龙江', 2024, '物理类', '本科批', 494, 36432, '医学影像学(振兴龙江)', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), '黑龙江', 2024, '物理类', '本科批', 499, 34833, '预防医学(振兴龙江)', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), '黑龙江', 2024, '物理类', '本科批', 496, 35821, '药学(振兴龙江)', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), '黑龙江', 2024, '物理类', '本科批', 477, 42266, '精神医学(联合办学)', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), '黑龙江', 2024, '物理类', '本科批', 492, 37058, '口腔医学(联合办学)', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔医学院' AND province='黑龙江'), '黑龙江', 2024, '物理类', '本科批', 481, 40839, '临床医学(联合办学)', '齐齐哈尔医学院招生办', 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔大学' AND province='黑龙江'), '黑龙江', 2024, '历史类', '本科普通批', 455, 0, '普通类', '齐齐哈尔大学招生就业处', 'https://zs.qqhru.edu.cn/info/1022/4505.htm'),
((SELECT id FROM college WHERE name='齐齐哈尔大学' AND province='黑龙江'), '黑龙江', 2024, '物理类', '本科普通批', 409, 0, '普通类', '齐齐哈尔大学招生就业处', 'https://zs.qqhru.edu.cn/info/1022/4505.htm')
ON CONFLICT (college_id, province, year, subject, batch, admission_type) DO UPDATE SET
    min_score = EXCLUDED.min_score,
    min_rank = EXCLUDED.min_rank,
    source_name = EXCLUDED.source_name,
    source_url = EXCLUDED.source_url,
    updated_at = CURRENT_TIMESTAMP;

INSERT INTO college_score_line (college_id, province, year, subject, batch, min_score, avg_score, max_score, admission_type, source_name, source_url)
VALUES
((SELECT id FROM college WHERE name='营口理工学院' AND province='辽宁'), '黑龙江', 2025, '物理类', '本科批', 475, 478, 480, '电气工程及其自动化', '营口理工学院招生与就业处', 'https://www.yku.edu.cn/zsxxw/info/1024/2261.htm'),
((SELECT id FROM college WHERE name='营口理工学院' AND province='辽宁'), '黑龙江', 2025, '物理类', '本科批', 468, 470, 472, '自动化', '营口理工学院招生与就业处', 'https://www.yku.edu.cn/zsxxw/info/1024/2261.htm'),
((SELECT id FROM college WHERE name='营口理工学院' AND province='辽宁'), '黑龙江', 2025, '物理类', '本科批', 461, 466, 474, '机械设计制造及其自动化', '营口理工学院招生与就业处', 'https://www.yku.edu.cn/zsxxw/info/1024/2261.htm'),
((SELECT id FROM college WHERE name='营口理工学院' AND province='辽宁'), '黑龙江', 2025, '物理类', '本科批', 456, 458, 460, '智能科学与技术', '营口理工学院招生与就业处', 'https://www.yku.edu.cn/zsxxw/info/1024/2261.htm'),
((SELECT id FROM college WHERE name='营口理工学院' AND province='辽宁'), '黑龙江', 2025, '物理类', '本科批', 455, 459, 465, '机械电子工程', '营口理工学院招生与就业处', 'https://www.yku.edu.cn/zsxxw/info/1024/2261.htm'),
((SELECT id FROM college WHERE name='营口理工学院' AND province='辽宁'), '黑龙江', 2025, '物理类', '本科批', 452, 456, 462, '新能源材料与器件', '营口理工学院招生与就业处', 'https://www.yku.edu.cn/zsxxw/info/1024/2261.htm'),
((SELECT id FROM college WHERE name='营口理工学院' AND province='辽宁'), '黑龙江', 2025, '物理类', '本科批', 452, 454, 455, '焊接技术与工程', '营口理工学院招生与就业处', 'https://www.yku.edu.cn/zsxxw/info/1024/2261.htm'),
((SELECT id FROM college WHERE name='营口理工学院' AND province='辽宁'), '黑龙江', 2025, '物理类', '本科批', 444, 450, 459, '能源化学工程', '营口理工学院招生与就业处', 'https://www.yku.edu.cn/zsxxw/info/1024/2261.htm'),
((SELECT id FROM college WHERE name='营口理工学院' AND province='辽宁'), '黑龙江', 2025, '物理类', '本科批', 442, 446, 451, '材料成型及控制工程', '营口理工学院招生与就业处', 'https://www.yku.edu.cn/zsxxw/info/1024/2261.htm'),
((SELECT id FROM college WHERE name='营口理工学院' AND province='辽宁'), '黑龙江', 2025, '物理类', '本科批', 441, 445, 448, '应用化学', '营口理工学院招生与就业处', 'https://www.yku.edu.cn/zsxxw/info/1024/2261.htm'),
((SELECT id FROM college WHERE name='营口理工学院' AND province='辽宁'), '黑龙江', 2025, '物理类', '本科批', 440, 446, 451, '能源与环境系统工程', '营口理工学院招生与就业处', 'https://www.yku.edu.cn/zsxxw/info/1024/2261.htm'),
((SELECT id FROM college WHERE name='营口理工学院' AND province='辽宁'), '黑龙江', 2025, '物理类', '本科批', 440, 442, 445, '环境科学与工程', '营口理工学院招生与就业处', 'https://www.yku.edu.cn/zsxxw/info/1024/2261.htm'),
((SELECT id FROM college WHERE name='营口理工学院' AND province='辽宁'), '黑龙江', 2025, '物理类', '本科批', 439, 443, 448, '化学工程与工业生物工程', '营口理工学院招生与就业处', 'https://www.yku.edu.cn/zsxxw/info/1024/2261.htm'),
((SELECT id FROM college WHERE name='营口理工学院' AND province='辽宁'), '黑龙江', 2025, '物理类', '本科批', 429, 431, 432, '金融工程', '营口理工学院招生与就业处', 'https://www.yku.edu.cn/zsxxw/info/1024/2261.htm'),
((SELECT id FROM college WHERE name='营口理工学院' AND province='辽宁'), '黑龙江', 2025, '物理类', '本科批', 426, 431, 438, '物流工程', '营口理工学院招生与就业处', 'https://www.yku.edu.cn/zsxxw/info/1024/2261.htm'),
((SELECT id FROM college WHERE name='营口理工学院' AND province='辽宁'), '黑龙江', 2025, '物理类', '本科批', 425, 426, 427, '物流管理', '营口理工学院招生与就业处', 'https://www.yku.edu.cn/zsxxw/info/1024/2261.htm'),
((SELECT id FROM college WHERE name='营口理工学院' AND province='辽宁'), '黑龙江', 2025, '物理类', '本科批', 424, 431, 438, '供应链管理', '营口理工学院招生与就业处', 'https://www.yku.edu.cn/zsxxw/info/1024/2261.htm'),
((SELECT id FROM college WHERE name='营口理工学院' AND province='辽宁'), '黑龙江', 2025, '历史类', '本科批', 459, 460, 460, '供应链管理', '营口理工学院招生与就业处', 'https://www.yku.edu.cn/zsxxw/info/1024/2261.htm'),
((SELECT id FROM college WHERE name='营口理工学院' AND province='辽宁'), '黑龙江', 2025, '历史类', '本科批', 457, 458, 458, '物流管理', '营口理工学院招生与就业处', 'https://www.yku.edu.cn/zsxxw/info/1024/2261.htm'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 420, 424, 431, '商务经济学', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 426, 431, 439, '金融学', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 418, 420, 425, '学前教育(师范类)', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 420, 428, 447, '小学教育(师范类)', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 431, 446, 488, '英语(师范类)', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 418, 421, 432, '俄语', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 418, 420, 431, '商务英语', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 420, 427, 453, '网络与新媒体', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 449, 462, 485, '会计学', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 438, 444, 463, '财务管理', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 418, 419, 420, '旅游管理', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 420, 433, 451, '地理信息科学', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 437, 456, 485, '数学与应用数学(师范类)', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 427, 433, 457, '物理学(师范类)', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 429, 436, 460, '化学(师范类)', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 437, 442, 458, '电气工程与智能控制', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 439, 443, 456, '电子信息工程', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 431, 436, 447, '人工智能', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 445, 451, 475, '计算机科学与技术', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 431, 437, 448, '软件工程', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 421, 425, 434, '土木工程', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 426, 429, 437, '建筑电气与智能化', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 421, 424, 436, '城市地下空间工程', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 421, 427, 440, '智能建造', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 422, 425, 436, '精细化工', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 422, 425, 431, '环境生态工程', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 426, 430, 436, '食品科学与工程', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 423, 426, 436, '食品质量与安全', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 399, 407, 422, '食品科学与工程(中外合作办学)', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 390, 394, 397, '金融学(协作计划)', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'),
((SELECT id FROM college WHERE name='哈尔滨学院' AND province='黑龙江'), '黑龙江', 2025, '物理类', '本科批', 393, 396, 398, '会计学(协作计划)', '哈尔滨学院招生信息网', 'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700')
ON CONFLICT (college_id, province, year, subject, batch, admission_type) DO UPDATE SET
    min_score = EXCLUDED.min_score,
    avg_score = EXCLUDED.avg_score,
    max_score = EXCLUDED.max_score,
    source_name = EXCLUDED.source_name,
    source_url = EXCLUDED.source_url,
    updated_at = CURRENT_TIMESTAMP;

-- 清洗网页复制带入的异常空白字符，保证后续查询与去重稳定。
UPDATE college
SET
    name = regexp_replace(name, '\s+', '', 'g'),
    city = regexp_replace(city, '\s+', '', 'g')
WHERE source_url IN (
    'https://zhaosheng.qmu.edu.cn',
    'https://zs.qqhru.edu.cn',
    'https://zsxx.hrbu.edu.cn',
    'https://www.yku.edu.cn/zsxxw'
);

UPDATE province_score_line
SET
    subject = regexp_replace(subject, '\s+', '', 'g'),
    batch = regexp_replace(batch, '\s+', '', 'g')
WHERE source_url = 'https://www.harbin.gov.cn/haerbin/c104889/202506/c01_1065500.shtml';

UPDATE college_major
SET
    major_name = regexp_replace(major_name, '\s+', '', 'g'),
    study_years = regexp_replace(study_years, '\s+', '', 'g'),
    tuition_fee = regexp_replace(tuition_fee, '\s+', '', 'g')
WHERE source_url = 'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm';

UPDATE college_score_line
SET
    subject = regexp_replace(subject, '\s+', '', 'g'),
    batch = regexp_replace(batch, '\s+', '', 'g'),
    admission_type = regexp_replace(admission_type, '\s+', '', 'g')
WHERE source_url IN (
    'https://zhaosheng.qmu.edu.cn/2024/1202/c6688a187827/page.htm',
    'https://zs.qqhru.edu.cn/info/1022/4505.htm',
    'https://www.yku.edu.cn/zsxxw/info/1024/2261.htm',
    'https://zsxx.hrbu.edu.cn/f/newsCenter/article/f8216453493545aa86605140c295c700'
);