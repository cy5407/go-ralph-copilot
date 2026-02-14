package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Color 定義終端顏色代碼
type Color string

const (
	ColorReset   Color = "\033[0m"
	ColorRed     Color = "\033[31m"
	ColorGreen   Color = "\033[32m"
	ColorYellow  Color = "\033[33m"
	ColorBlue    Color = "\033[34m"
	ColorMagenta Color = "\033[35m"
	ColorCyan    Color = "\033[36m"
	ColorWhite   Color = "\033[37m"
	ColorBold    Color = "\033[1m"
)

// 全局設定
var (
	colorEnabled = true
	verboseMode  = false
	quietMode    = false
	outputFormat = "text" // text, json, table
)

// SetColorEnabled 設置是否啟用彩色輸出
func SetColorEnabled(enabled bool) {
	colorEnabled = enabled
}

// SetVerbose 設置詳細模式
func SetVerbose(verbose bool) {
	verboseMode = verbose
}

// SetQuiet 設置靜默模式
func SetQuiet(quiet bool) {
	quietMode = quiet
}

// SetOutputFormat 設置輸出格式
func SetOutputFormat(format string) {
	outputFormat = format
}

// Colorize 將文字染色
func Colorize(text string, color Color) string {
	if !colorEnabled {
		return text
	}
	return string(color) + text + string(ColorReset)
}

// PrintSuccess 打印成功訊息
func PrintSuccess(format string, args ...interface{}) {
	if quietMode {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Println(Colorize("✅ "+msg, ColorGreen))
}

// PrintError 打印錯誤訊息
func PrintError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, Colorize("❌ "+msg, ColorRed))
}

