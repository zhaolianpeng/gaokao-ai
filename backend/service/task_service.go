package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gaokao-ai/backend/logging"
	"gaokao-ai/backend/model"
	"gaokao-ai/backend/repository"
)

type TaskService struct {
	repo      *repository.TaskRepository
	aiService *AIService
}

func NewTaskService(repo *repository.TaskRepository, aiService *AIService) *TaskService {
	return &TaskService{repo: repo, aiService: aiService}
}

func (s *TaskService) SubmitAgentRecommend(ctx context.Context, req model.AgentRecommendRequest) (string, string, error) {
	studentJSON, err := json.Marshal(req.Student)
	if err != nil {
		return "", "", err
	}
	templatesJSON, err := json.Marshal(req.Templates)
	if err != nil {
		return "", "", err
	}
	taskID, err := s.repo.CreateTask(ctx, "AI 智能体报考建议", studentJSON, templatesJSON, []byte(`{}`), req.Demand, "agent-recommend")
	if err != nil {
		return "", "", err
	}
	logging.LogEvent("task_submit", map[string]any{"taskId": taskID, "taskType": "agent-recommend", "student": req.Student, "demandPreview": logging.PreviewString(req.Demand, 512), "templateCount": len(req.Templates)})
	go s.runAgentRecommend(taskID, req)
	return strconv.Itoa(taskID), "pending", nil
}

func (s *TaskService) SubmitAnalyzeTask(ctx context.Context, req model.AIAnalyzeRequest) (string, string, error) {
	student := model.TaskStudent{
		Province:     req.Student.Province,
		Subject:      req.Student.Subject,
		AnalysisYear: strconv.Itoa(req.Student.Year),
		Score:        req.Student.Score,
		Rank:         req.Student.Rank,
		TargetMajor:  req.Student.TargetMajor,
		Notes:        req.Student.Notes,
		Year:         req.Student.Year,
	}
	studentJSON, err := json.Marshal(student)
	if err != nil {
		return "", "", err
	}
	recommendJSON, err := json.Marshal(req.Recommend)
	if err != nil {
		return "", "", err
	}
	taskID, err := s.repo.CreateTask(ctx, "AI 志愿分析报告", studentJSON, []byte(`[]`), recommendJSON, req.ExtraNotes, "analyze")
	if err != nil {
		return "", "", err
	}
	logging.LogEvent("task_submit", map[string]any{"taskId": taskID, "taskType": "analyze", "student": student, "extraNotesPreview": logging.PreviewString(req.ExtraNotes, 512), "chongCount": len(req.Recommend.Chong), "wenCount": len(req.Recommend.Wen), "baoCount": len(req.Recommend.Bao)})
	go s.runAnalyzeTask(taskID, req)
	return strconv.Itoa(taskID), "pending", nil
}

