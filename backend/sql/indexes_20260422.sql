CREATE EXTENSION IF NOT EXISTS pg_trgm;

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