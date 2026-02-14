package ghcopilot

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// UICallback 定義 UI 回調介面，用於在執行過程中向用戶提供反饋
type UICallback interface {
	// OnLoopStart 當開始執行迴圈時調用
	OnLoopStart(loopNumber int, maxLoops int)
	
	// OnLoopComplete 當完成執行迴圈時調用
	OnLoopComplete(loopNumber int, result *LoopResult)
	
	// OnProgress 報告進度
	OnProgress(message string)
	
	// OnError 報告錯誤
	OnError(err error)
	
	// OnWarning 報告警告
	OnWarning(message string)
	
	// OnVerbose 報告詳細資訊（僅在 verbose 模式）
	OnVerbose(message string)
	
	// OnComplete 所有迴圈完成時調用
	OnComplete(totalLoops int, err error)
	
	// OnStreamOutput 串流輸出一行 stdout（即時顯示 Copilot 執行過程）
	OnStreamOutput(line string)
	
	// OnStreamError 串流輸出一行 stderr
	OnStreamError(line string)
}

// DefaultUICallback 預設 UI 回調實作
type DefaultUICallback struct {
	writer        io.Writer
	verbose       bool
	quiet         bool
	showSpinner   bool
	streamEnabled bool // 控制是否顯示串流輸出
	currentLoop   int
	maxLoops      int
	startTime     time.Time
}

// NewDefaultUICallback 創建預設 UI 回調
func NewDefaultUICallback(verbose, quiet bool) *DefaultUICallback {
	return &DefaultUICallback{
		writer:        os.Stdout,
		verbose:       verbose,
		quiet:         quiet,
		showSpinner:   !quiet,
		streamEnabled: !quiet, // 串流輸出在非 quiet 模式下啟用
		startTime:     time.Now(),
	}
}

// NewDefaultUICallbackWithStream 創建帶串流控制的 UI 回調
func NewDefaultUICallbackWithStream(verbose, quiet, stream bool) *DefaultUICallback {
	return &DefaultUICallback{
		writer:        os.Stdout,
		verbose:       verbose,
		quiet:         quiet,
		showSpinner:   !quiet,
		streamEnabled: stream && !quiet, // 串流需要同時滿足 stream 旗標且非 quiet
		startTime:     time.Now(),
	}
}

func (cb *DefaultUICallback) OnLoopStart(loopNumber int, maxLoops int) {
	if cb.quiet {
		return
	}
	
	cb.currentLoop = loopNumber
	cb.maxLoops = maxLoops
	
	// 計算進度百分比
	percent := float64(loopNumber-1) / float64(maxLoops) * 100
	
	// 顯示進度
	fmt.Fprintf(cb.writer, "\n%s 迴圈 %d/%d (%.0f%%) %s\n",
		colorize("▶", colorCyan),
		loopNumber,
		maxLoops,
		percent,
		colorize("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━", colorCyan),
	)
}

func (cb *DefaultUICallback) OnLoopComplete(loopNumber int, result *LoopResult) {
	if cb.quiet {
		return
	}
	
	// 顯示結果
	if result.ShouldContinue {
		fmt.Fprintf(cb.writer, "%s 迴圈 %d 完成：繼續執行\n",
			colorize("✓", colorGreen),
			loopNumber,
		)
		if cb.verbose && result.ExitReason != "" {
			fmt.Fprintf(cb.writer, "  原因: %s\n", result.ExitReason)
		}
	} else {
		fmt.Fprintf(cb.writer, "%s 迴圈 %d 完成：停止執行\n",
			colorize("⦿", colorYellow),
			loopNumber,
		)
		if result.ExitReason != "" {
			fmt.Fprintf(cb.writer, "  原因: %s\n", result.ExitReason)
		}
	}
}

func (cb *DefaultUICallback) OnProgress(message string) {
	if cb.quiet {
		return
	}
	fmt.Fprintf(cb.writer, "%s %s\n", colorize("⏳", colorYellow), message)
}

func (cb *DefaultUICallback) OnError(err error) {
	if err == nil {
		return
	}
	
	// 分析錯誤類型並提供友善訊息
	errMsg := err.Error()
	actionableMsg := makeErrorActionable(errMsg)
	
	fmt.Fprintf(cb.writer, "\n%s 錯誤: %s\n", colorize("❌", colorRed), errMsg)
	if actionableMsg != "" {
		fmt.Fprintf(cb.writer, "%s 建議: %s\n", colorize("💡", colorYellow), actionableMsg)
	}
}

func (cb *DefaultUICallback) OnWarning(message string) {
	if cb.quiet {
		return
	}
	fmt.Fprintf(cb.writer, "%s %s\n", colorize("⚠️", colorYellow), message)
}

func (cb *DefaultUICallback) OnVerbose(message string) {
	if !cb.verbose || cb.quiet {
		return
	}
	fmt.Fprintf(cb.writer, "%s %s\n", colorize("🔍", colorCyan), message)
}

