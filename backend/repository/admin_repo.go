package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gaokao-ai/backend/model"
)

type AdminRepository struct {
	db *observedDB
}

type defaultVIPProduct struct {
	ProductID   string
	Name        string
	Description string
	AmountFen   int
	SortOrder   int
}

var defaultVIPProducts = []defaultVIPProduct{
	{ProductID: "vip_single", Name: "VIP 次卡", Description: "单次深度服务", AmountFen: 1, SortOrder: 10},
	{ProductID: "vip_day", Name: "VIP 天卡", Description: "1 天内不限次使用", AmountFen: 1, SortOrder: 20},
	{ProductID: "vip_month", Name: "VIP 月卡", Description: "30 天内不限次使用", AmountFen: 1, SortOrder: 30},
	{ProductID: "vip_season", Name: "VIP 季卡", Description: "90 天内不限次使用", AmountFen: 1, SortOrder: 40},
}

func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{db: observeDB(db)}
}

func appendAdminTextLikeFilter(clauses *[]string, args *[]any, column string, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	*clauses = append(*clauses, fmt.Sprintf("%s LIKE ?", column))
	*args = append(*args, buildLike(trimmed))
}

func appendAdminIntFilter(clauses *[]string, args *[]any, column string, value int) {
	if value <= 0 {
		return
	}
	*clauses = append(*clauses, fmt.Sprintf("%s = ?", column))
	*args = append(*args, value)
}

