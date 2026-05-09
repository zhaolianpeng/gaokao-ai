INSERT INTO province_score_line (province, year, subject, batch, score) VALUES
('黑龙江', 2024, '理科', '本科一批', 512),
('黑龙江', 2023, '理科', '本科一批', 503)
ON CONFLICT (province, year, subject, batch) DO UPDATE SET score = EXCLUDED.score;

INSERT INTO score_rank (province, year, subject, score, rank, count) VALUES
('黑龙江', 2024, '理科', 600, 4200, 120),
('黑龙江', 2024, '理科', 590, 6100, 180),
('黑龙江', 2024, '理科', 580, 7500, 260)
ON CONFLICT (province, year, subject, score) DO UPDATE SET rank = EXCLUDED.rank, count = EXCLUDED.count;

INSERT INTO college (id, name, province, city, level, is_985, is_211, is_double_first, website) VALUES
(1, '哈尔滨工业大学', '黑龙江', '哈尔滨', '本科', TRUE, TRUE, TRUE, 'https://www.hit.edu.cn'),
(2, '哈尔滨工程大学', '黑龙江', '哈尔滨', '本科', FALSE, TRUE, TRUE, 'https://www.hrbeu.edu.cn'),
(3, '东北林业大学', '黑龙江', '哈尔滨', '本科', FALSE, TRUE, TRUE, 'https://www.nefu.edu.cn'),
(4, '黑龙江大学', '黑龙江', '哈尔滨', '本科', FALSE, FALSE, FALSE, 'https://www.hlju.edu.cn'),
(5, '黑龙江科技大学', '黑龙江', '哈尔滨', '本科', FALSE, FALSE, FALSE, 'https://www.usth.edu.cn')
ON CONFLICT (id) DO UPDATE SET city = EXCLUDED.city, website = EXCLUDED.website;

INSERT INTO college_score_line (college_id, province, year, subject, batch, min_score, min_rank, avg_score, max_score, admission_type) VALUES
(1, '黑龙江', 2024, '理科', '本科一批', 603, 4300, 611, 625, '普通类'),
(2, '黑龙江', 2024, '理科', '本科一批', 588, 6200, 595, 603, '普通类'),
(3, '黑龙江', 2024, '理科', '本科一批', 580, 7100, 586, 593, '普通类'),
(4, '黑龙江', 2024, '理科', '本科一批', 565, 9000, 571, 579, '普通类'),
(5, '黑龙江', 2024, '理科', '本科一批', 550, 11800, 556, 563, '普通类')
ON CONFLICT (college_id, province, year, subject, batch, admission_type) DO UPDATE SET min_score = EXCLUDED.min_score, min_rank = EXCLUDED.min_rank, avg_score = EXCLUDED.avg_score, max_score = EXCLUDED.max_score;

INSERT INTO major_catalog (id, name, category, discipline, degree_level, study_years) VALUES
(1, '计算机科学与技术', '工学', '计算机类', '本科', '4年'),
(2, '电子信息工程', '工学', '电子信息类', '本科', '4年'),
(3, '自动化', '工学', '自动化类', '本科', '4年'),
(4, '软件工程', '工学', '计算机类', '本科', '4年'),
(5, '人工智能', '工学', '电子信息类', '本科', '4年')
ON CONFLICT (name) DO UPDATE SET category = EXCLUDED.category, discipline = EXCLUDED.discipline;

INSERT INTO college_major (college_id, major_catalog_id, major_name, education_level, study_years, is_national_featured) VALUES
(1, 1, '计算机科学与技术', '本科', '4年', TRUE),
(1, 2, '电子信息工程', '本科', '4年', TRUE),
(2, 3, '自动化', '本科', '4年', FALSE),
(3, 4, '软件工程', '本科', '4年', FALSE),
(4, 5, '人工智能', '本科', '4年', FALSE)
ON CONFLICT (college_id, major_name) DO UPDATE SET education_level = EXCLUDED.education_level, study_years = EXCLUDED.study_years;