func (s *TaskService) GetTaskStatus(ctx context.Context, taskID string) (*model.TaskStatusResponse, error) {
	parsedID, err := strconv.Atoi(strings.TrimSpace(taskID))
	if err != nil || parsedID <= 0 {
		return nil, fmt.Errorf("invalid task id")
	}
	record, err := s.repo.GetTask(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	response := &model.TaskStatusResponse{
		TaskID:       strconv.Itoa(record.ID),
		Title:        record.Title,
		Status:       record.Status,
		Ready:        record.Status == "succeeded",
		Failed:       record.Status == "failed",
		ErrorMessage: record.ErrorMessage,
		Report:       record.Report,
		Provider:     record.Provider,
	}
	if len(record.StudentJSON) > 0 {
		var student model.TaskStudent
		if err := json.Unmarshal(record.StudentJSON, &student); err == nil {
			response.Student = &student
		}
	}
	if len(record.SuggestionsJSON) > 0 {
		var suggestions []model.AgentSuggestion
		if err := json.Unmarshal(record.SuggestionsJSON, &suggestions); err == nil {
			response.Suggestions = suggestions
		}
	}
	return response, nil
}

func (s *TaskService) runAgentRecommend(taskID int, req model.AgentRecommendRequest) {
	ctx := context.Background()
	logging.LogEvent("task_run_start", map[string]any{"taskId": taskID, "taskType": "agent-recommend"})
	if err := s.repo.MarkProcessing(ctx, taskID); err != nil {
		log.Printf("mark agent task processing failed: %v", err)
		logging.LogEvent("task_run_error", map[string]any{"taskId": taskID, "taskType": "agent-recommend", "step": "mark_processing", "error": err.Error()})
		return
	}
	suggestions := buildExploreSuggestions(req.Student, req.Demand, req.Templates)
	report, provider := s.buildAgentReport(ctx, req)
	suggestionsJSON, _ := json.Marshal(suggestions)
	logging.LogEvent("task_compute", map[string]any{"taskId": taskID, "taskType": "agent-recommend", "suggestionCount": len(suggestions), "provider": provider, "reportPreview": logging.PreviewString(report, 512), "reportLength": len(report)})
	if err := s.repo.CompleteTask(ctx, taskID, report, provider, suggestionsJSON); err != nil {
		log.Printf("complete agent task failed: %v", err)
		logging.LogEvent("task_run_error", map[string]any{"taskId": taskID, "taskType": "agent-recommend", "step": "complete", "error": err.Error()})
		return
	}
	logging.LogEvent("task_run_complete", map[string]any{"taskId": taskID, "taskType": "agent-recommend", "provider": provider, "suggestionCount": len(suggestions), "reportLength": len(report)})
}

func (s *TaskService) runAnalyzeTask(taskID int, req model.AIAnalyzeRequest) {
	ctx := context.Background()
	logging.LogEvent("task_run_start", map[string]any{"taskId": taskID, "taskType": "analyze"})
	if err := s.repo.MarkProcessing(ctx, taskID); err != nil {
		log.Printf("mark analyze task processing failed: %v", err)
		logging.LogEvent("task_run_error", map[string]any{"taskId": taskID, "taskType": "analyze", "step": "mark_processing", "error": err.Error()})
		return
	}
	report, err := s.aiService.Analyze(ctx, req)
	if err != nil {
		if failErr := s.repo.FailTask(ctx, taskID, err.Error()); failErr != nil {
			log.Printf("fail analyze task failed: %v", failErr)
			logging.LogEvent("task_run_error", map[string]any{"taskId": taskID, "taskType": "analyze", "step": "mark_failed", "error": failErr.Error()})
		}
		logging.LogEvent("task_run_error", map[string]any{"taskId": taskID, "taskType": "analyze", "step": "analyze", "error": err.Error()})
		return
	}
	if err := s.repo.CompleteTask(ctx, taskID, report, providerName(report), []byte(`[]`)); err != nil {
		log.Printf("complete analyze task failed: %v", err)
		logging.LogEvent("task_run_error", map[string]any{"taskId": taskID, "taskType": "analyze", "step": "complete", "error": err.Error()})
		return
	}
	logging.LogEvent("task_compute", map[string]any{"taskId": taskID, "taskType": "analyze", "provider": providerName(report), "reportPreview": logging.PreviewString(report, 512), "reportLength": len(report)})
	logging.LogEvent("task_run_complete", map[string]any{"taskId": taskID, "taskType": "analyze", "provider": providerName(report), "reportLength": len(report)})
}

func (s *TaskService) buildAgentReport(ctx context.Context, req model.AgentRecommendRequest) (string, string) {
	prompt := buildAgentPrompt(req.Student, req.Demand, req.Templates)
	logging.LogEvent("task_prompt", map[string]any{"taskType": "agent-recommend", "promptPreview": logging.PreviewString(prompt, 512), "promptLength": len(prompt)})
	if !s.aiService.HasAPIKey() {
		logging.LogEvent("task_fallback", map[string]any{"taskType": "agent-recommend", "reason": "missing_api_key"})
		return buildLocalAgentAdvice(req.Student, req.Demand, req.Templates, "missing_key"), "local"
	}
	report, err := s.aiService.GenerateText(ctx, "你是中国高考志愿填报智能体，擅长将考生需求转化为可执行的志愿策略。", prompt, 0.4)
	if err != nil {
		logging.LogEvent("task_fallback", map[string]any{"taskType": "agent-recommend", "reason": "generate_failed", "error": err.Error()})
		return buildLocalAgentAdvice(req.Student, req.Demand, req.Templates, "timeout"), "local-fallback"
	}
	return report, "deepseek"
}

func providerName(report string) string {
	if strings.Contains(report, "未配置 DEEPSEEK_API_KEY") || strings.Contains(report, "本地模板报告") {
		return "local"
	}
	return "deepseek"
}

func buildAgentPrompt(student model.TaskStudent, demand string, templates []string) string {
	yearText := student.AnalysisYear
	if strings.TrimSpace(yearText) == "" && student.Year > 0 {
		yearText = strconv.Itoa(student.Year)
	}
	if strings.TrimSpace(yearText) == "" {
		yearText = "2025"
	}
	filledInfo := []string{
		fmt.Sprintf("省份：%s", defaultString(student.Province, "黑龙江")),
		fmt.Sprintf("科类：%s", defaultString(student.Subject, "未填写")),
		fmt.Sprintf("查询年份：%s", yearText),
		fmt.Sprintf("分数：%s", formatFilledNumber(student.Score)),
		fmt.Sprintf("排名：%s", formatFilledNumber(student.Rank)),
		fmt.Sprintf("意向专业：%s", defaultString(student.TargetMajor, "未填写")),
		fmt.Sprintf("补充偏好：%s", defaultString(student.Notes, "未填写")),
	}
	templateText := "无"
	if len(templates) > 0 {
		lines := make([]string, 0, len(templates))
		for _, item := range templates {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				lines = append(lines, "- "+trimmed)
			}
		}
		if len(lines) > 0 {
			templateText = strings.Join(lines, "\n")
		}
	}
	return "你现在要作为黑龙江高考志愿智能体，为考生输出一份“需求驱动”的报考分析。\n\n以下是用户当前已经填写的信息，请全部纳入分析，不要忽略：\n" + strings.Join(filledInfo, "\n") + "\n\n用户本次选择/使用的常用需求模板：\n" + templateText + "\n\n本次用户输入的核心需求：\n" + demand + "\n\n请直接输出一份可执行建议，必须包含：\n1. 先用 2-4 句话总结这位考生当前最适合的报考方向\n2. 从城市、学校层次、专业方向、调剂接受度四个维度拆解用户需求\n3. 给出“优先级排序建议”，说明哪些条件必须优先，哪些条件需要妥协\n4. 给出冲稳保三档策略，但用自然语言描述，不要求列具体学校名单\n5. 给出后续操作建议，明确下一步应该去院校库重点查什么，最好点出 3-5 个可检索关键词\n6. 如果用户需求本身互相冲突，要明确指出冲突点和取舍方式\n\n请用中文输出，结构清晰，避免空话套话。"
}