func (r *AdminRepository) EnsureBootstrap(ctx context.Context, defaultPasswordHash string) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS mis_admin_user (
			id INT AUTO_INCREMENT PRIMARY KEY,
			username VARCHAR(64) NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			display_name VARCHAR(100) NOT NULL DEFAULT '',
			phone VARCHAR(32) NOT NULL DEFAULT '',
			role VARCHAR(50) NOT NULL DEFAULT 'staff',
			status VARCHAR(20) NOT NULL DEFAULT 'enabled',
			last_login_at TIMESTAMP NULL DEFAULT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE KEY uq_mis_admin_user_username (username),
			KEY idx_mis_admin_user_status (status)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS vip_product_config (
			id INT AUTO_INCREMENT PRIMARY KEY,
			product_id VARCHAR(64) NOT NULL,
			name VARCHAR(100) NOT NULL,
			description VARCHAR(255) NOT NULL DEFAULT '',
			amount_fen INT NOT NULL DEFAULT 1,
			enabled TINYINT(1) NOT NULL DEFAULT 1,
			validity_type VARCHAR(20) NOT NULL DEFAULT 'unlimited',
			valid_times INT NOT NULL DEFAULT 0,
			valid_from TIMESTAMP NULL DEFAULT NULL,
			valid_until TIMESTAMP NULL DEFAULT NULL,
			sort_order INT NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE KEY uq_vip_product_config_product_id (product_id),
			KEY idx_vip_product_config_sort_order (sort_order)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS vip_entry_control_config (
			id TINYINT NOT NULL PRIMARY KEY,
			show_vip_entry TINYINT(1) NOT NULL DEFAULT 1,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS share_gate_control_config (
			id TINYINT NOT NULL PRIMARY KEY,
			require_share_for_ai_report TINYINT(1) NOT NULL DEFAULT 0,
			require_share_for_college_major TINYINT(1) NOT NULL DEFAULT 0,
			require_share_for_recommend_result TINYINT(1) NOT NULL DEFAULT 0,
			require_share_for_plan_compare TINYINT(1) NOT NULL DEFAULT 0,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS vip_payment_order (
			id INT AUTO_INCREMENT PRIMARY KEY,
			order_id VARCHAR(64) NOT NULL,
			user_id VARCHAR(24) NOT NULL DEFAULT '',
			openid VARCHAR(128) NOT NULL DEFAULT '',
			product_id VARCHAR(64) NOT NULL DEFAULT '',
			product_name VARCHAR(100) NOT NULL DEFAULT '',
			content VARCHAR(255) NOT NULL DEFAULT '',
			amount_fen INT NOT NULL DEFAULT 0,
			status VARCHAR(32) NOT NULL DEFAULT 'created',
			payment_channel VARCHAR(32) NOT NULL DEFAULT 'wechat-pay',
			prepay_id VARCHAR(128) NOT NULL DEFAULT '',
			transaction_id VARCHAR(128) NOT NULL DEFAULT '',
			remark VARCHAR(255) NOT NULL DEFAULT '',
			paid_at TIMESTAMP NULL DEFAULT NULL,
			expires_at TIMESTAMP NULL DEFAULT NULL,
			effective_from TIMESTAMP NULL DEFAULT NULL,
			effective_until TIMESTAMP NULL DEFAULT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE KEY uq_vip_payment_order_order_id (order_id),
			KEY idx_vip_payment_order_user_id (user_id),
			KEY idx_vip_payment_order_status (status),
			KEY idx_vip_payment_order_product_id (product_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS profile_option_config (
			id INT AUTO_INCREMENT PRIMARY KEY,
			option_type VARCHAR(32) NOT NULL,
			option_label VARCHAR(128) NOT NULL,
			option_value VARCHAR(128) NOT NULL DEFAULT '',
			sort_order INT NOT NULL DEFAULT 0,
			enabled TINYINT(1) NOT NULL DEFAULT 1,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE KEY uq_profile_option_config_main (option_type, option_value),
			KEY idx_profile_option_config_type (option_type, enabled, sort_order)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	}
	for _, statement := range statements {
		if _, err := r.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	alterStatements := []struct {
		column    string
		tableName string
		statement string
	}{
		{tableName: "vip_product_config", column: "validity_type", statement: `ALTER TABLE vip_product_config ADD COLUMN validity_type VARCHAR(20) NOT NULL DEFAULT 'unlimited' AFTER enabled`},
		{tableName: "vip_product_config", column: "valid_times", statement: `ALTER TABLE vip_product_config ADD COLUMN valid_times INT NOT NULL DEFAULT 0 AFTER validity_type`},
		{tableName: "vip_product_config", column: "valid_from", statement: `ALTER TABLE vip_product_config ADD COLUMN valid_from TIMESTAMP NULL DEFAULT NULL AFTER valid_times`},
		{tableName: "vip_product_config", column: "valid_until", statement: `ALTER TABLE vip_product_config ADD COLUMN valid_until TIMESTAMP NULL DEFAULT NULL AFTER valid_from`},
		{tableName: "mini_auth_user", column: "id_card", statement: `ALTER TABLE mini_auth_user ADD COLUMN id_card VARCHAR(64) NOT NULL DEFAULT '' AFTER avatar_url`},
		{tableName: "mini_auth_user", column: "object_id", statement: `ALTER TABLE mini_auth_user ADD COLUMN object_id VARCHAR(24) NOT NULL DEFAULT '' AFTER id`},
		{tableName: "mini_auth_user", column: "school_name", statement: `ALTER TABLE mini_auth_user ADD COLUMN school_name VARCHAR(128) NOT NULL DEFAULT '' AFTER id_card`},
		{tableName: "mini_auth_user", column: "school_year", statement: `ALTER TABLE mini_auth_user ADD COLUMN school_year VARCHAR(64) NOT NULL DEFAULT '' AFTER school_name`},
		{tableName: "mini_auth_user", column: "class_name", statement: `ALTER TABLE mini_auth_user ADD COLUMN class_name VARCHAR(128) NOT NULL DEFAULT '' AFTER school_year`},
		{tableName: "mini_auth_user", column: "student_no", statement: `ALTER TABLE mini_auth_user ADD COLUMN student_no VARCHAR(64) NOT NULL DEFAULT '' AFTER class_name`},
		{tableName: "mini_auth_user", column: "from_recommend", statement: `ALTER TABLE mini_auth_user ADD COLUMN from_recommend TINYINT(1) NOT NULL DEFAULT 0 AFTER student_no`},
		{tableName: "vip_payment_order", column: "expires_at", statement: `ALTER TABLE vip_payment_order ADD COLUMN expires_at TIMESTAMP NULL DEFAULT NULL AFTER paid_at`},
		{tableName: "vip_payment_order", column: "effective_from", statement: `ALTER TABLE vip_payment_order ADD COLUMN effective_from TIMESTAMP NULL DEFAULT NULL AFTER paid_at`},
		{tableName: "vip_payment_order", column: "effective_until", statement: `ALTER TABLE vip_payment_order ADD COLUMN effective_until TIMESTAMP NULL DEFAULT NULL AFTER effective_from`},
		{tableName: "province_score_line", column: "source_name", statement: `ALTER TABLE province_score_line ADD COLUMN source_name VARCHAR(100) NOT NULL DEFAULT '' AFTER score`},
		{tableName: "province_score_line", column: "source_url", statement: `ALTER TABLE province_score_line ADD COLUMN source_url TEXT NULL AFTER source_name`},
		{tableName: "province_score_line", column: "updated_at", statement: `ALTER TABLE province_score_line ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP AFTER source_url`},
		{tableName: "score_rank", column: "source_name", statement: `ALTER TABLE score_rank ADD COLUMN source_name VARCHAR(100) NOT NULL DEFAULT '' AFTER ` + "`count`" + ``},
		{tableName: "score_rank", column: "source_url", statement: `ALTER TABLE score_rank ADD COLUMN source_url TEXT NULL AFTER source_name`},
		{tableName: "score_rank", column: "updated_at", statement: `ALTER TABLE score_rank ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP AFTER source_url`},
		{tableName: "share_gate_control_config", column: "require_share_for_recommend_result", statement: `ALTER TABLE share_gate_control_config ADD COLUMN require_share_for_recommend_result TINYINT(1) NOT NULL DEFAULT 0 AFTER require_share_for_college_major`},
		{tableName: "share_gate_control_config", column: "require_share_for_plan_compare", statement: `ALTER TABLE share_gate_control_config ADD COLUMN require_share_for_plan_compare TINYINT(1) NOT NULL DEFAULT 0 AFTER require_share_for_recommend_result`},
	}
	for _, item := range alterStatements {
		exists, err := r.columnExists(ctx, item.tableName, item.column)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := r.db.ExecContext(ctx, item.statement); err != nil {
			return err
		}
	}
	if err := r.ensureMiniAuthUserObjectIDs(ctx); err != nil {
		return err
	}
	if err := r.ensureVIPPaymentOrderUserIDObjectIDs(ctx); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO mis_admin_user (username, password_hash, display_name, phone, role, status)
		SELECT 'admin', ?, '系统管理员', '', 'super-admin', 'enabled'
		FROM DUAL
		WHERE NOT EXISTS (SELECT 1 FROM mis_admin_user WHERE username = 'admin')
	`, defaultPasswordHash); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO vip_entry_control_config (id, show_vip_entry)
		VALUES (1, 1)
		ON DUPLICATE KEY UPDATE id = id
	`); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO share_gate_control_config (
			id,
			require_share_for_ai_report,
			require_share_for_college_major,
			require_share_for_recommend_result,
			require_share_for_plan_compare
		)
		VALUES (1, 0, 0, 0, 0)
		ON DUPLICATE KEY UPDATE id = id
	`); err != nil {
		return err
	}
	for _, product := range defaultVIPProducts {
		if _, err := r.db.ExecContext(ctx, `
			INSERT IGNORE INTO vip_product_config (product_id, name, description, amount_fen, enabled, validity_type, valid_times, sort_order)
			VALUES (?, ?, ?, ?, 1, 'unlimited', 0, ?)
		`, product.ProductID, product.Name, product.Description, product.AmountFen, product.SortOrder); err != nil {
			return err
		}
	}
	return nil

}

func normalizePage(page, limit int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

func buildLike(keyword string) string {
	trimmed := strings.TrimSpace(keyword)
	if trimmed == "" {
		return "%"
	}
	return "%" + trimmed + "%"
}

func (r *AdminRepository) GetDashboard(ctx context.Context) (*model.AdminDashboard, error) {
	queries := []struct {
		query string
		set   func(*model.AdminDashboard, int)
	}{
		{"SELECT COUNT(*) FROM college", func(d *model.AdminDashboard, value int) { d.CollegeCount = value }},
		{"SELECT COUNT(*) FROM province_score_line", func(d *model.AdminDashboard, value int) { d.ProvinceLineCount = value }},
		{"SELECT COUNT(*) FROM score_rank", func(d *model.AdminDashboard, value int) { d.ScoreRankCount = value }},
		{"SELECT COUNT(*) FROM mini_auth_user", func(d *model.AdminDashboard, value int) { d.StudentCount = value }},
		{"SELECT COUNT(*) FROM mis_admin_user", func(d *model.AdminDashboard, value int) { d.StaffCount = value }},
		{"SELECT COUNT(*) FROM agent_recommend_task WHERE task_type = 'analyze'", func(d *model.AdminDashboard, value int) { d.VolunteerCount = value }},
		{"SELECT COUNT(*) FROM agent_recommend_task", func(d *model.AdminDashboard, value int) { d.AITaskCount = value }},
		{"SELECT COUNT(*) FROM vip_product_config", func(d *model.AdminDashboard, value int) { d.VIPProductCount = value }},
	}
	dashboard := &model.AdminDashboard{}
	for _, item := range queries {
		var value int
		if err := r.db.QueryRowContext(ctx, item.query).Scan(&value); err != nil {
			return nil, err
		}
		item.set(dashboard, value)
	}
	return dashboard, nil
}

func (r *AdminRepository) GetAdminUserAuthByUsername(ctx context.Context, username string) (*model.AdminUserAuth, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, username, display_name, phone, role, status,
			COALESCE(last_login_at, CURRENT_TIMESTAMP), created_at, updated_at, password_hash
		FROM mis_admin_user
		WHERE username = ?
	`, strings.TrimSpace(username))
	var item model.AdminUserAuth
	if err := row.Scan(&item.ID, &item.Username, &item.DisplayName, &item.Phone, &item.Role, &item.Status, &item.LastLoginAt, &item.CreatedAt, &item.UpdatedAt, &item.PasswordHash); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *AdminRepository) GetAdminUserByID(ctx context.Context, id int) (*model.AdminUser, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, username, display_name, phone, role, status,
			COALESCE(last_login_at, CURRENT_TIMESTAMP), created_at, updated_at
		FROM mis_admin_user WHERE id = ?
	`, id)
	var item model.AdminUser
	if err := row.Scan(&item.ID, &item.Username, &item.DisplayName, &item.Phone, &item.Role, &item.Status, &item.LastLoginAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *AdminRepository) TouchAdminLogin(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `UPDATE mis_admin_user SET last_login_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

func (r *AdminRepository) ListAdminUsers(ctx context.Context, req model.AdminUserListRequest) ([]model.AdminUser, int, error) {
	page, limit := req.Normalized()
	like := buildLike(req.Keyword)
	var total int
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM mis_admin_user
		WHERE username LIKE ? OR display_name LIKE ? OR phone LIKE ?
	`, like, like, like).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, username, display_name, phone, role, status,
			COALESCE(last_login_at, CURRENT_TIMESTAMP), created_at, updated_at
		FROM mis_admin_user
		WHERE username LIKE ? OR display_name LIKE ? OR phone LIKE ?
		ORDER BY id DESC LIMIT ? OFFSET ?
	`, like, like, like, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]model.AdminUser, 0)
	for rows.Next() {
		var item model.AdminUser
		if err := rows.Scan(&item.ID, &item.Username, &item.DisplayName, &item.Phone, &item.Role, &item.Status, &item.LastLoginAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *AdminRepository) SaveAdminUser(ctx context.Context, item model.AdminUser, passwordHash string) (int, error) {
	if item.ID > 0 {
		if strings.TrimSpace(passwordHash) != "" {
			_, err := r.db.ExecContext(ctx, `
				UPDATE mis_admin_user
				SET username = ?, display_name = ?, phone = ?, role = ?, status = ?, password_hash = ?, updated_at = CURRENT_TIMESTAMP
				WHERE id = ?
			`, strings.TrimSpace(item.Username), strings.TrimSpace(item.DisplayName), strings.TrimSpace(item.Phone), strings.TrimSpace(item.Role), strings.TrimSpace(item.Status), passwordHash, item.ID)
			return item.ID, err
		}
		_, err := r.db.ExecContext(ctx, `
			UPDATE mis_admin_user
			SET username = ?, display_name = ?, phone = ?, role = ?, status = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, strings.TrimSpace(item.Username), strings.TrimSpace(item.DisplayName), strings.TrimSpace(item.Phone), strings.TrimSpace(item.Role), strings.TrimSpace(item.Status), item.ID)
		return item.ID, err
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO mis_admin_user (username, password_hash, display_name, phone, role, status)
		VALUES (?, ?, ?, ?, ?, ?)
	`, strings.TrimSpace(item.Username), passwordHash, strings.TrimSpace(item.DisplayName), strings.TrimSpace(item.Phone), strings.TrimSpace(item.Role), strings.TrimSpace(item.Status))
	if err != nil {
		return 0, err
	}
	insertID, _ := result.LastInsertId()
	return int(insertID), nil
}

func (r *AdminRepository) DeleteAdminUser(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM mis_admin_user WHERE id = ?`, id)
	return err
}

func (r *AdminRepository) ListColleges(ctx context.Context, req model.AdminCollegeListRequest) ([]model.AdminCollege, int, error) {
	page, limit := req.Normalized()
	conditions := []string{"1 = 1"}
	args := make([]any, 0)
	if strings.TrimSpace(req.Keyword) != "" {
		like := buildLike(req.Keyword)
		keywordConditions := []string{
			`name LIKE ?`,
			`province LIKE ?`,
			`city LIKE ?`,
			`level LIKE ?`,
			`school_type LIKE ?`,
			`ownership_type LIKE ?`,
			`ranking LIKE ?`,
			`code LIKE ?`,
			`affiliation LIKE ?`,
			`education_level LIKE ?`,
			`address LIKE ?`,
			`website LIKE ?`,
			`softscience_grade LIKE ?`,
			`softscience_ranking LIKE ?`,
			`CAST(tags AS CHAR) LIKE ?`,
			`CAST(school_level_tags AS CHAR) LIKE ?`,
		}
		for range keywordConditions {
			args = append(args, like)
		}
		normalizedKeyword := strings.ToLower(strings.TrimSpace(req.Keyword))
		if strings.Contains(normalizedKeyword, "985") {
			keywordConditions = append(keywordConditions, `is_985 = 1`)
		}
		if strings.Contains(normalizedKeyword, "211") {
			keywordConditions = append(keywordConditions, `is_211 = 1`)
		}
		if strings.Contains(normalizedKeyword, "双一流") || strings.Contains(normalizedKeyword, "double first") {
			keywordConditions = append(keywordConditions, `is_double_first = 1`)
		}
		conditions = append(conditions, `(`+strings.Join(keywordConditions, ` OR `)+`)`)
	}
	if strings.TrimSpace(req.Name) != "" {
		conditions = append(conditions, `name LIKE ?`)
		args = append(args, buildLike(req.Name))
	}
	if strings.TrimSpace(req.Level) != "" {
		conditions = append(conditions, `level LIKE ?`)
		args = append(args, buildLike(req.Level))
	}
	if strings.TrimSpace(req.SchoolType) != "" {
		conditions = append(conditions, `school_type LIKE ?`)
		args = append(args, buildLike(req.SchoolType))
	}
	if strings.TrimSpace(req.OwnershipType) != "" {
		conditions = append(conditions, `ownership_type LIKE ?`)
		args = append(args, buildLike(req.OwnershipType))
	}
	if strings.TrimSpace(req.Province) != "" {
		conditions = append(conditions, `province LIKE ?`)
		args = append(args, buildLike(req.Province))
	}
	if strings.TrimSpace(req.City) != "" {
		conditions = append(conditions, `city LIKE ?`)
		args = append(args, buildLike(req.City))
	}
	switch strings.TrimSpace(req.Is985) {
	case "1":
		conditions = append(conditions, `is_985 = 1`)
	case "0":
		conditions = append(conditions, `is_985 = 0`)
	}
	switch strings.TrimSpace(req.Is211) {
	case "1":
		conditions = append(conditions, `is_211 = 1`)
	case "0":
		conditions = append(conditions, `is_211 = 0`)
	}
	switch strings.TrimSpace(req.IsDoubleFirst) {
	case "1":
		conditions = append(conditions, `is_double_first = 1`)
	case "0":
		conditions = append(conditions, `is_double_first = 0`)
	}
	where := strings.Join(conditions, " AND ")
	countArgs := append([]any{}, args...)
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM college WHERE `+where, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}
	queryArgs := append(append([]any{}, args...), limit, (page-1)*limit)
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, province, city, level, is_985, is_211, is_double_first,
			COALESCE(website, ''), COALESCE(ranking, ''), COALESCE(school_type, ''), COALESCE(ownership_type, ''),
			CAST(recommended_postgraduate_rate AS CHAR), updated_at
		FROM college
		WHERE `+where+`
		ORDER BY id DESC LIMIT ? OFFSET ?
	`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]model.AdminCollege, 0)
	for rows.Next() {
		var item model.AdminCollege
		if err := rows.Scan(&item.ID, &item.Name, &item.Province, &item.City, &item.Level, &item.Is985, &item.Is211, &item.IsDoubleFirst, &item.Website, &item.Ranking, &item.SchoolType, &item.OwnershipType, &item.RecommendedPostgraduateRate, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *AdminRepository) SaveCollege(ctx context.Context, item model.AdminCollege) (int, error) {
	if item.ID > 0 {
		_, err := r.db.ExecContext(ctx, `
			UPDATE college
			SET name = ?, province = ?, city = ?, level = ?, is_985 = ?, is_211 = ?, is_double_first = ?,
				website = ?, ranking = ?, school_type = ?, ownership_type = ?, recommended_postgraduate_rate = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, item.Name, item.Province, item.City, item.Level, item.Is985, item.Is211, item.IsDoubleFirst, item.Website, item.Ranking, item.SchoolType, item.OwnershipType, item.RecommendedPostgraduateRate, item.ID)
		return item.ID, err
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO college (name, province, city, level, is_985, is_211, is_double_first, website, ranking, school_type, ownership_type, recommended_postgraduate_rate, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, item.Name, item.Province, item.City, item.Level, item.Is985, item.Is211, item.IsDoubleFirst, item.Website, item.Ranking, item.SchoolType, item.OwnershipType, item.RecommendedPostgraduateRate)
	if err != nil {
		return 0, err
	}
	id, _ := result.LastInsertId()
	return int(id), nil
}

func (r *AdminRepository) ListProvinceScoreLines(ctx context.Context, req model.AdminProvinceScoreLineListRequest) ([]model.AdminProvinceScoreLine, int, error) {
	page, limit := req.Normalized()
	like := buildLike(req.Keyword)
	clauses := []string{"(province LIKE ? OR subject LIKE ? OR batch LIKE ?)"}
	args := []any{like, like, like}
	appendAdminTextLikeFilter(&clauses, &args, "province", req.Province)
	appendAdminIntFilter(&clauses, &args, "year", req.Year)
	appendAdminTextLikeFilter(&clauses, &args, "subject", req.Subject)
	appendAdminTextLikeFilter(&clauses, &args, "batch", req.Batch)
	whereSQL := strings.Join(clauses, " AND ")
	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM province_score_line WHERE %s`, whereSQL)
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	listArgs := append(append([]any{}, args...), limit, (page-1)*limit)
	listQuery := fmt.Sprintf(`
		SELECT id, province, year, subject, batch, score, COALESCE(source_name, ''), COALESCE(source_url, ''), updated_at
		FROM province_score_line
		WHERE %s
		ORDER BY year DESC, province ASC, subject ASC, batch ASC, score DESC, id DESC LIMIT ? OFFSET ?
	`, whereSQL)
	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]model.AdminProvinceScoreLine, 0)
	for rows.Next() {
		var item model.AdminProvinceScoreLine
		if err := rows.Scan(&item.ID, &item.Province, &item.Year, &item.Subject, &item.Batch, &item.Score, &item.SourceName, &item.SourceURL, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *AdminRepository) SaveProvinceScoreLine(ctx context.Context, item model.AdminProvinceScoreLine) (int, error) {
	if item.ID > 0 {
		_, err := r.db.ExecContext(ctx, `
			UPDATE province_score_line SET province = ?, year = ?, subject = ?, batch = ?, score = ?, source_name = ?, source_url = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
		`, item.Province, item.Year, item.Subject, item.Batch, item.Score, item.SourceName, item.SourceURL, item.ID)
		return item.ID, err
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO province_score_line (province, year, subject, batch, score, source_name, source_url, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, item.Province, item.Year, item.Subject, item.Batch, item.Score, item.SourceName, item.SourceURL)
	if err != nil {
		return 0, err
	}
	id, _ := result.LastInsertId()
	return int(id), nil
}

func (r *AdminRepository) DeleteProvinceScoreLine(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM province_score_line WHERE id = ?`, id)
	return err
}

func (r *AdminRepository) ListScoreRanks(ctx context.Context, req model.AdminScoreRankListRequest) ([]model.AdminScoreRank, int, error) {
	page, limit := req.Normalized()
	like := buildLike(req.Keyword)
	clauses := []string{"(province LIKE ? OR subject LIKE ? OR CAST(score AS CHAR) LIKE ? OR CAST(`rank` AS CHAR) LIKE ?)"}
	args := []any{like, like, like, like}
	appendAdminTextLikeFilter(&clauses, &args, "province", req.Province)
	appendAdminIntFilter(&clauses, &args, "year", req.Year)
	appendAdminTextLikeFilter(&clauses, &args, "subject", req.Subject)
	appendAdminIntFilter(&clauses, &args, "score", req.Score)
	whereSQL := strings.Join(clauses, " AND ")
	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM score_rank WHERE %s`, whereSQL)
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	listArgs := append(append([]any{}, args...), limit, (page-1)*limit)
	listQuery := fmt.Sprintf(`
		SELECT id, province, year, subject, score, `+"`rank`"+`, `+"`count`"+`, updated_at
		FROM score_rank
		WHERE %s
		ORDER BY year DESC, province ASC, subject ASC, score DESC, `+"`rank`"+` ASC, id DESC LIMIT ? OFFSET ?
	`, whereSQL)
	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]model.AdminScoreRank, 0)
	for rows.Next() {
		var item model.AdminScoreRank
		if err := rows.Scan(&item.ID, &item.Province, &item.Year, &item.Subject, &item.Score, &item.Rank, &item.Count, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *AdminRepository) SaveScoreRank(ctx context.Context, item model.AdminScoreRank) (int, error) {
	if item.ID > 0 {
		_, err := r.db.ExecContext(ctx, `
			UPDATE score_rank SET province = ?, year = ?, subject = ?, score = ?, `+"`rank`"+` = ?, `+"`count`"+` = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
		`, item.Province, item.Year, item.Subject, item.Score, item.Rank, item.Count, item.ID)
		return item.ID, err
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO score_rank (province, year, subject, score, `+"`rank`"+`, `+"`count`"+`, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, item.Province, item.Year, item.Subject, item.Score, item.Rank, item.Count)
	if err != nil {
		return 0, err
	}
	id, _ := result.LastInsertId()
	return int(id), nil
}

func (r *AdminRepository) DeleteScoreRank(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM score_rank WHERE id = ?`, id)
	return err
}

func (r *AdminRepository) ListStudents(ctx context.Context, req model.AdminStudentListRequest) ([]model.AdminStudent, int, error) {
	page, limit := req.Normalized()
	like := buildLike(req.Keyword)
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM mini_auth_user WHERE openid LIKE ? OR phone LIKE ? OR nickname LIKE ? OR school_name LIKE ? OR class_name LIKE ? OR student_no LIKE ?`, like, like, like, like, like, like).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT object_id, openid, phone, nickname, COALESCE(avatar_url, ''), COALESCE(id_card, ''), COALESCE(school_name, ''), COALESCE(school_year, ''), COALESCE(class_name, ''), COALESCE(student_no, ''), COALESCE(from_recommend, 0), login_type, created_at, updated_at, last_login_at
		FROM mini_auth_user
		WHERE openid LIKE ? OR phone LIKE ? OR nickname LIKE ? OR school_name LIKE ? OR class_name LIKE ? OR student_no LIKE ?
		ORDER BY id DESC LIMIT ? OFFSET ?
	`, like, like, like, like, like, like, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]model.AdminStudent, 0)
	for rows.Next() {
		var item model.AdminStudent
		if err := rows.Scan(&item.ID, &item.OpenID, &item.Phone, &item.Nickname, &item.AvatarURL, &item.IDCard, &item.SchoolName, &item.SchoolYear, &item.ClassName, &item.StudentNo, &item.FromRecommend, &item.LoginType, &item.CreatedAt, &item.UpdatedAt, &item.LastLoginAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *AdminRepository) SaveStudent(ctx context.Context, item model.AdminStudent) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE mini_auth_user
		SET phone = ?, nickname = ?, avatar_url = ?, id_card = ?, school_name = ?, school_year = ?, class_name = ?, student_no = ?, from_recommend = ?, login_type = ?, updated_at = CURRENT_TIMESTAMP
		WHERE object_id = ?
	`, item.Phone, item.Nickname, item.AvatarURL, item.IDCard, item.SchoolName, item.SchoolYear, item.ClassName, item.StudentNo, item.FromRecommend, item.LoginType, item.ID)
	return err
}

func (r *AdminRepository) ListProfileOptions(ctx context.Context, req model.AdminProfileOptionListRequest) ([]model.AdminProfileOption, int, error) {
	page, limit := req.Normalized()
	clauses := []string{"1 = 1"}
	args := make([]any, 0)
	appendAdminTextLikeFilter(&clauses, &args, "option_type", req.OptionType)
	if strings.TrimSpace(req.Keyword) != "" {
		like := buildLike(req.Keyword)
		clauses = append(clauses, `(option_label LIKE ? OR option_value LIKE ?)`)
		args = append(args, like, like)
	}
	where := strings.Join(clauses, " AND ")
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM profile_option_config WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	queryArgs := append(append([]any{}, args...), limit, (page-1)*limit)
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, option_type, option_label, option_value, sort_order, enabled, updated_at
		FROM profile_option_config
		WHERE `+where+`
		ORDER BY option_type ASC, sort_order ASC, id ASC
		LIMIT ? OFFSET ?
	`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]model.AdminProfileOption, 0)
	for rows.Next() {
		var item model.AdminProfileOption
		if err := rows.Scan(&item.ID, &item.OptionType, &item.OptionLabel, &item.OptionValue, &item.SortOrder, &item.Enabled, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *AdminRepository) SaveProfileOption(ctx context.Context, item model.AdminProfileOption) (int, error) {
	optionType := strings.TrimSpace(item.OptionType)
	optionLabel := strings.TrimSpace(item.OptionLabel)
	optionValue := strings.TrimSpace(item.OptionValue)
	if optionType == "" || optionLabel == "" {
		return 0, fmt.Errorf("option type and label required")
	}
	if optionValue == "" {
		optionValue = optionLabel
	}
	if item.ID > 0 {
		_, err := r.db.ExecContext(ctx, `
			UPDATE profile_option_config
			SET option_type = ?, option_label = ?, option_value = ?, sort_order = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, optionType, optionLabel, optionValue, item.SortOrder, item.Enabled, item.ID)
		return item.ID, err
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO profile_option_config (option_type, option_label, option_value, sort_order, enabled)
		VALUES (?, ?, ?, ?, ?)
	`, optionType, optionLabel, optionValue, item.SortOrder, item.Enabled)
	if err != nil {
		return 0, err
	}
	id, _ := result.LastInsertId()
	return int(id), nil
}

func (r *AdminRepository) DeleteProfileOption(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM profile_option_config WHERE id = ?`, id)
	return err
}

func (r *AdminRepository) ListEnabledProfileOptions(ctx context.Context) (*model.ProfileOptionCatalogResponse, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT option_type, option_label, option_value
		FROM profile_option_config
		WHERE enabled = 1
		ORDER BY option_type ASC, sort_order ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := &model.ProfileOptionCatalogResponse{
		Schools:     make([]model.ProfileOptionItem, 0),
		SchoolYears: make([]model.ProfileOptionItem, 0),
		ClassNames:  make([]model.ProfileOptionItem, 0),
	}
	for rows.Next() {
		var optionType string
		var label string
		var value string
		if err := rows.Scan(&optionType, &label, &value); err != nil {
			return nil, err
		}
		item := model.ProfileOptionItem{Label: label, Value: value}
		switch optionType {
		case "school":
			result.Schools = append(result.Schools, item)
		case "schoolYear":
			result.SchoolYears = append(result.SchoolYears, item)
		case "className":
			result.ClassNames = append(result.ClassNames, item)
		}
	}
	return result, rows.Err()
}

func (r *AdminRepository) ListTasks(ctx context.Context, req model.AdminTaskListRequest) ([]model.AdminTask, int, error) {
	page, limit := req.Normalized()
	conditions := []string{"1 = 1"}
	args := make([]any, 0)
	if strings.TrimSpace(req.Keyword) != "" {
		like := buildLike(req.Keyword)
		conditions = append(conditions, `(title LIKE ? OR demand LIKE ? OR report LIKE ?)`)
		args = append(args, like, like, like)
	}
	if strings.TrimSpace(req.TaskType) != "" {
		conditions = append(conditions, `task_type = ?`)
		args = append(args, strings.TrimSpace(req.TaskType))
	}
	if strings.TrimSpace(req.Status) != "" {
		conditions = append(conditions, `status = ?`)
		args = append(args, strings.TrimSpace(req.Status))
	}
	where := strings.Join(conditions, " AND ")
	countQuery := `SELECT COUNT(*) FROM agent_recommend_task WHERE ` + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	queryArgs := append(append([]any{}, args...), limit, (page-1)*limit)
	query := `
		SELECT id, title, task_type, status, provider, demand,
			CAST(student AS CHAR), COALESCE(report, ''), COALESCE(error_message, ''), attempt_count,
			created_at, updated_at, COALESCE(completed_at, CURRENT_TIMESTAMP)
		FROM agent_recommend_task WHERE ` + where + `
		ORDER BY id DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]model.AdminTask, 0)
	for rows.Next() {
		var item model.AdminTask
		if err := rows.Scan(&item.ID, &item.Title, &item.TaskType, &item.Status, &item.Provider, &item.Demand, &item.Student, &item.Report, &item.ErrorMessage, &item.AttemptCount, &item.CreatedAt, &item.UpdatedAt, &item.CompletedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *AdminRepository) DeleteTask(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM agent_recommend_task WHERE id = ?`, id)
	return err
}

func (r *AdminRepository) UpsertPaymentOrder(ctx context.Context, item model.AdminOrder) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO vip_payment_order (
			order_id, user_id, openid, product_id, product_name, content, amount_fen,
			status, payment_channel, prepay_id, transaction_id, remark, paid_at, expires_at, effective_from, effective_until
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			user_id = CASE WHEN VALUES(user_id) <> '' THEN VALUES(user_id) ELSE user_id END,
			openid = CASE WHEN VALUES(openid) <> '' THEN VALUES(openid) ELSE openid END,
			product_id = CASE WHEN VALUES(product_id) <> '' THEN VALUES(product_id) ELSE product_id END,
			product_name = CASE WHEN VALUES(product_name) <> '' THEN VALUES(product_name) ELSE product_name END,
			content = CASE WHEN VALUES(content) <> '' THEN VALUES(content) ELSE content END,
			amount_fen = CASE WHEN VALUES(amount_fen) > 0 THEN VALUES(amount_fen) ELSE amount_fen END,
			status = VALUES(status),
			payment_channel = CASE WHEN VALUES(payment_channel) <> '' THEN VALUES(payment_channel) ELSE payment_channel END,
			prepay_id = CASE WHEN VALUES(prepay_id) <> '' THEN VALUES(prepay_id) ELSE prepay_id END,
			transaction_id = CASE WHEN VALUES(transaction_id) <> '' THEN VALUES(transaction_id) ELSE transaction_id END,
			remark = CASE WHEN VALUES(remark) <> '' THEN VALUES(remark) ELSE remark END,
			paid_at = COALESCE(VALUES(paid_at), paid_at),
			expires_at = COALESCE(VALUES(expires_at), expires_at),
			effective_from = COALESCE(VALUES(effective_from), effective_from),
			effective_until = COALESCE(VALUES(effective_until), effective_until),
			updated_at = CURRENT_TIMESTAMP
	`, strings.TrimSpace(item.OrderID), item.UserID, strings.TrimSpace(item.OpenID), strings.TrimSpace(item.ProductID), strings.TrimSpace(item.ProductName), strings.TrimSpace(item.Content), item.AmountFen, strings.TrimSpace(item.Status), strings.TrimSpace(item.PaymentChannel), strings.TrimSpace(item.PrepayID), strings.TrimSpace(item.TransactionID), strings.TrimSpace(item.Remark), item.PaidAt, item.ExpiresAt, item.EffectiveFrom, item.EffectiveUntil)
	return err
}

func (r *AdminRepository) ListOrders(ctx context.Context, req model.AdminOrderListRequest) ([]model.AdminOrder, int, error) {
	if err := r.CloseExpiredCreatedOrders(ctx, time.Now()); err != nil {
		return nil, 0, err
	}
	page, limit := req.Normalized()
	conditions := []string{"1 = 1"}
	args := make([]any, 0)
	if strings.TrimSpace(req.Keyword) != "" {
		like := buildLike(req.Keyword)
		conditions = append(conditions, `(o.order_id LIKE ? OR o.product_id LIKE ? OR o.product_name LIKE ? OR o.content LIKE ? OR u.nickname LIKE ? OR u.phone LIKE ?)`)
		args = append(args, like, like, like, like, like, like)
	}
	if strings.TrimSpace(req.Status) != "" {
		conditions = append(conditions, `o.status = ?`)
		args = append(args, strings.TrimSpace(req.Status))
	}
	if strings.TrimSpace(req.ProductID) != "" {
		conditions = append(conditions, `o.product_id = ?`)
		args = append(args, strings.TrimSpace(req.ProductID))
	}
	where := strings.Join(conditions, " AND ")
	countQuery := `
		SELECT COUNT(*)
		FROM vip_payment_order o
		LEFT JOIN mini_auth_user u ON u.object_id = o.user_id
		WHERE ` + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	queryArgs := append(append([]any{}, args...), limit, (page-1)*limit)
	query := `
		SELECT o.id, o.order_id, o.user_id, COALESCE(u.nickname, ''), COALESCE(u.phone, ''), o.openid,
			o.product_id, o.product_name, o.content, o.amount_fen, o.status, o.payment_channel,
			o.prepay_id, o.transaction_id, o.remark, o.paid_at, o.expires_at, o.effective_from, o.effective_until, o.created_at, o.updated_at
		FROM vip_payment_order o
		LEFT JOIN mini_auth_user u ON u.object_id = o.user_id
		WHERE ` + where + `
		ORDER BY o.id DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]model.AdminOrder, 0)
	for rows.Next() {
		var item model.AdminOrder
		var paidAt sql.NullTime
		var expiresAt sql.NullTime
		var effectiveFrom sql.NullTime
		var effectiveUntil sql.NullTime
		if err := rows.Scan(&item.ID, &item.OrderID, &item.UserID, &item.UserNickname, &item.UserPhone, &item.OpenID, &item.ProductID, &item.ProductName, &item.Content, &item.AmountFen, &item.Status, &item.PaymentChannel, &item.PrepayID, &item.TransactionID, &item.Remark, &paidAt, &expiresAt, &effectiveFrom, &effectiveUntil, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if paidAt.Valid {
			item.PaidAt = &paidAt.Time
		}
		if expiresAt.Valid {
			item.ExpiresAt = &expiresAt.Time
		}
		if effectiveFrom.Valid {
			item.EffectiveFrom = &effectiveFrom.Time
		}
		if effectiveUntil.Valid {
			item.EffectiveUntil = &effectiveUntil.Time
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *AdminRepository) GetLatestPaidOrderByUserID(ctx context.Context, userID string) (*model.AdminOrder, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, order_id, user_id, openid, product_id, product_name, content, amount_fen,
			status, payment_channel, prepay_id, transaction_id, remark, paid_at, expires_at, effective_from, effective_until, created_at, updated_at
		FROM vip_payment_order
		WHERE user_id = ? AND status = 'paid'
		ORDER BY COALESCE(paid_at, created_at) DESC, id DESC
		LIMIT 1
	`, userID)

	var item model.AdminOrder
	var paidAt sql.NullTime
	var expiresAt sql.NullTime
	var effectiveFrom sql.NullTime
	var effectiveUntil sql.NullTime
	if err := row.Scan(&item.ID, &item.OrderID, &item.UserID, &item.OpenID, &item.ProductID, &item.ProductName, &item.Content, &item.AmountFen, &item.Status, &item.PaymentChannel, &item.PrepayID, &item.TransactionID, &item.Remark, &paidAt, &expiresAt, &effectiveFrom, &effectiveUntil, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	if paidAt.Valid {
		item.PaidAt = &paidAt.Time
	}
	if expiresAt.Valid {
		item.ExpiresAt = &expiresAt.Time
	}
	if effectiveFrom.Valid {
		item.EffectiveFrom = &effectiveFrom.Time
	}
	if effectiveUntil.Valid {
		item.EffectiveUntil = &effectiveUntil.Time
	}
	return &item, nil
}

func (r *AdminRepository) InferVIPMembership(order model.AdminOrder, product *model.VIPProductConfig, now time.Time) model.VIPMembershipStatusResponse {
	startTime := order.CreatedAt
	if order.PaidAt != nil && !order.PaidAt.IsZero() {
		startTime = *order.PaidAt
	}
	if order.EffectiveFrom != nil && !order.EffectiveFrom.IsZero() {
		startTime = *order.EffectiveFrom
	}

	validityType := ""
	validTimes := 0
	var validFrom *time.Time
	var validUntil *time.Time
	productName := strings.TrimSpace(order.ProductName)
	if product != nil {
		if productName == "" {
			productName = strings.TrimSpace(product.Name)
		}
		validityType = strings.TrimSpace(product.ValidityType)
		validTimes = product.ValidTimes
		validFrom = product.ValidFrom
		validUntil = product.ValidUntil
	}

	if validityType == "" || validityType == "unlimited" {
		switch strings.TrimSpace(order.ProductID) {
		case "vip_single":
			validityType = "times"
			if validTimes <= 0 {
				validTimes = 1
			}
		case "vip_day":
			validityType = "range"
			if validFrom == nil {
				validFrom = &startTime
			}
			if validUntil == nil {
				end := startTime.Add(24 * time.Hour)
				validUntil = &end
			}
		case "vip_month":
			validityType = "range"
			if validFrom == nil {
				validFrom = &startTime
			}
			if validUntil == nil {
				end := startTime.AddDate(0, 0, 30)
				validUntil = &end
			}
		case "vip_season":
			validityType = "range"
			if validFrom == nil {
				validFrom = &startTime
			}
			if validUntil == nil {
				end := startTime.AddDate(0, 0, 90)
				validUntil = &end
			}
		}
	}

	if validityType == "" {
		validityType = "unlimited"
	}

	if validFrom != nil && !validFrom.IsZero() {
		startTime = *validFrom
	}
	if order.EffectiveUntil != nil && !order.EffectiveUntil.IsZero() {
		validUntil = order.EffectiveUntil
	}

	status := model.VIPMembershipStatusResponse{
		Active:      true,
		OrderID:     strings.TrimSpace(order.OrderID),
		ProductID:   strings.TrimSpace(order.ProductID),
		ProductName: productName,
		LevelType:   validityType,
		StartAt:     startTime.UnixMilli(),
		PaidAt:      startTime.UnixMilli(),
		StartText:   startTime.Format("2006-01-02 15:04"),
	}
	if order.PaidAt != nil && !order.PaidAt.IsZero() {
		status.PaidAt = order.PaidAt.UnixMilli()
	}

	switch validityType {
	case "times":
		if validTimes <= 0 {
			validTimes = 1
		}
		status.LevelText = "单次"
		status.StatusText = "单次会员"
		status.ValidityText = fmt.Sprintf("%d 次有效", validTimes)
		status.EndText = fmt.Sprintf("剩余 %d 次内有效", validTimes)
	case "range":
		status.LevelText = "长期"
		status.StatusText = "长期会员"
		if validUntil != nil && !validUntil.IsZero() {
			status.EndAt = validUntil.UnixMilli()
			status.EndText = validUntil.Format("2006-01-02 15:04")
			status.ValidityText = fmt.Sprintf("%s 至 %s", status.StartText, status.EndText)
			if validUntil.Before(now) {
				status.Active = false
				status.StatusText = "已到期"
			}
		} else {
			status.EndText = "长期有效"
			status.ValidityText = "长期有效"
		}
	default:
		status.LevelText = "长期"
		status.StatusText = "长期会员"
		status.ValidityText = "长期有效"
		status.EndText = "长期有效"
	}

	if status.ProductName == "" {
		status.ProductName = strings.TrimSpace(order.ProductID)
	}
	return status
}

func (r *AdminRepository) SaveOrder(ctx context.Context, item model.AdminOrder) (int, error) {
	if item.ID > 0 {
		_, err := r.db.ExecContext(ctx, `
			UPDATE vip_payment_order
			SET order_id = ?, user_id = ?, openid = ?, product_id = ?, product_name = ?, content = ?, amount_fen = ?,
				status = ?, payment_channel = ?, prepay_id = ?, transaction_id = ?, remark = ?, paid_at = ?, expires_at = ?, effective_from = ?, effective_until = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, strings.TrimSpace(item.OrderID), item.UserID, strings.TrimSpace(item.OpenID), strings.TrimSpace(item.ProductID), strings.TrimSpace(item.ProductName), strings.TrimSpace(item.Content), item.AmountFen, strings.TrimSpace(item.Status), strings.TrimSpace(item.PaymentChannel), strings.TrimSpace(item.PrepayID), strings.TrimSpace(item.TransactionID), strings.TrimSpace(item.Remark), item.PaidAt, item.ExpiresAt, item.EffectiveFrom, item.EffectiveUntil, item.ID)
		return item.ID, err
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO vip_payment_order (
			order_id, user_id, openid, product_id, product_name, content, amount_fen,
				status, payment_channel, prepay_id, transaction_id, remark, paid_at, expires_at, effective_from, effective_until
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, strings.TrimSpace(item.OrderID), item.UserID, strings.TrimSpace(item.OpenID), strings.TrimSpace(item.ProductID), strings.TrimSpace(item.ProductName), strings.TrimSpace(item.Content), item.AmountFen, strings.TrimSpace(item.Status), strings.TrimSpace(item.PaymentChannel), strings.TrimSpace(item.PrepayID), strings.TrimSpace(item.TransactionID), strings.TrimSpace(item.Remark), item.PaidAt, item.ExpiresAt, item.EffectiveFrom, item.EffectiveUntil)
	if err != nil {
		return 0, err
	}
	id, _ := result.LastInsertId()
	return int(id), nil
}

func (r *AdminRepository) GetPaymentOrderByOrderID(ctx context.Context, orderID string) (*model.AdminOrder, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, order_id, user_id, openid, product_id, product_name, content, amount_fen,
			status, payment_channel, prepay_id, transaction_id, remark, paid_at, expires_at, effective_from, effective_until, created_at, updated_at
		FROM vip_payment_order
		WHERE order_id = ?
		LIMIT 1
	`, strings.TrimSpace(orderID))
	var item model.AdminOrder
	var paidAt sql.NullTime
	var expiresAt sql.NullTime
	var effectiveFrom sql.NullTime
	var effectiveUntil sql.NullTime
	if err := row.Scan(&item.ID, &item.OrderID, &item.UserID, &item.OpenID, &item.ProductID, &item.ProductName, &item.Content, &item.AmountFen, &item.Status, &item.PaymentChannel, &item.PrepayID, &item.TransactionID, &item.Remark, &paidAt, &expiresAt, &effectiveFrom, &effectiveUntil, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	if paidAt.Valid {
		item.PaidAt = &paidAt.Time
	}
	if expiresAt.Valid {
		item.ExpiresAt = &expiresAt.Time
	}
	if effectiveFrom.Valid {
		item.EffectiveFrom = &effectiveFrom.Time
	}
	if effectiveUntil.Valid {
		item.EffectiveUntil = &effectiveUntil.Time
	}
	return &item, nil
}

func (r *AdminRepository) CloseExpiredCreatedOrders(ctx context.Context, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE vip_payment_order
		SET status = 'closed',
			expires_at = COALESCE(expires_at, DATE_ADD(created_at, INTERVAL 10 MINUTE)),
			remark = CASE WHEN COALESCE(remark, '') = '' THEN '订单超时自动关闭' ELSE remark END,
			updated_at = CURRENT_TIMESTAMP
		WHERE status = 'created' AND COALESCE(expires_at, DATE_ADD(created_at, INTERVAL 10 MINUTE)) <= ?
	`, now)
	return err
}

func (r *AdminRepository) ListVIPProducts(ctx context.Context) ([]model.VIPProductConfig, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.id, p.product_id, p.name, p.description, p.amount_fen, p.enabled,
			COALESCE(p.validity_type, 'unlimited'), COALESCE(p.valid_times, 0), p.valid_from, p.valid_until,
			COALESCE(o.order_count, 0), p.sort_order, p.created_at, p.updated_at
		FROM vip_product_config p
		LEFT JOIN (
			SELECT product_id, COUNT(*) AS order_count
			FROM vip_payment_order
			GROUP BY product_id
		) o ON o.product_id = p.product_id
		ORDER BY p.sort_order ASC, p.id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.VIPProductConfig, 0)
	for rows.Next() {
		var item model.VIPProductConfig
		var validFrom sql.NullTime
		var validUntil sql.NullTime
		if err := rows.Scan(&item.ID, &item.ProductID, &item.Name, &item.Description, &item.AmountFen, &item.Enabled, &item.ValidityType, &item.ValidTimes, &validFrom, &validUntil, &item.OrderCount, &item.SortOrder, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if validFrom.Valid {
			item.ValidFrom = &validFrom.Time
		}
		if validUntil.Valid {
			item.ValidUntil = &validUntil.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *AdminRepository) GetVIPProductByProductID(ctx context.Context, productID string) (*model.VIPProductConfig, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT p.id, p.product_id, p.name, p.description, p.amount_fen, p.enabled,
			COALESCE(p.validity_type, 'unlimited'), COALESCE(p.valid_times, 0), p.valid_from, p.valid_until,
			COALESCE(o.order_count, 0), p.sort_order, p.created_at, p.updated_at
		FROM vip_product_config p
		LEFT JOIN (
			SELECT product_id, COUNT(*) AS order_count
			FROM vip_payment_order
			GROUP BY product_id
		) o ON o.product_id = p.product_id
		WHERE p.product_id = ?
	`, strings.TrimSpace(productID))
	var item model.VIPProductConfig
	var validFrom sql.NullTime
	var validUntil sql.NullTime
	if err := row.Scan(&item.ID, &item.ProductID, &item.Name, &item.Description, &item.AmountFen, &item.Enabled, &item.ValidityType, &item.ValidTimes, &validFrom, &validUntil, &item.OrderCount, &item.SortOrder, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	if validFrom.Valid {
		item.ValidFrom = &validFrom.Time
	}
	if validUntil.Valid {
		item.ValidUntil = &validUntil.Time
	}
	return &item, nil
}

func (r *AdminRepository) SaveVIPProduct(ctx context.Context, item model.VIPProductConfig) (int, error) {
	validityType := strings.TrimSpace(item.ValidityType)
	if validityType == "" {
		validityType = "unlimited"
	}
	if item.ID > 0 {
		_, err := r.db.ExecContext(ctx, `
			UPDATE vip_product_config SET product_id = ?, name = ?, description = ?, amount_fen = ?, enabled = ?, validity_type = ?, valid_times = ?, valid_from = ?, valid_until = ?, sort_order = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
		`, item.ProductID, item.Name, item.Description, item.AmountFen, item.Enabled, validityType, item.ValidTimes, item.ValidFrom, item.ValidUntil, item.SortOrder, item.ID)
		return item.ID, err
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO vip_product_config (product_id, name, description, amount_fen, enabled, validity_type, valid_times, valid_from, valid_until, sort_order)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, item.ProductID, item.Name, item.Description, item.AmountFen, item.Enabled, validityType, item.ValidTimes, item.ValidFrom, item.ValidUntil, item.SortOrder)
	if err != nil {
		return 0, err
	}
	id, _ := result.LastInsertId()
	return int(id), nil
}

func (r *AdminRepository) GetVIPEntryControlConfig(ctx context.Context) (*model.VIPEntryControlConfig, error) {
	var item model.VIPEntryControlConfig
	if err := r.db.QueryRowContext(ctx, `
		SELECT show_vip_entry, updated_at
		FROM vip_entry_control_config
		WHERE id = 1
	`).Scan(&item.ShowVIPEntry, &item.UpdatedAt); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *AdminRepository) SaveVIPEntryControlConfig(ctx context.Context, showVIPEntry bool) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO vip_entry_control_config (id, show_vip_entry, updated_at)
		VALUES (1, ?, CURRENT_TIMESTAMP)
		ON DUPLICATE KEY UPDATE show_vip_entry = VALUES(show_vip_entry), updated_at = CURRENT_TIMESTAMP
	`, showVIPEntry)
	return err
}

func (r *AdminRepository) ShouldShowVIPEntry(ctx context.Context) (bool, error) {
	item, err := r.GetVIPEntryControlConfig(ctx)
	if err != nil {
		return false, err
	}
	return item.ShowVIPEntry, nil
}

func (r *AdminRepository) GetShareGateControlConfig(ctx context.Context) (*model.ShareGateControlConfig, error) {
	var item model.ShareGateControlConfig
	if err := r.db.QueryRowContext(ctx, `
		SELECT require_share_for_ai_report, require_share_for_college_major, require_share_for_recommend_result, require_share_for_plan_compare, updated_at
		FROM share_gate_control_config
		WHERE id = 1
	`).Scan(&item.RequireShareForAIReport, &item.RequireShareForCollegeMajor, &item.RequireShareForRecommendResult, &item.RequireShareForPlanCompare, &item.UpdatedAt); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *AdminRepository) SaveShareGateControlConfig(ctx context.Context, requireShareForAIReport, requireShareForCollegeMajor, requireShareForRecommendResult, requireShareForPlanCompare bool) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO share_gate_control_config (
			id,
			require_share_for_ai_report,
			require_share_for_college_major,
			require_share_for_recommend_result,
			require_share_for_plan_compare,
			updated_at
		)
		VALUES (1, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON DUPLICATE KEY UPDATE
			require_share_for_ai_report = VALUES(require_share_for_ai_report),
			require_share_for_college_major = VALUES(require_share_for_college_major),
			require_share_for_recommend_result = VALUES(require_share_for_recommend_result),
			require_share_for_plan_compare = VALUES(require_share_for_plan_compare),
			updated_at = CURRENT_TIMESTAMP
	`, requireShareForAIReport, requireShareForCollegeMajor, requireShareForRecommendResult, requireShareForPlanCompare)
	return err
}

func (r *AdminRepository) GetShareGateConfig(ctx context.Context) (*model.ShareGateConfigResponse, error) {
	item, err := r.GetShareGateControlConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &model.ShareGateConfigResponse{
		RequireShareForAIReport:        item.RequireShareForAIReport,
		RequireShareForCollegeMajor:    item.RequireShareForCollegeMajor,
		RequireShareForRecommendResult: item.RequireShareForRecommendResult,
		RequireShareForPlanCompare:     item.RequireShareForPlanCompare,
	}, nil
}

func (r *AdminRepository) SetVIPProductEnabled(ctx context.Context, id int, enabled bool) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE vip_product_config SET enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, enabled, id)
	return err
}