// PrintWarning 打印警告訊息
func PrintWarning(format string, args ...interface{}) {
	if quietMode {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Println(Colorize("⚠️  "+msg, ColorYellow))
}

// PrintInfo 打印資訊訊息
func PrintInfo(format string, args ...interface{}) {
	if quietMode {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Println(Colorize("ℹ️  "+msg, ColorBlue))
}

// PrintVerbose 打印詳細訊息（僅在 verbose 模式）
func PrintVerbose(format string, args ...interface{}) {
	if !verboseMode || quietMode {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Println(Colorize("🔍 "+msg, ColorCyan))
}

// PrintProgress 打印進度訊息
func PrintProgress(format string, args ...interface{}) {
	if quietMode {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Println(Colorize("⏳ "+msg, ColorYellow))
}

// ProgressBar 進度條結構
type ProgressBar struct {
	total       int
	current     int
	width       int
	description string
	startTime   time.Time
	writer      io.Writer
}

// NewProgressBar 創建新的進度條
func NewProgressBar(total int, description string) *ProgressBar {
	return &ProgressBar{
		total:       total,
		current:     0,
		width:       50,
		description: description,
		startTime:   time.Now(),
		writer:      os.Stdout,
	}
}

// Update 更新進度
func (pb *ProgressBar) Update(current int) {
	if quietMode {
		return
	}
	
	pb.current = current
	pb.Render()
}

// Increment 增加進度
func (pb *ProgressBar) Increment() {
	pb.Update(pb.current + 1)
}

// Render 渲染進度條
func (pb *ProgressBar) Render() {
	if quietMode {
		return
	}
	
	percent := float64(pb.current) / float64(pb.total) * 100
	filledWidth := int(float64(pb.width) * float64(pb.current) / float64(pb.total))
	
	// 計算預估剩餘時間
	elapsed := time.Since(pb.startTime)
	var eta string
	if pb.current > 0 {
		avgTime := elapsed / time.Duration(pb.current)
		remaining := avgTime * time.Duration(pb.total-pb.current)
		eta = fmt.Sprintf(" ETA: %s", formatDuration(remaining))
	}
	
	// 構建進度條
	bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", pb.width-filledWidth)
	
	// 打印（覆蓋當前行）
	fmt.Fprintf(pb.writer, "\r%s [%s] %d/%d (%.1f%%)%s",
		pb.description,
		Colorize(bar, ColorGreen),
		pb.current,
		pb.total,
		percent,
		eta,
	)
	
	// 完成時換行
	if pb.current >= pb.total {
		fmt.Fprintln(pb.writer)
	}
}

// Complete 完成進度條
func (pb *ProgressBar) Complete() {
	pb.Update(pb.total)
	fmt.Fprintln(pb.writer)
}

// formatDuration 格式化時間長度
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

// Spinner 旋轉指示器
type Spinner struct {
	frames      []string
	current     int
	description string
	active      bool
	stopChan    chan struct{}
	writer      io.Writer
}

// NewSpinner 創建新的旋轉指示器
func NewSpinner(description string) *Spinner {
	return &Spinner{
		frames:      []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		current:     0,
		description: description,
		active:      false,
		stopChan:    make(chan struct{}),
		writer:      os.Stdout,
	}
}

// Start 開始旋轉
func (s *Spinner) Start() {
	if quietMode || s.active {
		return
	}
	
	s.active = true
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		
		for {
			select {
			case <-s.stopChan:
				return
			case <-ticker.C:
				s.current = (s.current + 1) % len(s.frames)
				fmt.Fprintf(s.writer, "\r%s %s",
					Colorize(s.frames[s.current], ColorCyan),
					s.description,
				)
			}
		}
	}()
}

// Stop 停止旋轉
func (s *Spinner) Stop(finalMessage string) {
	if !s.active {
		return
	}
	
	s.active = false
	close(s.stopChan)
	
	// 清除當前行並打印最終訊息
	fmt.Fprintf(s.writer, "\r\033[K") // 清除當前行
	if finalMessage != "" {
		fmt.Fprintln(s.writer, finalMessage)
	}
}

// Table 表格輸出工具
type Table struct {
	headers []string
	rows    [][]string
	writer  io.Writer
}

// NewTable 創建新的表格
func NewTable(headers []string) *Table {
	return &Table{
		headers: headers,
		rows:    make([][]string, 0),
		writer:  os.Stdout,
	}
}

// AddRow 添加行
func (t *Table) AddRow(row []string) {
	t.rows = append(t.rows, row)
}

// Render 渲染表格
func (t *Table) Render() {
	if quietMode {
		return
	}
	
	// 計算列寬
	colWidths := make([]int, len(t.headers))
	for i, h := range t.headers {
		colWidths[i] = len(h)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}
	
	// 打印分隔線
	printSeparator := func() {
		for i, w := range colWidths {
			if i == 0 {
				fmt.Fprint(t.writer, "┌")
			} else {
				fmt.Fprint(t.writer, "┬")
			}
			fmt.Fprint(t.writer, strings.Repeat("─", w+2))
		}
		fmt.Fprintln(t.writer, "┐")
	}
	
	// 打印標題
	printSeparator()
	for i, h := range t.headers {
		if i == 0 {
			fmt.Fprint(t.writer, "│ ")
		} else {
			fmt.Fprint(t.writer, " │ ")
		}
		fmt.Fprint(t.writer, Colorize(h+strings.Repeat(" ", colWidths[i]-len(h)), ColorBold))
	}
	fmt.Fprintln(t.writer, " │")
	
	// 打印分隔線
	for i, w := range colWidths {
		if i == 0 {
			fmt.Fprint(t.writer, "├")
		} else {
			fmt.Fprint(t.writer, "┼")
		}
		fmt.Fprint(t.writer, strings.Repeat("─", w+2))
	}
	fmt.Fprintln(t.writer, "┤")
	
	// 打印行
	for _, row := range t.rows {
		for i, cell := range row {
			if i == 0 {
				fmt.Fprint(t.writer, "│ ")
			} else {
				fmt.Fprint(t.writer, " │ ")
			}
			fmt.Fprint(t.writer, cell+strings.Repeat(" ", colWidths[i]-len(cell)))
		}
		fmt.Fprintln(t.writer, " │")
	}
	
	// 打印底部分隔線
	for i, w := range colWidths {
		if i == 0 {
			fmt.Fprint(t.writer, "└")
		} else {
			fmt.Fprint(t.writer, "┴")
		}
		fmt.Fprint(t.writer, strings.Repeat("─", w+2))
	}
	fmt.Fprintln(t.writer, "┘")
}