func buildLocalAgentAdvice(student model.TaskStudent, demand string, templates []string, fallbackReason string) string {
	text := strings.ToLower(strings.Join([]string{student.TargetMajor, student.Notes, demand, strings.Join(templates, " ")}, " "))
	wantsHarbin := strings.Contains(text, "哈尔滨")
	wantsInsideProvince := strings.Contains(text, "省内") || strings.Contains(text, "黑龙江")
	wantsPublic := strings.Contains(text, "公办")
	acceptsAdjustment := strings.Contains(text, "调剂")
	wants211 := strings.Contains(text, "211") || strings.Contains(text, "双一流")
	majorDirection := student.TargetMajor
	if strings.TrimSpace(majorDirection) == "" {
		if regexp.MustCompile("计算机|软件|电子信息").MatchString(text) {
			majorDirection = "计算机/电子信息方向"
		} else {
			majorDirection = "未明确具体专业方向"
		}
	}
	keywords := make([]string, 0, 5)
	if wantsHarbin {
		keywords = append(keywords, "哈尔滨")
	}
	if wantsPublic {
		keywords = append(keywords, "公办")
	}
	if strings.Contains(text, "计算机") || strings.Contains(text, "软件") {
		keywords = append(keywords, "计算机")
	}
	if strings.Contains(text, "电子信息") {
		keywords = append(keywords, "电子信息")
	}
	if wants211 {
		keywords = append(keywords, "211")
	}
	if len(keywords) == 0 && strings.TrimSpace(student.TargetMajor) != "" {
		keywords = append(keywords, student.TargetMajor)
	}
	keywords = uniqueStrings(keywords)
	if len(keywords) > 5 {
		keywords = keywords[:5]
	}
	summary := "当前需求更适合先按学校层次和专业方向做第一轮筛选。"
	if wantsHarbin {
		summary = "当前需求明显偏向“城市优先”，哈尔滨应当被放在第一筛选层。"
	}
	publicSummary := "你没有把公办设为硬条件，后续可以把学校层次和专业匹配放在更前面。"
	if wantsPublic {
		publicSummary = "你已经明确偏向公办院校，这会直接压缩可选范围，但能提高结果稳定性。"
	}
	adjustSummary := "你没有明确接受调剂，后续需要重点审查专业组内专业跨度。"
	if acceptsAdjustment {
		adjustSummary = "你接受组内调剂，这对保住城市或学校层次是有帮助的。"
	}
	conflict := "当前需求没有绝对冲突，但城市、层次、专业三者不一定能同时最优，需要接受局部妥协。"
	if wantsHarbin && wants211 && strings.Contains(student.Subject, "历史") {
		conflict = "“优先哈尔滨”与“冲 211/双一流”同时成立时，历史类下可选面会明显变窄，需要在城市和层次之间做取舍。"
	} else if wantsHarbin && wantsPublic {
		conflict = "“优先哈尔滨”与“优先公办”可以同时成立，但会抬高筛选门槛，热门专业可能需要接受专业让步。"
	}
	fallbackIntro := "以下内容为本地模板建议。当前先返回一份结构化可执行分析。"
	if fallbackReason == "missing_key" {
		fallbackIntro = "以下内容为本地模板建议。当前服务未配置 DEEPSEEK_API_KEY，所以先返回一份结构化可执行分析。"
	} else if fallbackReason == "timeout" {
		fallbackIntro = "以下内容为本地模板建议。当前智能体生成时间较长，先返回一份可执行分析；你也可以稍后继续重试获取更细化结果。"
	}
	lines := []string{
		fallbackIntro,
		"",
		"## 一、总体判断",
		fmt.Sprintf("黑龙江 %s %s 口径下，你当前已填写的信息是：%s、%s、意向专业 %s。", defaultString(student.Subject, "未填写科类"), defaultString(yearOrAnalysisYear(student), "2025"), scoreText(student.Score), rankText(student.Rank), defaultString(student.TargetMajor, "未填写")),
		summary,
		publicSummary,
		adjustSummary,
		"",
		"## 二、需求拆解",
		fmt.Sprintf("城市维度：%s", cityAdvice(wantsHarbin, wantsInsideProvince)),
		fmt.Sprintf("学校层次：%s", schoolAdvice(wants211, wantsPublic)),
		fmt.Sprintf("专业方向：当前重点应放在 %s，同时留意相近替代方向，如软件工程、网络空间安全、电子信息类。", majorDirection),
		fmt.Sprintf("调剂接受度：%s", adjustAdvice(acceptsAdjustment)),
		"",
		"## 三、优先级排序建议",
		fmt.Sprintf("第一优先级：%s", ternary(wantsHarbin, "城市与办学属性", "专业方向与学校层次")),
		fmt.Sprintf("第二优先级：%s", ternary(wantsPublic, "公办属性与学费可接受度", "城市偏好与调剂接受度")),
		fmt.Sprintf("第三优先级：%s", ternary(acceptsAdjustment, "组内专业结构是否可接受", "是否需要为了保专业而牺牲城市或层次")),
		"建议不要把所有条件都设成硬门槛，否则结果会过窄。",
		"",
		"## 四、冲稳保策略",
		"冲刺：把城市、层次、专业三者中最看重的两项锁死，第三项允许有限妥协，用来尝试更高层次目标。",
		"稳妥：优先保住最核心需求，例如哈尔滨 + 公办，或者公办 + 计算机方向，再在这个范围里找匹配度更高的组。",
		"保底：城市、层次、专业三项里至少放松一项，重点保可录取性和可接受的组内专业结构。",
		"",
		"## 五、下一步怎么查",
		"建议先去院校库逐个检索这些关键词，并对比组内专业结构、最低位次和是否接受调剂：",
	}
	if len(keywords) == 0 {
		keywords = []string{"哈尔滨", "公办", "计算机", "电子信息"}
	}
	for index, keyword := range keywords {
		lines = append(lines, fmt.Sprintf("%d. %s", index+1, keyword))
	}
	lines = append(lines,
		"",
		"## 六、需求冲突与取舍",
		conflict,
		"如果后续配置好 DeepSeek Key，这一页会返回更细化、更贴近你输入语义的分析结果。",
	)
	return strings.Join(lines, "\n")
}