func (r *AdminRepository) DeleteVIPProduct(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM vip_product_config WHERE id = ?`, id)
	return err
}

func compactStudent(raw string) string {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return raw
	}
	parts := make([]string, 0, 4)
	for _, key := range []string{"province", "subject", "score", "rank", "targetMajor"} {
		if value, ok := parsed[key]; ok {
			parts = append(parts, fmt.Sprintf("%s:%v", key, value))
		}
	}
	if len(parts) == 0 {
		return raw
	}
	return strings.Join(parts, " | ")
}

func (r *AdminRepository) columnExists(ctx context.Context, tableName, columnName string) (bool, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?
	`, strings.TrimSpace(tableName), strings.TrimSpace(columnName)).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *AdminRepository) ensureMiniAuthUserObjectIDs(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, `SELECT id FROM mini_auth_user WHERE COALESCE(object_id, '') = '' ORDER BY id ASC`)
	if err != nil {
		return err
	}
	defer rows.Close()
	legacyIDs := make([]int, 0)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return err
		}
		legacyIDs = append(legacyIDs, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, legacyID := range legacyIDs {
		objectID, err := newObjectID()
		if err != nil {
			return err
		}
		if _, err := r.db.ExecContext(ctx, `UPDATE mini_auth_user SET object_id = ? WHERE id = ? AND COALESCE(object_id, '') = ''`, objectID, legacyID); err != nil {
			return err
		}
	}
	indexExists, err := r.indexExists(ctx, "mini_auth_user", "uq_mini_auth_user_object_id")
	if err != nil {
		return err
	}
	if indexExists {
		return nil
	}
	if _, err := r.db.ExecContext(ctx, `CREATE UNIQUE INDEX uq_mini_auth_user_object_id ON mini_auth_user (object_id)`); err != nil {
		return err
	}
	return nil
}

