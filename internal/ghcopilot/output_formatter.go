package ghcopilot

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// OutputFormat 定義輸出格式類型
type OutputFormat string

const (
	FormatText  OutputFormat = "text"
	FormatJSON  OutputFormat = "json"
	FormatTable OutputFormat = "table"
)

// OutputFormatter 格式化輸出結果
type OutputFormatter struct {
	format OutputFormat
	writer io.Writer
}

// NewOutputFormatter 創建新的輸出格式化器
func NewOutputFormatter(format OutputFormat) *OutputFormatter {
	return &OutputFormatter{
		format: format,
		writer: os.Stdout,
	}
}

// SetWriter 設置輸出目標
func (f *OutputFormatter) SetWriter(w io.Writer) {
	f.writer = w
}

// FormatResults 格式化迴圈結果列表
func (f *OutputFormatter) FormatResults(results []*LoopResult, totalTime time.Duration, err error) error {
	switch f.format {
	case FormatJSON:
		return f.formatJSON(results, totalTime, err)
	case FormatTable:
		return f.formatTable(results, totalTime, err)
	case FormatText:
		return f.formatText(results, totalTime, err)
	default:
		return fmt.Errorf("unsupported output format: %s", f.format)
	}
}

// formatJSON 格式化為 JSON
func (f *OutputFormatter) formatJSON(results []*LoopResult, totalTime time.Duration, err error) error {
	output := map[string]interface{}{
		"total_loops":   len(results),
		"total_time_ms": totalTime.Milliseconds(),
		"success":       err == nil,
		"results":       results,
	}
	
	if err != nil {
		output["error"] = err.Error()
	}
	
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// formatTable 格式化為表格
func (f *OutputFormatter) formatTable(results []*LoopResult, totalTime time.Duration, err error) error {
	// 打印表格標題
	fmt.Fprintln(f.writer, "┌─────────┬──────────────┬───────────┬────────────────────────────────┐")
	fmt.Fprintln(f.writer, "│ 迴圈    │ 完成分數     │ 繼續執行  │ 退出原因                       │")
	fmt.Fprintln(f.writer, "├─────────┼──────────────┼───────────┼────────────────────────────────┤")
	
	// 打印每個結果
	for i, result := range results {
		continueStr := "是"
		if !result.ShouldContinue {
			continueStr = "否"
		}
		
		// 截斷過長的退出原因
		exitReason := result.ExitReason
		if len(exitReason) > 30 {
			exitReason = exitReason[:27] + "..."
		}
		
		fmt.Fprintf(f.writer, "│ %-7d │ %-12d │ %-9s │ %-30s │\n",
			i+1,
			result.CompletionScore,
			continueStr,
			exitReason,
		)
	}
	
	// 打印表格底部
	fmt.Fprintln(f.writer, "└─────────┴──────────────┴───────────┴────────────────────────────────┘")
	
	// 打印摘要
	fmt.Fprintf(f.writer, "\n總迴圈數: %d\n", len(results))
	fmt.Fprintf(f.writer, "總耗時: %s\n", formatDuration(totalTime))
	
	if err != nil {
		fmt.Fprintf(f.writer, "結束原因: %v\n", err)
	} else if hasFailedResults(results) {
		fmt.Fprintln(f.writer, "結束原因: 執行失敗")
	} else {
		fmt.Fprintln(f.writer, "結束原因: 任務完成")
	}
	
	return nil
}

// formatText 格式化為純文字
func (f *OutputFormatter) formatText(results []*LoopResult, totalTime time.Duration, err error) error {
	fmt.Fprintln(f.writer, strings.Repeat("═", 60))
	fmt.Fprintln(f.writer, "  執行結果摘要")
	fmt.Fprintln(f.writer, strings.Repeat("═", 60))
	fmt.Fprintf(f.writer, "總迴圈數: %d\n", len(results))
	fmt.Fprintf(f.writer, "總耗時: %s\n", formatDuration(totalTime))
	
	if err != nil {
		fmt.Fprintf(f.writer, "結束原因: %v\n", err)
	} else if hasFailedResults(results) {
		fmt.Fprintln(f.writer, "結束原因: 執行失敗")
	} else {
		fmt.Fprintln(f.writer, "結束原因: 任務完成")
	}
	
	// 顯示每個迴圈的簡要
	if len(results) > 0 {
		fmt.Fprintln(f.writer)
		fmt.Fprintln(f.writer, "迴圈歷史:")
		for i, r := range results {
			continueStr := "否"
			if r.ShouldContinue {
				continueStr = "是"
			}
			fmt.Fprintf(f.writer, "  [%d] 繼續=%s, 分數=%d, 原因=%s\n", 
				i+1, continueStr, r.CompletionScore, r.ExitReason)
		}
	}
	
	fmt.Fprintln(f.writer, strings.Repeat("═", 60))
	return nil
}

// FormatStatus 格式化狀態資訊
func (f *OutputFormatter) FormatStatus(status *ClientStatus) error {
	switch f.format {
	case FormatJSON:
		return f.formatStatusJSON(status)
	case FormatTable:
		return f.formatStatusTable(status)
	case FormatText:
		return f.formatStatusText(status)
	default:
		return fmt.Errorf("unsupported output format: %s", f.format)
	}
}

func (f *OutputFormatter) formatStatusJSON(status *ClientStatus) error {
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(status)
}

func (f *OutputFormatter) formatStatusTable(status *ClientStatus) error {
	fmt.Fprintln(f.writer, "┌─────────────────────────┬────────────────────────┐")
	fmt.Fprintln(f.writer, "│ 屬性                    │ 值                     │")
	fmt.Fprintln(f.writer, "├─────────────────────────┼────────────────────────┤")
	fmt.Fprintf(f.writer, "│ 初始化                  │ %-22v │\n", status.Initialized)
	fmt.Fprintf(f.writer, "│ 已關閉                  │ %-22v │\n", status.Closed)
	fmt.Fprintf(f.writer, "│ 熔斷器狀態              │ %-22s │\n", status.CircuitBreakerState)
	fmt.Fprintf(f.writer, "│ 熔斷器打開              │ %-22v │\n", status.CircuitBreakerOpen)
	fmt.Fprintf(f.writer, "│ 已執行迴圈數            │ %-22d │\n", status.LoopsExecuted)
	fmt.Fprintln(f.writer, "└─────────────────────────┴────────────────────────┘")
	
	if status.Summary != nil && len(status.Summary) > 0 {
		fmt.Fprintln(f.writer)
		fmt.Fprintln(f.writer, "摘要:")
		for k, v := range status.Summary {
			fmt.Fprintf(f.writer, "  %s: %v\n", k, v)
		}
	}
	
	return nil
}

func (f *OutputFormatter) formatStatusText(status *ClientStatus) error {
	fmt.Fprintln(f.writer, strings.Repeat("═", 40))
	fmt.Fprintln(f.writer, "  Ralph Loop 狀態")
	fmt.Fprintln(f.writer, strings.Repeat("═", 40))
	fmt.Fprintf(f.writer, "初始化: %v\n", status.Initialized)
	fmt.Fprintf(f.writer, "已關閉: %v\n", status.Closed)
	fmt.Fprintf(f.writer, "熔斷器狀態: %s\n", status.CircuitBreakerState)
	fmt.Fprintf(f.writer, "熔斷器打開: %v\n", status.CircuitBreakerOpen)
	fmt.Fprintf(f.writer, "已執行迴圈數: %d\n", status.LoopsExecuted)
	
	if status.Summary != nil && len(status.Summary) > 0 {
		fmt.Fprintln(f.writer)
		fmt.Fprintln(f.writer, "摘要:")
		for k, v := range status.Summary {
			fmt.Fprintf(f.writer, "  %s: %v\n", k, v)
		}
	}
	
	fmt.Fprintln(f.writer, strings.Repeat("═", 40))
	return nil
}

// hasFailedResults 檢查是否有失敗的迴圈結果
func hasFailedResults(results []*LoopResult) bool {
	for _, result := range results {
		if result.IsFailed() {
			return true
		}
	}
	return false
}
