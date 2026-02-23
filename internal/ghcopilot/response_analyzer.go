package ghcopilot

import (
	"regexp"
	"strings"
)

// CopilotStatus 代表 Copilot 的狀態輸出
type CopilotStatus struct {
	Status     string
	ExitSignal bool
	TasksDone  string
	RawBlock   string
}

// ResponseAnalyzer 用於分析 Copilot 回應
type ResponseAnalyzer struct {
	response             string
	completionScore      int
	isTestOnlyLoop       bool
	completionIndicators []string
	previousErrors       []string
	consecutiveErrors    int
}

// NewResponseAnalyzer 建立新的回應分析器
func NewResponseAnalyzer(response string) *ResponseAnalyzer {
	return &ResponseAnalyzer{
		response:             response,
		completionScore:      0,
		isTestOnlyLoop:       false,
		completionIndicators: []string{},
		previousErrors:       []string{},
		consecutiveErrors:    0,
	}
}

// ParseStructuredOutput 解析結構化輸出區塊
func (ra *ResponseAnalyzer) ParseStructuredOutput() *CopilotStatus {
	// 查找 ---COPILOT_STATUS--- 或 ---RALPH_STATUS--- 區塊，支援 CRLF
	pattern := `(?s)---(?:COPILOT_STATUS|RALPH_STATUS)---\r?\n(.*?)\r?\n---END(?:_STATUS|_RALPH_STATUS)---`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(ra.response)

	if len(matches) < 2 {
		return nil
	}

	// 正規化 CRLF
	block := strings.ReplaceAll(matches[1], "\r\n", "\n")
	status := &CopilotStatus{
		RawBlock: block,
	}

	// 提取各個欄位
	lines := strings.Split(block, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "STATUS:") {
			status.Status = strings.TrimSpace(strings.TrimPrefix(line, "STATUS:"))
		} else if strings.HasPrefix(line, "EXIT_SIGNAL:") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "EXIT_SIGNAL:"))
			status.ExitSignal = strings.ToLower(value) == "true"
		} else if strings.HasPrefix(line, "TASKS_DONE:") {
			status.TasksDone = strings.TrimSpace(strings.TrimPrefix(line, "TASKS_DONE:"))
		}
	}

	return status
}

// CalculateCompletionScore 計算完成分數
func (ra *ResponseAnalyzer) CalculateCompletionScore() int {
	score := 0

	// 檢查結構化輸出
	status := ra.ParseStructuredOutput()
	if status != nil && status.ExitSignal {
		score += 100
		ra.completionIndicators = append(ra.completionIndicators, "explicit_exit_signal")
	}

	// 檢查完成關鍵字
	completionKeywords := []string{
		"完成", "完全完成", "全部完成", "done", "finished", "completed",
		"已全部完成", "所有任務已完成", "準備就緒",
	}

	for _, keyword := range completionKeywords {
		if strings.Contains(strings.ToLower(ra.response), strings.ToLower(keyword)) {
			score += 10
			ra.completionIndicators = append(ra.completionIndicators, keyword)
			break
		}
	}

	// 檢查無工作模式
	noWorkPatterns := []string{
		"沒有更多工作", "no more work", "沒有其他",
		"no further changes", "沒有待辦",
		// Layer 2：補充「任務不需要做」的自然語言備用偵測
		"不需要更新", "無需更新", "無需修改", "不需要push", "無需push",
		"不需要 push", "無需 push", "不需要git push", "無需git push",
		"已是最新", "已完整", "狀態良好", "無需任何",
		"no update needed", "no changes needed", "up to date",
		"nothing to push", "no action required", "already up to date",
	}

	for _, pattern := range noWorkPatterns {
		if strings.Contains(strings.ToLower(ra.response), strings.ToLower(pattern)) {
			score += 15
			ra.completionIndicators = append(ra.completionIndicators, "no_work_mode")
			break
		}
	}

	// 檢查輸出長度下降（表示逐漸接近完成）
	if len(ra.response) < 500 {
		score += 10
		ra.completionIndicators = append(ra.completionIndicators, "short_output")
	}

	ra.completionScore = score
	return score
}