func (r *AdminRepository) ensureVIPPaymentOrderUserIDObjectIDs(ctx context.Context) error {
	typeValue, err := r.columnType(ctx, "vip_payment_order", "user_id")
	if err != nil {
		return err
	}
	if !strings.Contains(typeValue, "char") && !strings.Contains(typeValue, "text") {
		if _, err := r.db.ExecContext(ctx, `ALTER TABLE vip_payment_order MODIFY COLUMN user_id VARCHAR(24) NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE vip_payment_order o
		JOIN mini_auth_user u ON CONVERT(u.id, CHAR(24)) COLLATE utf8mb4_unicode_ci = o.user_id COLLATE utf8mb4_unicode_ci
		SET o.user_id = u.object_id
		WHERE o.user_id <> '' AND u.object_id <> '' AND o.user_id <> u.object_id
	`)
	return err
}

func (r *AdminRepository) columnType(ctx context.Context, tableName, columnName string) (string, error) {
	var dataType string
	if err := r.db.QueryRowContext(ctx, `
		SELECT LOWER(COALESCE(data_type, ''))
		FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?
	`, strings.TrimSpace(tableName), strings.TrimSpace(columnName)).Scan(&dataType); err != nil {
		return "", err
	}
	return strings.TrimSpace(dataType), nil
}

func (r *AdminRepository) indexExists(ctx context.Context, tableName, indexName string) (bool, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM information_schema.statistics
		WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?
	`, strings.TrimSpace(tableName), strings.TrimSpace(indexName)).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}
