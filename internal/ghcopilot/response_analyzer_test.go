package ghcopilot

import (
	"testing"
)

// TestNewResponseAnalyzer 測試建立新的回應分析器
func TestNewResponseAnalyzer(t *testing.T) {
	ra := NewResponseAnalyzer("test response")
	if ra == nil {
		t.Error("NewResponseAnalyzer() 傳回 nil")
	}
	if ra.response != "test response" {
		t.Errorf("response 應為 'test response'，但為 '%s'", ra.response)
	}
}

// TestParseStructuredOutput 測試解析結構化輸出
func TestParseStructuredOutput(t *testing.T) {
	response := `---COPILOT_STATUS---
STATUS: CONTINUE
EXIT_SIGNAL: true
TASKS_DONE: 3/5
---END_STATUS---

其他輸出内容`

	ra := NewResponseAnalyzer(response)
	status := ra.ParseStructuredOutput()

	if status == nil {
		t.Error("ParseStructuredOutput() 應傳回非 nil")
		return
	}

	if status.Status != "CONTINUE" {
		t.Errorf("Status 應為 'CONTINUE'，但為 '%s'", status.Status)
	}

	if !status.ExitSignal {
		t.Error("ExitSignal 應為 true")
	}

	if status.TasksDone != "3/5" {
		t.Errorf("TasksDone 應為 '3/5'，但為 '%s'", status.TasksDone)
	}
}

// TestCalculateCompletionScore 測試完成分數計算
func TestCalculateCompletionScore(t *testing.T) {
	response := `---COPILOT_STATUS---
EXIT_SIGNAL: true
---END_STATUS---

所有任務已完成`

	ra := NewResponseAnalyzer(response)
	score := ra.CalculateCompletionScore()

	if score == 0 {
		t.Error("完成分數應大於 0")
	}

	if len(ra.completionIndicators) == 0 {
		t.Error("應有至少一個完成指標")
	}
}

// TestDetectTestOnlyLoop 測試偵測測試專屬迴圈
func TestDetectTestOnlyLoop(t *testing.T) {
	testResponse := "運行單元測試以驗證所有功能。執行測試。測試通過率為 100%"
	ra := NewResponseAnalyzer(testResponse)

	if !ra.DetectTestOnlyLoop() {
		t.Error("應偵測到測試專屬迴圈")
	}

	implResponse := "實作新功能。開發用戶認證模組。建立新的 API 端點。"
	ra2 := NewResponseAnalyzer(implResponse)

	if ra2.DetectTestOnlyLoop() {
		t.Error("不應偵測到測試專屬迴圈")
	}
}

// TestDetectStuckState 測試偵測卡住狀態
func TestDetectStuckState(t *testing.T) {
	ra := NewResponseAnalyzer("Error: Connection timeout")

	// 多次相同錯誤
	for i := 0; i < 6; i++ {
		isStuck, msg := ra.DetectStuckState()
		if i < 4 {
			// 前 4 次不應視為卡住
			if isStuck {
				t.Errorf("錯誤次數 %d 不應視為卡住", i+1)
			}
		} else if i == 4 {
			// 第 5 次時應偵測到卡住
			if !isStuck {
				t.Errorf("相同錯誤應在 5 次後被視為卡住。訊息: %s", msg)
			}
		}
	}
}

// TestDualConditionVerification 測試完成偵測邏輯
func TestDualConditionVerification(t *testing.T) {
	// 自然語言分數夠高（≥ 30 + 至少 2 個指標）→ 應視為完成
	// "完成" = +10（完成關鍵字）, "沒有更多工作" = +15（no_work）, 長度 < 500 = +10（短輸出）→ 35 分 + 3 指標
	response1 := "完成\n沒有更多工作"
	ra1 := NewResponseAnalyzer(response1)
	ra1.CalculateCompletionScore()

	if !ra1.IsCompleted() {
		t.Errorf("自然語言分數 %d + %d 個指標應視為完成", ra1.completionScore, len(ra1.completionIndicators))
	}

	// 分數不夠（只有一個弱指標，< 20 分）→ 不應視為完成
	response1b := "這是一段很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長很長的普通回覆，沒有任何完成或結束的跡象在裡面"
	ra1b := NewResponseAnalyzer(response1b)
	ra1b.CalculateCompletionScore()

	if ra1b.IsCompleted() {
		t.Errorf("無完成跡象不應視為完成，但分數=%d 指標=%d", ra1b.completionScore, len(ra1b.completionIndicators))
	}

	// EXIT_SIGNAL=true 單獨就夠，不需要分數
	response2 := `---COPILOT_STATUS---
EXIT_SIGNAL: true
---END_STATUS---`
	ra2 := NewResponseAnalyzer(response2)

	if !ra2.IsCompleted() {
		t.Error("有 EXIT_SIGNAL=true 應視為完成")
	}

	// 兩個條件都滿足
	response3 := `---COPILOT_STATUS---
EXIT_SIGNAL: true
---END_STATUS---

所有任務已完成，準備就緒`
	ra3 := NewResponseAnalyzer(response3)
	ra3.CalculateCompletionScore()

	if !ra3.IsCompleted() {
		t.Error("兩個條件都滿足應視為完成")
	}

	// Markdown 格式不影響偵測：**無需 push** + 完成 → score ≥ 30
	response4 := "已完成。目前分支與 origin/master **已同步**（**無需 push**）"
	ra4 := NewResponseAnalyzer(response4)
	ra4.CalculateCompletionScore()

	if !ra4.IsCompleted() {
		t.Errorf("Markdown 格式的「無需 push」應被識別，但分數=%d 指標=%v", ra4.completionScore, ra4.completionIndicators)
	}
}

// TestNormalizeError 測試錯誤正規化
func TestNormalizeError(t *testing.T) {
	ra := NewResponseAnalyzer("")

	error1 := "Error at line 42 in /path/to/file.go"
	error2 := "Error at line 100 in /path/to/file.go"

	norm1 := ra.normalizeError(error1)
	norm2 := ra.normalizeError(error2)

	// 應該正規化掉行號和路徑
	if norm1 != norm2 {
		t.Errorf("相同類型的錯誤應正規化為相同字串，但得到:\n%s\n%s", norm1, norm2)
	}
}

// TestGetAnalysisSummary 測試取得分析摘要
func TestGetAnalysisSummary(t *testing.T) {
	response := `---COPILOT_STATUS---
EXIT_SIGNAL: false
---END_STATUS---

完成了一些任務`

	ra := NewResponseAnalyzer(response)
	summary := ra.GetAnalysisSummary()

	if summary == nil {
		t.Error("GetAnalysisSummary() 應傳回非 nil")
		return
	}

	if _, ok := summary["completion_score"]; !ok {
		t.Error("摘要應包含 completion_score")
	}

	if _, ok := summary["is_test_only_loop"]; !ok {
		t.Error("摘要應包含 is_test_only_loop")
	}

	if _, ok := summary["response_length"]; !ok {
		t.Error("摘要應包含 response_length")
	}
}

// TestMultipleIndicators 測試多個完成指標
func TestMultipleIndicators(t *testing.T) {
	response := `---COPILOT_STATUS---
EXIT_SIGNAL: true
---END_STATUS---

所有任務已完成。沒有更多工作要做。系統準備就緒。`

	ra := NewResponseAnalyzer(response)
	ra.CalculateCompletionScore()

	if len(ra.completionIndicators) < 2 {
		t.Errorf("應有至少 2 個指標，但只有 %d 個", len(ra.completionIndicators))
	}
}
