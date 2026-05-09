package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"gaokao-ai/backend/model"
)

type AIService struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewAIService(apiKey, baseURL string, timeout time.Duration) *AIService {
	if timeout <= 0 {
		timeout = 120 * time.Second
	}

	return &AIService{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (s *AIService) Analyze(ctx context.Context, req model.AIAnalyzeRequest) (string, error) {
	prompt := buildPrompt(req)
	if !s.HasAPIKey() {
		return "当前服务未配置 DEEPSEEK_API_KEY，先返回本地模板报告。\n\n" + prompt, nil
	}
	return s.GenerateText(ctx, "你是中国高考志愿填报专家。", prompt, 0.3)
}

func (s *AIService) HasAPIKey() bool {
	return strings.TrimSpace(s.apiKey) != ""
}

func (s *AIService) GenerateText(ctx context.Context, systemPrompt, prompt string, temperature float64) (string, error) {
	if strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("empty prompt")
	}
	if s.apiKey == "" {
		return "", fmt.Errorf("missing deepseek api key")
	}

	payload := map[string]any{
		"model": "deepseek-chat",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": prompt},
		},
		"temperature": temperature,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal deepseek request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		log.Printf("[DeepSeek] request error: %v", err)
		return "", fmt.Errorf("deepseek request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Printf("[DeepSeek] read response error: %v, status=%d, partial_body=%s", readErr, resp.StatusCode, previewBody(respBytes))
		return "", fmt.Errorf("deepseek read response failed: %w", readErr)
	}

	if len(bytes.TrimSpace(respBytes)) == 0 {
		return "", fmt.Errorf("deepseek returned empty response body")
	}

	if resp.StatusCode >= 300 {
		log.Printf("[DeepSeek] non-success status=%d body=%s", resp.StatusCode, previewBody(respBytes))
		return "", fmt.Errorf("deepseek http status=%d body=%s", resp.StatusCode, previewBody(respBytes))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		log.Printf("[DeepSeek] unmarshal error: %v, body=%s", err, previewBody(respBytes))
		return "", fmt.Errorf("deepseek invalid json response: %w, body=%s", err, previewBody(respBytes))
	}
	if parsed.Error != nil {
		log.Printf("[DeepSeek] api error: message=%q type=%q code=%q", parsed.Error.Message, parsed.Error.Type, parsed.Error.Code)
		return "", fmt.Errorf("deepseek api error: %s (type=%s code=%s)", parsed.Error.Message, parsed.Error.Type, parsed.Error.Code)
	}
	if len(parsed.Choices) == 0 {
		log.Printf("[DeepSeek] empty choices body=%s", previewBody(respBytes))
		return "", fmt.Errorf("deepseek empty choices response: %s", previewBody(respBytes))
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		log.Printf("[DeepSeek] empty content body=%s", previewBody(respBytes))
		return "", fmt.Errorf("deepseek returned empty content: %s", previewBody(respBytes))
	}
	return content, nil
}

func previewBody(body []byte) string {
	const maxLen = 1000
	text := strings.TrimSpace(string(body))
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "...(truncated)"
}

func buildPrompt(req model.AIAnalyzeRequest) string {
	formatList := func(items []model.RecommendItem) string {
		if len(items) == 0 {
			return "无"
		}
		lines := make([]string, 0, len(items))
		for _, item := range items {
			line := fmt.Sprintf("- %s %s%s | 批次:%s | 选科:%s | 计划:%d | 组最低位次:%d | 概率:%.0f%% | 组内专业:%s | 推荐理由:%s",
				item.CollegeName,
				item.GroupCode,
				item.GroupName,
				item.Batch,
				item.SubjectRequirement,
				item.PlanCount,
				item.MinRank,
				item.Probability*100,
				item.Major,
				item.RecommendationReason,
			)
			lines = append(lines, line)
		}
		return strings.Join(lines, "\n")
	}

	return fmt.Sprintf(`你现在要为黑龙江考生生成 2025 年专业组口径的志愿填报建议。

学生信息：
省份：%s
分数：%d
排名：%d
科类：%s
意向专业：%s
补充偏好：%s

推荐专业组：

冲刺组：
%s

稳妥组：
%s

保底组：
%s

请输出一份黑龙江专版志愿报告，必须包含：
1. 黑龙江当前分数/位次在所选科类中的总体判断
2. 冲稳保三档专业组怎么排，为什么这样排
3. 对意向专业的匹配度分析，哪些组最贴近目标专业
4. 需要警惕的风险：位次倒挂、计划过少、是否接受调剂、组内专业跨度
5. 一个可执行的正式填报策略，直接给出志愿梯度建议`,
		req.Student.Province,
		req.Student.Score,
		req.Student.Rank,
		req.Student.Subject,
		req.Student.TargetMajor,
		req.Student.Notes,
		formatList(req.Recommend.Chong),
		formatList(req.Recommend.Wen),
		formatList(req.Recommend.Bao),
	)
}