func buildExploreSuggestions(student model.TaskStudent, demand string, templates []string) []model.AgentSuggestion {
	suggestions := make([]model.AgentSuggestion, 0, 5)
	push := func(title, keyword string) {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			return
		}
		suggestions = append(suggestions, model.AgentSuggestion{
			ID:      title + "-" + keyword,
			Title:   title,
			Keyword: keyword,
			Subject: defaultString(student.Subject, "历史"),
		})
	}
	if strings.TrimSpace(student.TargetMajor) != "" {
		push("按意向专业筛选", student.TargetMajor)
	}
	combinedText := student.Notes + " " + demand
	if strings.Contains(combinedText, "哈尔滨") {
		push("优先哈尔滨", "哈尔滨")
	}
	if regexp.MustCompile("计算机|软件|电子信息").MatchString(combinedText) {
		if strings.Contains(combinedText, "计算机") || strings.Contains(combinedText, "软件") {
			push("查看计算机方向", "计算机")
		}
		if strings.Contains(combinedText, "电子信息") {
			push("查看电子信息方向", "电子信息")
		}
	}
	for _, template := range templates {
		switch strings.TrimSpace(template) {
		case "留省内":
			push("留省内", "黑龙江")
		case "优先哈尔滨":
			push("优先哈尔滨", "哈尔滨")
		case "优先公办":
			push("优先公办", "公办")
		case "冲 211":
			push("冲 211", "211")
		case "偏计算机":
			push("偏计算机", "计算机")
		}
	}
	seen := make(map[string]model.AgentSuggestion, len(suggestions))
	for _, item := range suggestions {
		seen[item.ID] = item
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]model.AgentSuggestion, 0, len(keys))
	for _, key := range keys {
		result = append(result, seen[key])
		if len(result) >= 5 {
			break
		}
	}
	return result
}