func (cb *DefaultUICallback) OnComplete(totalLoops int, err error) {
	if cb.quiet {
		return
	}
	
	elapsed := time.Since(cb.startTime)
	
	fmt.Fprintf(cb.writer, "\n%s\n", strings.Repeat("━", 60))
	fmt.Fprintf(cb.writer, "%s 執行完成\n", colorize("✅", colorGreen))
	fmt.Fprintf(cb.writer, "總迴圈數: %d\n", totalLoops)
	fmt.Fprintf(cb.writer, "總耗時: %s\n", formatDuration(elapsed))
	
	if err != nil {
		fmt.Fprintf(cb.writer, "結束原因: %v\n", err)
	} else {
		fmt.Fprintf(cb.writer, "結束原因: 任務完成\n")
	}
	fmt.Fprintf(cb.writer, "%s\n", strings.Repeat("━", 60))
}

func (cb *DefaultUICallback) OnStreamOutput(line string) {
	if !cb.streamEnabled {
		return
	}
	
	// 顯示串流輸出，帶有 [copilot] 前綴
	if line != "" {
		fmt.Fprintf(cb.writer, "%s %s\n", colorize("[copilot]", colorCyan), line)
	}
}

func (cb *DefaultUICallback) OnStreamError(line string) {
	if !cb.streamEnabled {
		return
	}
	
	// 顯示串流錯誤輸出，帶有 [copilot:err] 前綴
	if line != "" {
		fmt.Fprintf(cb.writer, "%s %s\n", colorize("[copilot:err]", colorRed), line)
	}
}

// makeErrorActionable 將錯誤訊息轉換為可操作的建議
func makeErrorActionable(errMsg string) string {
	errLower := strings.ToLower(errMsg)
	
	// CLI 相關錯誤
	if strings.Contains(errLower, "executable file not found") || 
	   strings.Contains(errLower, "command not found") {
		return "請確認 GitHub Copilot CLI 已安裝：\n" +
			"  Windows: winget install GitHub.Copilot\n" +
			"  macOS/Linux: npm install -g @github/copilot\n" +
			"  驗證: copilot --version"
	}
	
	// 超時錯誤
	if strings.Contains(errLower, "timeout") || strings.Contains(errLower, "逾時") {
		return "執行超時，請嘗試：\n" +
			"  1. 增加超時設定：-cli-timeout 120s\n" +
			"  2. 簡化您的 prompt\n" +
			"  3. 檢查網路連線"
	}
	
	// API Quota 錯誤
	if strings.Contains(errLower, "quota") || strings.Contains(errLower, "402") {
		return "API quota 已用盡，請：\n" +
			"  1. 等待 quota 重置（通常每小時或每月）\n" +
			"  2. 檢查 GitHub Copilot 訂閱狀態\n" +
			"  3. 使用模擬模式測試：COPILOT_MOCK_MODE=true"
	}
	
	// 認證錯誤
	if strings.Contains(errLower, "unauthorized") || 
	   strings.Contains(errLower, "401") ||
	   strings.Contains(errLower, "authentication") {
		return "認證失敗，請執行：\n" +
			"  copilot auth\n" +
			"確保您有有效的 GitHub Copilot 訂閱"
	}
	
	// 熔斷器錯誤
	if strings.Contains(errLower, "circuit breaker") {
		return "熔斷器已觸發，請：\n" +
			"  1. 使用 'ralph-loop reset' 重置熔斷器\n" +
			"  2. 改善 prompt 明確度\n" +
			"  3. 調整閾值：-max-loops 增加迴圈數"
	}
	
	// 無進展錯誤
	if strings.Contains(errLower, "no progress") || strings.Contains(errLower, "無進展") {
		return "偵測到無進展，建議：\n" +
			"  1. 修改 prompt 使其更具體\n" +
			"  2. 分解複雜任務為多個步驟\n" +
			"  3. 檢查當前程式碼狀態"
	}
	
	// 網路錯誤
	if strings.Contains(errLower, "connection") || 
	   strings.Contains(errLower, "network") ||
	   strings.Contains(errLower, "dial") {
		return "網路連線問題，請：\n" +
			"  1. 檢查網路連線\n" +
			"  2. 檢查代理設定\n" +
			"  3. 確認防火牆未封鎖"
	}
	
	return ""
}

// 簡單的顏色支援（避免依賴外部套件）
type color string

const (
	colorReset   color = "\033[0m"
	colorRed     color = "\033[31m"
	colorGreen   color = "\033[32m"
	colorYellow  color = "\033[33m"
	colorBlue    color = "\033[34m"
	colorCyan    color = "\033[36m"
	colorBold    color = "\033[1m"
)

var colorEnabled = true

func colorize(text string, c color) string {
	if !colorEnabled {
		return text
	}
	return string(c) + text + string(colorReset)
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	
	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	} else if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// DisableColor 禁用彩色輸出
func DisableColor() {
	colorEnabled = false
}

// EnableColor 啟用彩色輸出
func EnableColor() {
	colorEnabled = true
}