INSERT INTO college_admission_line (college_id, province, year, subject, major, min_score, min_rank, avg_score, batch, admission_type) VALUES
(1, '黑龙江', 2024, '理科', '计算机科学与技术', 610, 3500, 618, '本科一批', '普通类'),
(1, '黑龙江', 2024, '理科', '电子信息工程', 603, 4300, 608, '本科一批', '普通类'),
(2, '黑龙江', 2024, '理科', '自动化', 588, 6800, 593, '本科一批', '普通类'),
(2, '黑龙江', 2024, '理科', '通信工程', 592, 6200, 597, '本科一批', '普通类'),
(3, '黑龙江', 2024, '理科', '软件工程', 580, 7600, 585, '本科一批', '普通类'),
(3, '黑龙江', 2024, '理科', '数据科学与大数据技术', 584, 7100, 589, '本科一批', '普通类'),
(4, '黑龙江', 2024, '理科', '信息安全', 565, 9800, 571, '本科一批', '普通类'),
(4, '黑龙江', 2024, '理科', '人工智能', 572, 9000, 577, '本科一批', '普通类'),
(5, '黑龙江', 2024, '理科', '土木工程', 550, 12500, 556, '本科一批', '普通类'),
(5, '黑龙江', 2024, '理科', '机械设计制造及其自动化', 555, 11800, 560, '本科一批', '普通类')
ON CONFLICT (college_id, province, year, subject, major) DO UPDATE SET min_score = EXCLUDED.min_score, min_rank = EXCLUDED.min_rank, avg_score = EXCLUDED.avg_score, batch = EXCLUDED.batch, admission_type = EXCLUDED.admission_type;

INSERT INTO major_ranking (major_catalog_id, year, ranking_org, ranking, grade, score) VALUES
(1, 2024, '软科', '12', 'A', 92.50),
(2, 2024, '软科', '28', 'A-', 87.20),
(3, 2024, '校友会', '35', 'A-', 84.00),
(4, 2024, '软科', '20', 'A', 89.10),
(5, 2024, '校友会', '18', 'A', 90.00)
ON CONFLICT (major_catalog_id, year, ranking_org) DO UPDATE SET ranking = EXCLUDED.ranking, grade = EXCLUDED.grade, score = EXCLUDED.score;

INSERT INTO major_employment (major_catalog_id, year, employment_rate, average_salary, counterpart_rate, industry_distribution, city_distribution) VALUES
(1, 2024, 95.20, 12800, 82.00, '{"互联网": 48, "制造业": 22, "金融": 10}'::jsonb, '{"北京": 26, "上海": 19, "深圳": 17}'::jsonb),
(2, 2024, 92.10, 11300, 78.00, '{"通信": 33, "电子": 31, "互联网": 18}'::jsonb, '{"深圳": 24, "上海": 21, "杭州": 12}'::jsonb),
(3, 2024, 90.80, 10500, 76.50, '{"制造业": 46, "汽车": 18, "能源": 11}'::jsonb, '{"苏州": 15, "上海": 14, "广州": 10}'::jsonb),
(4, 2024, 96.30, 13100, 84.20, '{"互联网": 52, "游戏": 11, "企业服务": 15}'::jsonb, '{"杭州": 23, "深圳": 20, "北京": 18}'::jsonb),
(5, 2024, 93.60, 13600, 79.00, '{"人工智能": 41, "自动驾驶": 17, "云计算": 16}'::jsonb, '{"北京": 25, "上海": 18, "深圳": 18}'::jsonb)
ON CONFLICT (major_catalog_id, year) DO UPDATE SET employment_rate = EXCLUDED.employment_rate, average_salary = EXCLUDED.average_salary, counterpart_rate = EXCLUDED.counterpart_rate, industry_distribution = EXCLUDED.industry_distribution, city_distribution = EXCLUDED.city_distribution;

SELECT setval('college_id_seq', COALESCE((SELECT MAX(id) FROM college), 1));
SELECT setval('major_catalog_id_seq', COALESCE((SELECT MAX(id) FROM major_catalog), 1));
