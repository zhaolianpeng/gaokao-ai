CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS province_score_line (
  id SERIAL PRIMARY KEY,
  province VARCHAR(20) NOT NULL,
  year INT NOT NULL,
  subject VARCHAR(20) NOT NULL,
  batch VARCHAR(50) NOT NULL,
  score INT NOT NULL,
  source_name VARCHAR(100) DEFAULT '',
  source_url TEXT DEFAULT '',
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE province_score_line ADD COLUMN IF NOT EXISTS source_name VARCHAR(100) DEFAULT '';
ALTER TABLE province_score_line ADD COLUMN IF NOT EXISTS source_url TEXT DEFAULT '';
ALTER TABLE province_score_line ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
CREATE UNIQUE INDEX IF NOT EXISTS uq_province_score_line_main
ON province_score_line (province, year, subject, batch);

CREATE TABLE IF NOT EXISTS score_rank (
  id SERIAL PRIMARY KEY,
  province VARCHAR(20) NOT NULL,
  year INT NOT NULL,
  subject VARCHAR(20) NOT NULL,
  score INT NOT NULL,
  rank INT NOT NULL,
  count INT DEFAULT 0,
  source_name VARCHAR(100) DEFAULT '',
  source_url TEXT DEFAULT '',
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE score_rank ADD COLUMN IF NOT EXISTS source_name VARCHAR(100) DEFAULT '';
ALTER TABLE score_rank ADD COLUMN IF NOT EXISTS source_url TEXT DEFAULT '';
ALTER TABLE score_rank ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
CREATE UNIQUE INDEX IF NOT EXISTS uq_score_rank_main
ON score_rank (province, year, subject, score);

CREATE TABLE IF NOT EXISTS college (
  id SERIAL PRIMARY KEY,
  name VARCHAR(200) NOT NULL,
  province VARCHAR(50) NOT NULL,
  level VARCHAR(50) DEFAULT '',
  is_985 BOOLEAN DEFAULT FALSE,
  is_211 BOOLEAN DEFAULT FALSE,
  is_double_first BOOLEAN DEFAULT FALSE,
  code VARCHAR(50) DEFAULT '',
  city VARCHAR(50) DEFAULT '',
  city_level VARCHAR(50) DEFAULT '',
  address TEXT DEFAULT '',
  longitude NUMERIC(10, 6) DEFAULT 0,
  latitude NUMERIC(10, 6) DEFAULT 0,
  website TEXT DEFAULT '',
  tags JSONB DEFAULT '[]'::jsonb,
  school_level_tags JSONB DEFAULT '[]'::jsonb,
  change_notes TEXT DEFAULT '',
  affiliation VARCHAR(200) DEFAULT '',
  school_type VARCHAR(100) DEFAULT '',
  ownership_type VARCHAR(50) DEFAULT '',
  education_level VARCHAR(50) DEFAULT '',
  recommended_postgraduate_rate NUMERIC(5, 2) DEFAULT 0,
  ranking VARCHAR(50) DEFAULT '',
  transfer_policy TEXT DEFAULT '',
  master_major_count INT DEFAULT 0,
  master_major_list JSONB DEFAULT '[]'::jsonb,
  doctor_major_count INT DEFAULT 0,
  doctor_major_list JSONB DEFAULT '[]'::jsonb,
  admissions_charter_url TEXT DEFAULT '',
  softscience_grade VARCHAR(50) DEFAULT '',
  softscience_ranking VARCHAR(50) DEFAULT '',
  discipline_evaluation TEXT DEFAULT '',
  source_name VARCHAR(100) DEFAULT '',
  source_url TEXT DEFAULT '',
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE college ADD COLUMN IF NOT EXISTS code VARCHAR(50) DEFAULT '';
ALTER TABLE college ADD COLUMN IF NOT EXISTS city VARCHAR(50) DEFAULT '';
ALTER TABLE college ADD COLUMN IF NOT EXISTS city_level VARCHAR(50) DEFAULT '';
ALTER TABLE college ADD COLUMN IF NOT EXISTS address TEXT DEFAULT '';
ALTER TABLE college ADD COLUMN IF NOT EXISTS longitude NUMERIC(10, 6) DEFAULT 0;
ALTER TABLE college ADD COLUMN IF NOT EXISTS latitude NUMERIC(10, 6) DEFAULT 0;
ALTER TABLE college ADD COLUMN IF NOT EXISTS website TEXT DEFAULT '';
ALTER TABLE college ADD COLUMN IF NOT EXISTS tags JSONB DEFAULT '[]'::jsonb;
ALTER TABLE college ADD COLUMN IF NOT EXISTS school_level_tags JSONB DEFAULT '[]'::jsonb;
ALTER TABLE college ADD COLUMN IF NOT EXISTS change_notes TEXT DEFAULT '';
ALTER TABLE college ADD COLUMN IF NOT EXISTS affiliation VARCHAR(200) DEFAULT '';
ALTER TABLE college ADD COLUMN IF NOT EXISTS school_type VARCHAR(100) DEFAULT '';
ALTER TABLE college ADD COLUMN IF NOT EXISTS ownership_type VARCHAR(50) DEFAULT '';
ALTER TABLE college ADD COLUMN IF NOT EXISTS education_level VARCHAR(50) DEFAULT '';
ALTER TABLE college ADD COLUMN IF NOT EXISTS recommended_postgraduate_rate NUMERIC(5, 2) DEFAULT 0;
ALTER TABLE college ADD COLUMN IF NOT EXISTS ranking VARCHAR(50) DEFAULT '';
ALTER TABLE college ADD COLUMN IF NOT EXISTS transfer_policy TEXT DEFAULT '';
ALTER TABLE college ADD COLUMN IF NOT EXISTS master_major_count INT DEFAULT 0;
ALTER TABLE college ADD COLUMN IF NOT EXISTS master_major_list JSONB DEFAULT '[]'::jsonb;
ALTER TABLE college ADD COLUMN IF NOT EXISTS doctor_major_count INT DEFAULT 0;
ALTER TABLE college ADD COLUMN IF NOT EXISTS doctor_major_list JSONB DEFAULT '[]'::jsonb;
ALTER TABLE college ADD COLUMN IF NOT EXISTS admissions_charter_url TEXT DEFAULT '';
ALTER TABLE college ADD COLUMN IF NOT EXISTS softscience_grade VARCHAR(50) DEFAULT '';
ALTER TABLE college ADD COLUMN IF NOT EXISTS softscience_ranking VARCHAR(50) DEFAULT '';
ALTER TABLE college ADD COLUMN IF NOT EXISTS discipline_evaluation TEXT DEFAULT '';
ALTER TABLE college ADD COLUMN IF NOT EXISTS source_name VARCHAR(100) DEFAULT '';
ALTER TABLE college ADD COLUMN IF NOT EXISTS source_url TEXT DEFAULT '';
ALTER TABLE college ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
CREATE UNIQUE INDEX IF NOT EXISTS uq_college_name_province
ON college (name, province);

CREATE TABLE IF NOT EXISTS college_admission_line (
  id SERIAL PRIMARY KEY,
  college_id INT NOT NULL REFERENCES college(id),
  province VARCHAR(20) NOT NULL,
  year INT NOT NULL,
  subject VARCHAR(20) NOT NULL,
  major VARCHAR(200) NOT NULL,
  min_score INT NOT NULL,
  min_rank INT NOT NULL,
  avg_score INT NOT NULL,
  batch VARCHAR(50) DEFAULT '',
  admission_type VARCHAR(200) DEFAULT '',
  source_name VARCHAR(100) DEFAULT '',
  source_url TEXT DEFAULT '',
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE college_admission_line ADD COLUMN IF NOT EXISTS batch VARCHAR(50) DEFAULT '';
ALTER TABLE college_admission_line ADD COLUMN IF NOT EXISTS admission_type VARCHAR(200) DEFAULT '';
ALTER TABLE college_admission_line ALTER COLUMN admission_type TYPE VARCHAR(200);
ALTER TABLE college_admission_line ADD COLUMN IF NOT EXISTS source_name VARCHAR(100) DEFAULT '';
ALTER TABLE college_admission_line ADD COLUMN IF NOT EXISTS source_url TEXT DEFAULT '';
ALTER TABLE college_admission_line ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
DROP INDEX IF EXISTS uq_college_admission_line_main;
CREATE UNIQUE INDEX IF NOT EXISTS uq_college_admission_line_main
ON college_admission_line (college_id, province, year, subject, batch, admission_type, major);

ALTER TABLE mini_auth_user ADD COLUMN IF NOT EXISTS id_card VARCHAR(64) DEFAULT '';
ALTER TABLE mini_auth_user ADD COLUMN IF NOT EXISTS school_name VARCHAR(128) DEFAULT '';
ALTER TABLE mini_auth_user ADD COLUMN IF NOT EXISTS school_year VARCHAR(64) DEFAULT '';
ALTER TABLE mini_auth_user ADD COLUMN IF NOT EXISTS class_name VARCHAR(128) DEFAULT '';
ALTER TABLE mini_auth_user ADD COLUMN IF NOT EXISTS student_no VARCHAR(64) DEFAULT '';
ALTER TABLE mini_auth_user ADD COLUMN IF NOT EXISTS from_recommend BOOLEAN DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS profile_option_config (
  id SERIAL PRIMARY KEY,
  option_type VARCHAR(32) NOT NULL,
  option_label VARCHAR(128) NOT NULL,
  option_value VARCHAR(128) NOT NULL DEFAULT '',
  sort_order INT NOT NULL DEFAULT 0,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_profile_option_config_main
ON profile_option_config (option_type, option_value);

CREATE INDEX IF NOT EXISTS idx_profile_option_config_type
ON profile_option_config (option_type, enabled, sort_order);

CREATE TABLE IF NOT EXISTS college_score_line (
  id SERIAL PRIMARY KEY,
  college_id INT NOT NULL REFERENCES college(id),
  province VARCHAR(20) NOT NULL,
  year INT NOT NULL,
  subject VARCHAR(20) NOT NULL,
  batch VARCHAR(50) NOT NULL DEFAULT '',
  min_score INT NOT NULL DEFAULT 0,
  min_rank INT NOT NULL DEFAULT 0,
  avg_score INT NOT NULL DEFAULT 0,
  max_score INT NOT NULL DEFAULT 0,
  admission_type VARCHAR(200) NOT NULL DEFAULT '',
  source_name VARCHAR(100) DEFAULT '',
  source_url TEXT DEFAULT '',
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE college_score_line ALTER COLUMN admission_type TYPE VARCHAR(200);

CREATE UNIQUE INDEX IF NOT EXISTS uq_college_score_line_main
ON college_score_line (college_id, province, year, subject, batch, admission_type);

CREATE TABLE IF NOT EXISTS major_catalog (
  id SERIAL PRIMARY KEY,
  name VARCHAR(200) NOT NULL,
  category VARCHAR(100) DEFAULT '',
  discipline VARCHAR(100) DEFAULT '',
  degree_level VARCHAR(50) DEFAULT '',
  study_years VARCHAR(50) DEFAULT '',
  intro TEXT DEFAULT '',
  major_strength TEXT DEFAULT '',
  source_name VARCHAR(100) DEFAULT '',
  source_url TEXT DEFAULT '',
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE major_catalog ADD COLUMN IF NOT EXISTS major_strength TEXT DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS uq_major_catalog_name
ON major_catalog (name);

CREATE TABLE IF NOT EXISTS college_major (
  id SERIAL PRIMARY KEY,
  college_id INT NOT NULL REFERENCES college(id),
  major_catalog_id INT REFERENCES major_catalog(id),
  major_name VARCHAR(200) NOT NULL,
  major_full_name VARCHAR(500) DEFAULT '',
  school_major_code VARCHAR(50) DEFAULT '',
  education_level VARCHAR(50) DEFAULT '',
  tuition_fee VARCHAR(50) DEFAULT '',
  study_years VARCHAR(50) DEFAULT '',
  discipline_category VARCHAR(100) DEFAULT '',
  major_category VARCHAR(100) DEFAULT '',
  major_strength TEXT DEFAULT '',
  master_points JSONB DEFAULT '[]'::jsonb,
  doctor_points JSONB DEFAULT '[]'::jsonb,
  is_national_featured BOOLEAN DEFAULT FALSE,
  remarks TEXT DEFAULT '',
  source_name VARCHAR(100) DEFAULT '',
  source_url TEXT DEFAULT '',
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE college_major ADD COLUMN IF NOT EXISTS major_full_name VARCHAR(500) DEFAULT '';
ALTER TABLE college_major ADD COLUMN IF NOT EXISTS discipline_category VARCHAR(100) DEFAULT '';
ALTER TABLE college_major ADD COLUMN IF NOT EXISTS major_category VARCHAR(100) DEFAULT '';
ALTER TABLE college_major ADD COLUMN IF NOT EXISTS major_strength TEXT DEFAULT '';
ALTER TABLE college_major ADD COLUMN IF NOT EXISTS master_points JSONB DEFAULT '[]'::jsonb;
ALTER TABLE college_major ADD COLUMN IF NOT EXISTS doctor_points JSONB DEFAULT '[]'::jsonb;

CREATE UNIQUE INDEX IF NOT EXISTS uq_college_major_name
ON college_major (college_id, major_name);

CREATE TABLE IF NOT EXISTS major_ranking (
  id SERIAL PRIMARY KEY,
  major_catalog_id INT NOT NULL REFERENCES major_catalog(id),
  year INT NOT NULL,
  ranking_org VARCHAR(100) NOT NULL,
  ranking VARCHAR(50) DEFAULT '',
  grade VARCHAR(50) DEFAULT '',
  score NUMERIC(10, 2) DEFAULT 0,
  source_name VARCHAR(100) DEFAULT '',
  source_url TEXT DEFAULT '',
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_major_ranking_main
ON major_ranking (major_catalog_id, year, ranking_org);

CREATE TABLE IF NOT EXISTS major_employment (
  id SERIAL PRIMARY KEY,
  major_catalog_id INT NOT NULL REFERENCES major_catalog(id),
  year INT NOT NULL,
  employment_rate NUMERIC(5, 2) DEFAULT 0,
  average_salary NUMERIC(10, 2) DEFAULT 0,
  counterpart_rate NUMERIC(5, 2) DEFAULT 0,
  industry_distribution JSONB DEFAULT '{}'::jsonb,
  city_distribution JSONB DEFAULT '{}'::jsonb,
  source_name VARCHAR(100) DEFAULT '',
  source_url TEXT DEFAULT '',
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_major_employment_main
ON major_employment (major_catalog_id, year);

CREATE TABLE IF NOT EXISTS crawl_job_log (
  id SERIAL PRIMARY KEY,
  job_name VARCHAR(100) NOT NULL,
  dataset VARCHAR(100) NOT NULL,
  province VARCHAR(20) DEFAULT '',
  year INT DEFAULT 0,
  status VARCHAR(20) NOT NULL,
  row_count INT DEFAULT 0,
  detail TEXT DEFAULT '',
  started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  finished_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS source_import_batch (
  id SERIAL PRIMARY KEY,
  dataset VARCHAR(100) NOT NULL,
  source_name VARCHAR(100) DEFAULT '',
  source_url TEXT DEFAULT '',
  source_file TEXT DEFAULT '',
  province VARCHAR(50) DEFAULT '',
  year INT DEFAULT 0,
  row_count INT DEFAULT 0,
  file_hash VARCHAR(64) DEFAULT '',
  imported_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS gaokao_application_snapshot (
  id SERIAL PRIMARY KEY,
  import_batch_id INT NOT NULL REFERENCES source_import_batch(id),
  row_number INT NOT NULL,
  year INT NOT NULL,
  source_province VARCHAR(50) NOT NULL,
  subject VARCHAR(20) NOT NULL,
  batch VARCHAR(50) NOT NULL,
  batch_remark VARCHAR(100) DEFAULT '',
  category VARCHAR(100) DEFAULT '',
  college_code VARCHAR(50) DEFAULT '',
  college_name VARCHAR(200) NOT NULL,
  college_group_code VARCHAR(50) DEFAULT '',
  college_group_name VARCHAR(200) DEFAULT '',
  group_code VARCHAR(50) DEFAULT '',
  major_code VARCHAR(50) DEFAULT '',
  major_full_name VARCHAR(500) DEFAULT '',
  major_name VARCHAR(200) DEFAULT '',
  major_remark TEXT DEFAULT '',
  subject_requirement VARCHAR(100) DEFAULT '',
  education_level VARCHAR(50) DEFAULT '',
  current_plan_count INT DEFAULT 0,
  study_years VARCHAR(50) DEFAULT '',
  tuition_fee VARCHAR(50) DEFAULT '',
  group_plan_count INT DEFAULT 0,
  discipline_category VARCHAR(100) DEFAULT '',
  major_category VARCHAR(100) DEFAULT '',
  is_new_major BOOLEAN DEFAULT FALSE,
  group_min_score INT DEFAULT 0,
  group_min_rank INT DEFAULT 0,
  school_province VARCHAR(50) DEFAULT '',
  school_city VARCHAR(50) DEFAULT '',
  city_level VARCHAR(50) DEFAULT '',
  college_tags JSONB DEFAULT '[]'::jsonb,
  school_level_tags JSONB DEFAULT '[]'::jsonb,
  change_notes TEXT DEFAULT '',
  affiliation VARCHAR(200) DEFAULT '',
  school_type VARCHAR(100) DEFAULT '',
  ownership_type VARCHAR(50) DEFAULT '',
  school_level VARCHAR(50) DEFAULT '',
  recommended_postgraduate_rate NUMERIC(5, 2) DEFAULT 0,
  college_ranking VARCHAR(50) DEFAULT '',
  transfer_policy TEXT DEFAULT '',
  master_major_count INT DEFAULT 0,
  master_major_list JSONB DEFAULT '[]'::jsonb,
  doctor_major_count INT DEFAULT 0,
  doctor_major_list JSONB DEFAULT '[]'::jsonb,
  admissions_charter_url TEXT DEFAULT '',
  softscience_grade VARCHAR(50) DEFAULT '',
  softscience_ranking VARCHAR(50) DEFAULT '',
  discipline_evaluation TEXT DEFAULT '',
  major_strength TEXT DEFAULT '',
  major_master_points JSONB DEFAULT '[]'::jsonb,
  major_doctor_points JSONB DEFAULT '[]'::jsonb,
  raw_payload JSONB DEFAULT '{}'::jsonb,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_gaokao_application_snapshot_row
ON gaokao_application_snapshot (import_batch_id, row_number);

CREATE INDEX IF NOT EXISTS idx_gaokao_application_snapshot_lookup
ON gaokao_application_snapshot (source_province, year, subject, batch, college_name);

CREATE TABLE IF NOT EXISTS gaokao_application_snapshot_stat (
  id SERIAL PRIMARY KEY,
  snapshot_id INT NOT NULL REFERENCES gaokao_application_snapshot(id) ON DELETE CASCADE,
  stat_year INT NOT NULL,
  legacy_batch VARCHAR(50) DEFAULT '',
  plan_count INT DEFAULT 0,
  admitted_count INT DEFAULT 0,
  min_score INT DEFAULT 0,
  min_rank INT DEFAULT 0,
  max_score INT DEFAULT 0,
  max_rank INT DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_gaokao_application_snapshot_stat
ON gaokao_application_snapshot_stat (snapshot_id, stat_year);

CREATE TABLE IF NOT EXISTS agent_recommend_task (
  id SERIAL PRIMARY KEY,
  title VARCHAR(100) NOT NULL DEFAULT 'AI 智能体报考建议',
  student JSONB NOT NULL DEFAULT '{}'::jsonb,
  demand TEXT NOT NULL,
  templates JSONB NOT NULL DEFAULT '[]'::jsonb,
  suggestions JSONB NOT NULL DEFAULT '[]'::jsonb,
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  report TEXT DEFAULT '',
  provider VARCHAR(50) DEFAULT '',
  error_message TEXT DEFAULT '',
  attempt_count INT NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  started_at TIMESTAMP,
  completed_at TIMESTAMP,
  CONSTRAINT ck_agent_recommend_task_status CHECK (status IN ('pending', 'processing', 'succeeded', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_agent_recommend_task_status_updated
ON agent_recommend_task (status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_agent_recommend_task_created_at
ON agent_recommend_task (created_at DESC);

CREATE TABLE IF NOT EXISTS mini_feedback (
  id SERIAL PRIMARY KEY,
  content TEXT NOT NULL,
  contact VARCHAR(255) NOT NULL DEFAULT '',
  page VARCHAR(128) NOT NULL DEFAULT '',
  backend_base_url VARCHAR(255) NOT NULL DEFAULT '',
  phone VARCHAR(64) NOT NULL DEFAULT '',
  nickname VARCHAR(128) NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_mini_feedback_created_at
ON mini_feedback (created_at DESC);

CREATE TABLE IF NOT EXISTS college_program_group (
  id SERIAL PRIMARY KEY,
  college_id INT NOT NULL REFERENCES college(id),
  province VARCHAR(50) NOT NULL,
  year INT NOT NULL,
  subject VARCHAR(20) NOT NULL,
  batch VARCHAR(50) NOT NULL,
  batch_remark VARCHAR(100) DEFAULT '',
  category VARCHAR(100) DEFAULT '',
  group_code VARCHAR(50) DEFAULT '',
  group_name VARCHAR(200) DEFAULT '',
  subject_requirement VARCHAR(100) DEFAULT '',
  group_plan_count INT DEFAULT 0,
  group_min_score INT DEFAULT 0,
  group_min_rank INT DEFAULT 0,
  source_name VARCHAR(100) DEFAULT '',
  source_url TEXT DEFAULT '',
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_college_program_group_main
ON college_program_group (college_id, province, year, subject, batch, group_code);

CREATE TABLE IF NOT EXISTS college_enrollment_plan (
  id SERIAL PRIMARY KEY,
  college_id INT NOT NULL REFERENCES college(id),
  program_group_id INT REFERENCES college_program_group(id),
  major_catalog_id INT REFERENCES major_catalog(id),
  province VARCHAR(50) NOT NULL,
  year INT NOT NULL,
  subject VARCHAR(20) NOT NULL,
  batch VARCHAR(50) NOT NULL,
  batch_remark VARCHAR(100) DEFAULT '',
  category VARCHAR(100) DEFAULT '',
  major_code VARCHAR(50) DEFAULT '',
  major_full_name VARCHAR(500) DEFAULT '',
  major_name VARCHAR(200) NOT NULL,
  major_remark TEXT DEFAULT '',
  subject_requirement VARCHAR(100) DEFAULT '',
  education_level VARCHAR(50) DEFAULT '',
  plan_count INT DEFAULT 0,
  study_years VARCHAR(50) DEFAULT '',
  tuition_fee VARCHAR(50) DEFAULT '',
  discipline_category VARCHAR(100) DEFAULT '',
  major_category VARCHAR(100) DEFAULT '',
  is_new_major BOOLEAN DEFAULT FALSE,
  source_name VARCHAR(100) DEFAULT '',
  source_url TEXT DEFAULT '',
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_college_enrollment_plan_main
ON college_enrollment_plan (college_id, province, year, subject, batch, COALESCE(program_group_id, 0), major_code, major_name);

CREATE TABLE IF NOT EXISTS college_major_admission_stat (
  id SERIAL PRIMARY KEY,
  enrollment_plan_id INT NOT NULL REFERENCES college_enrollment_plan(id) ON DELETE CASCADE,
  stat_year INT NOT NULL,
  legacy_batch VARCHAR(50) DEFAULT '',
  plan_count INT DEFAULT 0,
  admitted_count INT DEFAULT 0,
  min_score INT DEFAULT 0,
  min_rank INT DEFAULT 0,
  max_score INT DEFAULT 0,
  max_rank INT DEFAULT 0,
  source_name VARCHAR(100) DEFAULT '',
  source_url TEXT DEFAULT '',
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_college_major_admission_stat_main
ON college_major_admission_stat (enrollment_plan_id, stat_year);

CREATE INDEX IF NOT EXISTS idx_college_admission_line_main
ON college_admission_line (province, year, subject, min_rank);

CREATE INDEX IF NOT EXISTS idx_college_score_line_main
ON college_score_line (province, year, subject, min_rank);

CREATE INDEX IF NOT EXISTS idx_major_ranking_year
ON major_ranking (year, ranking_org);

CREATE INDEX IF NOT EXISTS idx_major_employment_year
ON major_employment (year);

CREATE INDEX IF NOT EXISTS idx_college_enrollment_plan_lookup
ON college_enrollment_plan (province, year, subject, batch, major_name);

CREATE INDEX IF NOT EXISTS idx_college_major_admission_stat_year
ON college_major_admission_stat (stat_year, min_rank);

CREATE INDEX IF NOT EXISTS idx_province_score_line_query
ON province_score_line (province, year, subject, score DESC, batch);

CREATE INDEX IF NOT EXISTS idx_college_program_group_detail
ON college_program_group (college_id, province, year, subject, group_min_rank, group_code);

CREATE INDEX IF NOT EXISTS idx_college_enrollment_plan_detail
ON college_enrollment_plan (college_id, province, year, subject, program_group_id, id);

CREATE INDEX IF NOT EXISTS idx_college_enrollment_plan_recommend
ON college_enrollment_plan (province, year, subject, college_id, program_group_id);

CREATE INDEX IF NOT EXISTS idx_college_name_trgm
ON college USING GIN (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_college_enrollment_plan_major_name_trgm
ON college_enrollment_plan USING GIN (major_name gin_trgm_ops);
