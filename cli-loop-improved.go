package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Ralph Loop 啟發: 基於程序結束 + 結構化輸出解析

type LoopController struct {
	cliPath  string
	timeout  time.Duration
	maxLoops int

	// 退出訊號追蹤
	completionIndicators []int // Loop 編號
	exitSignals          []int
}

type CLIResponse struct {
	Output      string
	Duration    time.Duration
	Error       error
	ExitSignal  bool // AI 明確表示完成
	HasProgress bool // 有檔案變更
	Confidence  int  // 信心分數 0-100
}

func NewLoopController(cliPath string) *LoopController {
	return &LoopController{
		cliPath:              cliPath,
		timeout:              30 * time.Second,
		maxLoops:             100,
		completionIndicators: []int{},
		exitSignals:          []int{},
	}
}

// 核心方法 1: 執行 CLI 直到程序結束 (不用空閒偵測)
func (lc *LoopController) ExecuteCLI(ctx context.Context, args ...string) CLIResponse {
	startTime := time.Now()

	cmdCtx, cancel := context.WithTimeout(ctx, lc.timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, lc.cliPath, args...)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return CLIResponse{Error: err}
	}

	var output bytes.Buffer
	combined := io.MultiReader(stdout, stderr)

	// 🔑 即時讀取但不用空閒偵測來判斷結束
	go func() {
		scanner := bufio.NewScanner(combined)
		for scanner.Scan() {
			line := scanner.Text()
			output.WriteString(line + "\n")
			fmt.Printf("  %s\n", line)
		}
	}()

	// ✅ 等待程序自然結束
	err := cmd.Wait()
	duration := time.Since(startTime)

	response := CLIResponse{
		Output:   output.String(),
		Duration: duration,
		Error:    err,
	}

	return response
}

// 核心方法 2: 解析 AI 輸出尋找退出訊號
func (lc *LoopController) AnalyzeResponse(output string, loopNumber int) CLIResponse {
	response := CLIResponse{
		Output:     output,
		ExitSignal: false,
		Confidence: 0,
	}

	// 1. 檢查結構化輸出 (類似 RALPH_STATUS)
	if strings.Contains(output, "---COPILOT_STATUS---") {
		// 解析結構化區塊
		re := regexp.MustCompile(`EXIT_SIGNAL:\s*(true|false)`)
		if matches := re.FindStringSubmatch(output); len(matches) > 1 {
			response.ExitSignal = matches[1] == "true"
			response.Confidence += 100 // 明確訊號 = 最高信心
		}
	}

	// 2. 偵測完成關鍵字
	completionKeywords := []string{
		"done", "complete", "finished",
		"all tasks complete", "ready for review",
	}

	for _, keyword := range completionKeywords {
		if strings.Contains(strings.ToLower(output), keyword) {
			response.Confidence += 10
			break
		}
	}

	// 3. 檢測 "nothing to do" 模式
	noWorkPatterns := []string{
		"nothing to do", "no changes",
		"already implemented", "up to date",
	}

	for _, pattern := range noWorkPatterns {
		if strings.Contains(strings.ToLower(output), pattern) {
			response.Confidence += 15
		}
	}

	// 4. 更新完成指標
	if response.Confidence >= 60 {
		lc.completionIndicators = append(lc.completionIndicators, loopNumber)
		// 只保留最近 5 個
		if len(lc.completionIndicators) > 5 {
			lc.completionIndicators = lc.completionIndicators[1:]
		}
	}

	if response.ExitSignal {
		lc.exitSignals = append(lc.exitSignals, loopNumber)
	}

	return response
}

// 核心方法 3: 決定是否應該退出循環 (雙重驗證)
func (lc *LoopController) ShouldExit() (bool, string) {
	// 條件 1: 太多連續完成指標
	if len(lc.completionIndicators) >= 2 {
		// 條件 2: AI 明確表示完成
		if len(lc.exitSignals) > 0 {
			return true, "project_complete (verified by AI)"
		}

		// 只有啟發式偵測，但 AI 沒確認 → 繼續
		fmt.Println("[INFO] Completion patterns detected but AI has not confirmed, continuing...")
	}

	return false, ""
}

// 核心方法 4: 主循環邏輯
func (lc *LoopController) Run() {
	ctx := context.Background()

	for loopCount := 1; loopCount <= lc.maxLoops; loopCount++ {
		fmt.Printf("\n=== Loop #%d ===\n", loopCount)

		// 1. 檢查退出條件
		if shouldExit, reason := lc.ShouldExit(); shouldExit {
			fmt.Printf("\n✅ Graceful exit: %s\n", reason)
			fmt.Printf("Total loops: %d\n", loopCount-1)
			break
		}

		// 2. 執行 CLI
		response := lc.ExecuteCLI(ctx,
			"-ExecutionPolicy", "Bypass",
			"-File", "mock-copilot-cli.ps1",
			"-Command", "what-the-shell",
			"-Prompt", "list go files",
		)

		if response.Error != nil {
			fmt.Printf("Error: %v\n", response.Error)
			continue
		}

		fmt.Printf("Execution time: %v\n", response.Duration)

		// 3. 分析回應
		analysis := lc.AnalyzeResponse(response.Output, loopCount)
		fmt.Printf("Exit Signal: %v, Confidence: %d\n",
			analysis.ExitSignal, analysis.Confidence)

		// 4. 模擬任務處理延遲
		time.Sleep(2 * time.Second)
	}
}

// 進階: 支援結構化 JSON 輸出的版本
type CopilotStatus struct {
	Status       string `json:"status"` // "IN_PROGRESS" | "COMPLETE"
	ExitSignal   bool   `json:"exit_signal"`
	TasksDone    int    `json:"tasks_done"`
	FilesChanged int    `json:"files_changed"`
	Summary      string `json:"summary"`
}

func ParseStructuredOutput(output string) (*CopilotStatus, error) {
	// 尋找 JSON 區塊
	re := regexp.MustCompile(`(?s)---COPILOT_STATUS---\s*\{.*?\}\s*---END_STATUS---`)
	match := re.FindString(output)

	if match == "" {
		return nil, fmt.Errorf("no structured output found")
	}

	// 提取 JSON
	jsonStart := strings.Index(match, "{")
	jsonEnd := strings.LastIndex(match, "}")
	jsonStr := match[jsonStart : jsonEnd+1]

	var status CopilotStatus
	if err := json.Unmarshal([]byte(jsonStr), &status); err != nil {
		return nil, err
	}

	return &status, nil
}

func main() {
	fmt.Println("=== Ralph Loop 啟發式 CLI 循環控制器 ===\n")

	controller := NewLoopController("powershell.exe")
	controller.Run()

	fmt.Println("\n=== 關鍵發現 ===")
	fmt.Println("1. ✅ 等待程序結束 (cmd.Wait) 而非空閒偵測")
	fmt.Println("2. ✅ 解析結構化輸出尋找 EXIT_SIGNAL")
	fmt.Println("3. ✅ 雙重驗證: 啟發式 + AI 明確訊號")
	fmt.Println("4. ✅ 信心分數系統避免過早退出")
}