func yearOrAnalysisYear(student model.TaskStudent) string {
	if strings.TrimSpace(student.AnalysisYear) != "" {
		return student.AnalysisYear
	}
	if student.Year > 0 {
		return strconv.Itoa(student.Year)
	}
	return ""
}

func formatFilledNumber(value int) string {
	if value <= 0 {
		return "未填写"
	}
	return strconv.Itoa(value)
}

func scoreText(score int) string {
	if score <= 0 {
		return "未填写分数"
	}
	return strconv.Itoa(score) + " 分"
}

func rankText(rank int) string {
	if rank <= 0 {
		return "未填写排名"
	}
	return strconv.Itoa(rank) + " 名"
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func cityAdvice(wantsHarbin, wantsInsideProvince bool) string {
	if wantsHarbin {
		return "优先哈尔滨，应先在院校库用“哈尔滨”做首轮筛选。"
	}
	if wantsInsideProvince {
		return "优先黑龙江省内，可先看省内院校，再做城市细分。"
	}
	return "城市没有形成硬约束，可作为第二优先级。"
}

func schoolAdvice(wants211, wantsPublic bool) string {
	if wants211 {
		return "有明显冲层次诉求，建议把 211/双一流作为冲刺方向，而不是全部志愿的统一标准。"
	}
	if wantsPublic {
		return "优先公办是硬条件，应先排除高收费和民办项目。"
	}
	return "学校层次可以作为筛选条件之一，但不必压过专业匹配。"
}

func adjustAdvice(acceptsAdjustment bool) string {
	if acceptsAdjustment {
		return "接受组内调剂，适合优先保城市或学校层次。"
	}
	return "不接受调剂或未明确接受调剂，填报时要重点核查组内专业构成。"
}

func ternary(condition bool, left, right string) string {
	if condition {
		return left
	}
	return right
}