// DetectTestOnlyLoop 偵測是否為測試專屬迴圈
func (ra *ResponseAnalyzer) DetectTestOnlyLoop() bool {
	testPatterns := []string{
		"test", "testing", "單元測試", "集成測試",
		"run tests", "執行測試", "pytest", "unittest",
	}

	implementPatterns := []string{
		"implement", "feature", "功能", "實作", "開發", "添加",
		"modify", "fix", "修改", "解決", "建立",
	}

	// 計算測試相關的詞彙
	testCount := 0
	for _, pattern := range testPatterns {
		if count := strings.Count(strings.ToLower(ra.response), strings.ToLower(pattern)); count > 0 {
			testCount += count
		}
	}

	// 計算實作相關的詞彙
	implCount := 0
	for _, pattern := range implementPatterns {
		if count := strings.Count(strings.ToLower(ra.response), strings.ToLower(pattern)); count > 0 {
			implCount += count
		}
	}

	// 如果測試相關詞彙 > 實作相關詞彙，視為測試專屬
	ra.isTestOnlyLoop = testCount > implCount

	return ra.isTestOnlyLoop
}

// DetectStuckState 偵測卡住狀態
func (ra *ResponseAnalyzer) DetectStuckState() (bool, string) {
	// 正規化錯誤訊息（用於比較）
	currentError := ra.normalizeError(ra.response)

	if currentError == "" {
		ra.consecutiveErrors = 0
		return false, ""
	}

	// 檢查是否與最後的錯誤相同
	if len(ra.previousErrors) > 0 && ra.previousErrors[len(ra.previousErrors)-1] == currentError {
		ra.consecutiveErrors++

		// 如果連續 5 次相同錯誤，視為卡住
		if ra.consecutiveErrors >= 5 {
			return true, "相同錯誤已出現 5 次"
		}
	} else {
		ra.consecutiveErrors = 1
	}

	// 保留最後 3 個錯誤以供比較
	ra.previousErrors = append(ra.previousErrors, currentError)
	if len(ra.previousErrors) > 3 {
		ra.previousErrors = ra.previousErrors[1:]
	}

	return false, ""
}

// normalizeError 正規化錯誤訊息便於比較
func (ra *ResponseAnalyzer) normalizeError(text string) string {
	// 移除行號
	normalized := regexp.MustCompile(`line\s+\d+`).ReplaceAllString(text, "line")
	normalized = regexp.MustCompile(`:\d+:`).ReplaceAllString(normalized, "::")

	// 移除完整路徑，只保留檔名
	normalized = regexp.MustCompile(`/[^/]*?\.go`).ReplaceAllString(normalized, "FILE.go")
	normalized = regexp.MustCompile(`\\[^\\]*?\.\w+`).ReplaceAllString(normalized, "FILE")

	// 轉換為小寫並移除多餘空白
	normalized = strings.ToLower(normalized)
	normalized = strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " "))

	// 只取前 200 字符用於比較
	if len(normalized) > 200 {
		normalized = normalized[:200]
	}

	return normalized
}

// GetAnalysisSummary 取得分析摘要
func (ra *ResponseAnalyzer) GetAnalysisSummary() map[string]interface{} {
	ra.CalculateCompletionScore()

	return map[string]interface{}{
		"completion_score":      ra.completionScore,
		"completion_indicators": ra.completionIndicators,
		"is_test_only_loop":     ra.DetectTestOnlyLoop(),
		"response_length":       len(ra.response),
		"structured_output":     ra.ParseStructuredOutput(),
	}
}

// IsCompleted 確定是否應該完成
// 三層防禦策略
func (ra *ResponseAnalyzer) IsCompleted() bool {
	status := ra.ParseStructuredOutput()

	// Layer 3：EXIT_SIGNAL=true 單獨就夠，不需要第二條件
	if status != nil && status.ExitSignal {
		return true
	}

	// 備用：無結構化輸出，但自然語言分數夠高（≥ 30）且有 2 個指標
	if ra.completionScore >= 30 && len(ra.completionIndicators) >= 2 {
		return true
	}

	return false
}
