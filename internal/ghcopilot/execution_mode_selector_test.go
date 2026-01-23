package ghcopilot

import (
	"context"
	"errors"
	"testing"
	"time"
)

// ========================
// ExecutionMode 測試
// ========================

func TestExecutionModeString(t *testing.T) {
	tests := []struct {
		mode     ExecutionMode
		expected string
	}{
		{ModeCLI, "cli"},
		{ModeSDK, "sdk"},
		{ModeAuto, "auto"},
		{ModeHybrid, "hybrid"},
		{ExecutionMode(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.expected {
			t.Errorf("ExecutionMode(%d).String() = %q, expected %q", tt.mode, got, tt.expected)
		}
	}
}

// ========================
// TaskComplexity 測試
// ========================

func TestTaskComplexityString(t *testing.T) {
	tests := []struct {
		complexity TaskComplexity
		expected   string
	}{
		{ComplexitySimple, "simple"},
		{ComplexityMedium, "medium"},
		{ComplexityComplex, "complex"},
		{TaskComplexity(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.complexity.String(); got != tt.expected {
			t.Errorf("TaskComplexity(%d).String() = %q, expected %q", tt.complexity, got, tt.expected)
		}
	}
}

// ========================
// Task 測試
// ========================

func TestNewTask(t *testing.T) {
	task := NewTask("task-1", "test prompt")

	if task.ID != "task-1" {
		t.Errorf("expected ID 'task-1', got %q", task.ID)
	}
	if task.Prompt != "test prompt" {
		t.Errorf("expected Prompt 'test prompt', got %q", task.Prompt)
	}
	if task.Complexity != ComplexitySimple {
		t.Errorf("expected ComplexitySimple, got %v", task.Complexity)
	}
	if task.PreferredMode != ModeAuto {
		t.Errorf("expected ModeAuto, got %v", task.PreferredMode)
	}
}

func TestTask_WithComplexity(t *testing.T) {
	task := NewTask("1", "prompt").WithComplexity(ComplexityComplex)

	if task.Complexity != ComplexityComplex {
		t.Errorf("expected ComplexityComplex, got %v", task.Complexity)
	}
}

func TestTask_WithTimeout(t *testing.T) {
	task := NewTask("1", "prompt").WithTimeout(5 * time.Minute)

	if task.Timeout != 5*time.Minute {
		t.Errorf("expected 5m timeout, got %v", task.Timeout)
	}
}

func TestTask_WithPreferredMode(t *testing.T) {
	task := NewTask("1", "prompt").WithPreferredMode(ModeSDK)

	if task.PreferredMode != ModeSDK {
		t.Errorf("expected ModeSDK, got %v", task.PreferredMode)
	}
}

func TestTask_WithPriority(t *testing.T) {
	task := NewTask("1", "prompt").WithPriority(10)

	if task.Priority != 10 {
		t.Errorf("expected priority 10, got %d", task.Priority)
	}
}

func TestTask_WithTags(t *testing.T) {
	task := NewTask("1", "prompt").WithTags("tag1", "tag2")

	if len(task.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(task.Tags))
	}
	if task.Tags[0] != "tag1" || task.Tags[1] != "tag2" {
		t.Errorf("tags mismatch: %v", task.Tags)
	}
}

func TestTask_SetRequiresSDK(t *testing.T) {
	task := NewTask("1", "prompt").SetRequiresSDK(true)

	if !task.RequiresSDK {
		t.Error("expected RequiresSDK to be true")
	}
}

func TestTask_Chaining(t *testing.T) {
	task := NewTask("1", "prompt").
		WithComplexity(ComplexityMedium).
		WithTimeout(1 * time.Minute).
		WithPreferredMode(ModeHybrid).
		WithPriority(3).
		WithTags("test").
		SetRequiresSDK(true)

	if task.Complexity != ComplexityMedium {
		t.Error("complexity not set correctly")
	}
	if task.Timeout != 1*time.Minute {
		t.Error("timeout not set correctly")
	}
	if task.PreferredMode != ModeHybrid {
		t.Error("preferred mode not set correctly")
	}
	if task.Priority != 3 {
		t.Error("priority not set correctly")
	}
	if len(task.Tags) != 1 {
		t.Error("tags not set correctly")
	}
	if !task.RequiresSDK {
		t.Error("requires SDK not set correctly")
	}
}

// ========================
// ExecutionModeSelector 測試
// ========================

func TestNewExecutionModeSelector(t *testing.T) {
	selector := NewExecutionModeSelector()

	if selector == nil {
		t.Fatal("selector should not be nil")
	}
	if selector.GetDefaultMode() != ModeAuto {
		t.Errorf("expected default mode ModeAuto, got %v", selector.GetDefaultMode())
	}
	if !selector.IsFallbackEnabled() {
		t.Error("expected fallback enabled by default")
	}
	if !selector.IsSDKAvailable() {
		t.Error("expected SDK available by default")
	}
	if !selector.IsCLIAvailable() {
		t.Error("expected CLI available by default")
	}
}

func TestExecutionModeSelector_SetDefaultMode(t *testing.T) {
	selector := NewExecutionModeSelector()
	selector.SetDefaultMode(ModeSDK)

	if selector.GetDefaultMode() != ModeSDK {
		t.Errorf("expected ModeSDK, got %v", selector.GetDefaultMode())
	}
}

func TestExecutionModeSelector_SetFallbackEnabled(t *testing.T) {
	selector := NewExecutionModeSelector()
	selector.SetFallbackEnabled(false)

	if selector.IsFallbackEnabled() {
		t.Error("expected fallback disabled")
	}
}

func TestExecutionModeSelector_SetSDKAvailable(t *testing.T) {
	selector := NewExecutionModeSelector()
	selector.SetSDKAvailable(false)

	if selector.IsSDKAvailable() {
		t.Error("expected SDK unavailable")
	}
}

func TestExecutionModeSelector_SetCLIAvailable(t *testing.T) {
	selector := NewExecutionModeSelector()
	selector.SetCLIAvailable(false)

	if selector.IsCLIAvailable() {
		t.Error("expected CLI unavailable")
	}
}

func TestExecutionModeSelector_AddRule(t *testing.T) {
	selector := NewExecutionModeSelector()

	rule := SelectionRule{
		Name:      "test-rule",
		Priority:  1,
		Condition: func(task *Task) bool { return true },
		Mode:      ModeSDK,
	}
	selector.AddRule(rule)

	if selector.GetRuleCount() != 1 {
		t.Errorf("expected 1 rule, got %d", selector.GetRuleCount())
	}
}

func TestExecutionModeSelector_ClearRules(t *testing.T) {
	selector := NewExecutionModeSelector()
	selector.AddRule(SelectionRule{Name: "rule1", Priority: 1})
	selector.AddRule(SelectionRule{Name: "rule2", Priority: 2})

	selector.ClearRules()

	if selector.GetRuleCount() != 0 {
		t.Errorf("expected 0 rules after clear, got %d", selector.GetRuleCount())
	}
}

func TestExecutionModeSelector_Choose_PreferredMode(t *testing.T) {
	selector := NewExecutionModeSelector()

	task := NewTask("1", "prompt").WithPreferredMode(ModeSDK)
	mode := selector.Choose(task)

	if mode != ModeSDK {
		t.Errorf("expected ModeSDK, got %v", mode)
	}
}

func TestExecutionModeSelector_Choose_RequiresSDK(t *testing.T) {
	selector := NewExecutionModeSelector()

	task := NewTask("1", "prompt").SetRequiresSDK(true)
	mode := selector.Choose(task)

	if mode != ModeSDK {
		t.Errorf("expected ModeSDK when requires SDK, got %v", mode)
	}
}

func TestExecutionModeSelector_Choose_RequiresSDK_Fallback(t *testing.T) {
	selector := NewExecutionModeSelector()
	selector.SetSDKAvailable(false)

	task := NewTask("1", "prompt").SetRequiresSDK(true)
	mode := selector.Choose(task)

	// SDK 不可用時應該回退到 CLI
	if mode != ModeCLI {
		t.Errorf("expected ModeCLI fallback when SDK unavailable, got %v", mode)
	}
}

func TestExecutionModeSelector_Choose_SimpleTask(t *testing.T) {
	selector := NewExecutionModeSelector()

	task := NewTask("1", "prompt").WithComplexity(ComplexitySimple)
	mode := selector.Choose(task)

	if mode != ModeCLI {
		t.Errorf("expected ModeCLI for simple task, got %v", mode)
	}
}

func TestExecutionModeSelector_Choose_ComplexTask(t *testing.T) {
	selector := NewExecutionModeSelector()

	task := NewTask("1", "prompt").WithComplexity(ComplexityComplex)
	mode := selector.Choose(task)

	if mode != ModeSDK {
		t.Errorf("expected ModeSDK for complex task, got %v", mode)
	}
}

func TestExecutionModeSelector_Choose_WithRule(t *testing.T) {
	selector := NewExecutionModeSelector()

	rule := SelectionRule{
		Name:     "force-sdk-for-long-prompts",
		Priority: 1,
		Condition: func(task *Task) bool {
			return task != nil && len(task.Prompt) > 10
		},
		Mode: ModeSDK,
	}
	selector.AddRule(rule)

	task := NewTask("1", "this is a very long prompt that should trigger the rule")
	mode := selector.Choose(task)

	if mode != ModeSDK {
		t.Errorf("expected ModeSDK due to rule, got %v", mode)
	}
}

func TestExecutionModeSelector_Choose_AutoMode(t *testing.T) {
	selector := NewExecutionModeSelector()

	task := NewTask("1", "prompt") // 預設是 ModeAuto
	mode := selector.Choose(task)

	// 自動模式應該選擇 CLI（簡單任務）
	if mode != ModeCLI {
		t.Errorf("expected ModeCLI for auto mode with simple task, got %v", mode)
	}
}

func TestExecutionModeSelector_Choose_NilTask(t *testing.T) {
	selector := NewExecutionModeSelector()

	mode := selector.Choose(nil)

	// nil 任務應該使用預設模式（自動 -> CLI）
	if mode != ModeCLI {
		t.Errorf("expected ModeCLI for nil task, got %v", mode)
	}
}

func TestExecutionModeSelector_Choose_FallbackToSDK(t *testing.T) {
	selector := NewExecutionModeSelector()
	selector.SetCLIAvailable(false)

	task := NewTask("1", "prompt").WithPreferredMode(ModeCLI)
	mode := selector.Choose(task)

	// CLI 不可用時應該回退到 SDK
	if mode != ModeSDK {
		t.Errorf("expected ModeSDK fallback when CLI unavailable, got %v", mode)
	}
}

func TestExecutionModeSelector_Choose_HybridMode(t *testing.T) {
	selector := NewExecutionModeSelector()

	task := NewTask("1", "prompt").WithPreferredMode(ModeHybrid)
	mode := selector.Choose(task)

	if mode != ModeHybrid {
		t.Errorf("expected ModeHybrid, got %v", mode)
	}
}

func TestExecutionModeSelector_GetMetrics(t *testing.T) {
	selector := NewExecutionModeSelector()

	task1 := NewTask("1", "prompt").WithComplexity(ComplexitySimple)
	task2 := NewTask("2", "prompt").WithComplexity(ComplexityComplex)

	selector.Choose(task1) // CLI
	selector.Choose(task2) // SDK

	metrics := selector.GetMetrics()
	if metrics.TotalSelections != 2 {
		t.Errorf("expected 2 total selections, got %d", metrics.TotalSelections)
	}
	if metrics.CLISelections != 1 {
		t.Errorf("expected 1 CLI selection, got %d", metrics.CLISelections)
	}
	if metrics.SDKSelections != 1 {
		t.Errorf("expected 1 SDK selection, got %d", metrics.SDKSelections)
	}
}

func TestExecutionModeSelector_ResetMetrics(t *testing.T) {
	selector := NewExecutionModeSelector()

	selector.Choose(NewTask("1", "prompt"))
	selector.ResetMetrics()

	metrics := selector.GetMetrics()
	if metrics.TotalSelections != 0 {
		t.Errorf("expected 0 selections after reset, got %d", metrics.TotalSelections)
	}
}

// ========================
// PerformanceMonitor 測試
// ========================

func TestNewPerformanceMonitor(t *testing.T) {
	monitor := NewPerformanceMonitor()

	if monitor == nil {
		t.Fatal("monitor should not be nil")
	}
}

func TestPerformanceMonitor_RecordExecution_CLI(t *testing.T) {
	monitor := NewPerformanceMonitor()

	monitor.RecordExecution(ModeCLI, 100*time.Millisecond, nil)

	execs, avgTime, errorRate := monitor.GetCLIMetrics()
	if execs != 1 {
		t.Errorf("expected 1 execution, got %d", execs)
	}
	if avgTime != 100*time.Millisecond {
		t.Errorf("expected 100ms avg time, got %v", avgTime)
	}
	if errorRate != 0 {
		t.Errorf("expected 0 error rate, got %f", errorRate)
	}
}

func TestPerformanceMonitor_RecordExecution_SDK(t *testing.T) {
	monitor := NewPerformanceMonitor()

	monitor.RecordExecution(ModeSDK, 200*time.Millisecond, nil)

	execs, avgTime, _ := monitor.GetSDKMetrics()
	if execs != 1 {
		t.Errorf("expected 1 execution, got %d", execs)
	}
	if avgTime != 200*time.Millisecond {
		t.Errorf("expected 200ms avg time, got %v", avgTime)
	}
}

func TestPerformanceMonitor_RecordExecution_WithError(t *testing.T) {
	monitor := NewPerformanceMonitor()

	monitor.RecordExecution(ModeCLI, 100*time.Millisecond, errors.New("error"))
	monitor.RecordExecution(ModeCLI, 100*time.Millisecond, nil)

	_, _, errorRate := monitor.GetCLIMetrics()
	if errorRate != 0.5 {
		t.Errorf("expected 50%% error rate, got %f", errorRate)
	}
}

func TestPerformanceMonitor_GetPerformanceMetrics(t *testing.T) {
	monitor := NewPerformanceMonitor()

	monitor.RecordExecution(ModeCLI, 100*time.Millisecond, nil)
	monitor.RecordExecution(ModeSDK, 200*time.Millisecond, nil)

	metrics := monitor.GetPerformanceMetrics()
	if metrics.CLITime != 100*time.Millisecond {
		t.Errorf("expected CLI time 100ms, got %v", metrics.CLITime)
	}
	if metrics.SDKTime != 200*time.Millisecond {
		t.Errorf("expected SDK time 200ms, got %v", metrics.SDKTime)
	}
}

func TestPerformanceMonitor_Reset(t *testing.T) {
	monitor := NewPerformanceMonitor()

	monitor.RecordExecution(ModeCLI, 100*time.Millisecond, nil)
	monitor.Reset()

	execs, _, _ := monitor.GetCLIMetrics()
	if execs != 0 {
		t.Errorf("expected 0 executions after reset, got %d", execs)
	}
}

// ========================
// HybridExecutor 測試
// ========================

func TestNewHybridExecutor(t *testing.T) {
	executor := NewHybridExecutor(nil)

	if executor == nil {
		t.Fatal("executor should not be nil")
	}
	if executor.GetSelector() == nil {
		t.Error("expected selector to be initialized")
	}
	if executor.GetPerformanceMonitor() == nil {
		t.Error("expected monitor to be initialized")
	}
}

func TestHybridExecutor_SetCLIExecutor(t *testing.T) {
	executor := NewHybridExecutor(nil)

	called := false
	executor.SetCLIExecutor(func(ctx context.Context, prompt string) (string, error) {
		called = true
		return "cli result", nil
	})

	task := NewTask("1", "prompt").WithPreferredMode(ModeCLI)
	result, err := executor.Execute(context.Background(), task)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("CLI executor should have been called")
	}
	if result != "cli result" {
		t.Errorf("expected 'cli result', got %q", result)
	}
}

func TestHybridExecutor_SetSDKExecutor(t *testing.T) {
	executor := NewHybridExecutor(nil)

	called := false
	executor.SetSDKExecutor(func(ctx context.Context, prompt string) (string, error) {
		called = true
		return "sdk result", nil
	})

	task := NewTask("1", "prompt").WithPreferredMode(ModeSDK)
	result, err := executor.Execute(context.Background(), task)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("SDK executor should have been called")
	}
	if result != "sdk result" {
		t.Errorf("expected 'sdk result', got %q", result)
	}
}

func TestHybridExecutor_Execute_HybridMode_SDKSuccess(t *testing.T) {
	executor := NewHybridExecutor(nil)

	cliCalled := false
	sdkCalled := false

	executor.SetCLIExecutor(func(ctx context.Context, prompt string) (string, error) {
		cliCalled = true
		return "cli result", nil
	})
	executor.SetSDKExecutor(func(ctx context.Context, prompt string) (string, error) {
		sdkCalled = true
		return "sdk result", nil
	})

	task := NewTask("1", "prompt").WithPreferredMode(ModeHybrid)
	result, err := executor.Execute(context.Background(), task)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cliCalled {
		t.Error("CLI should not be called when SDK succeeds in hybrid mode")
	}
	if !sdkCalled {
		t.Error("SDK should be called first in hybrid mode")
	}
	if result != "sdk result" {
		t.Errorf("expected 'sdk result', got %q", result)
	}
}

func TestHybridExecutor_Execute_HybridMode_SDKFailure_Fallback(t *testing.T) {
	selector := NewExecutionModeSelector()
	selector.SetFallbackEnabled(true)
	executor := NewHybridExecutor(selector)

	cliCalled := false

	executor.SetCLIExecutor(func(ctx context.Context, prompt string) (string, error) {
		cliCalled = true
		return "cli result", nil
	})
	executor.SetSDKExecutor(func(ctx context.Context, prompt string) (string, error) {
		return "", errors.New("SDK error")
	})

	task := NewTask("1", "prompt").WithPreferredMode(ModeHybrid)
	result, err := executor.Execute(context.Background(), task)

	if err != nil {
		t.Errorf("unexpected error after fallback: %v", err)
	}
	if !cliCalled {
		t.Error("CLI should be called as fallback")
	}
	if result != "cli result" {
		t.Errorf("expected 'cli result' from fallback, got %q", result)
	}
}

func TestHybridExecutor_Execute_NilTask(t *testing.T) {
	executor := NewHybridExecutor(nil)

	executor.SetCLIExecutor(func(ctx context.Context, prompt string) (string, error) {
		return "result", nil
	})

	result, err := executor.Execute(context.Background(), nil)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "result" {
		t.Errorf("expected 'result', got %q", result)
	}
}

func TestHybridExecutor_RecordsPerformance(t *testing.T) {
	executor := NewHybridExecutor(nil)

	executor.SetCLIExecutor(func(ctx context.Context, prompt string) (string, error) {
		time.Sleep(10 * time.Millisecond)
		return "result", nil
	})

	task := NewTask("1", "prompt").WithPreferredMode(ModeCLI)
	_, _ = executor.Execute(context.Background(), task)

	execs, _, _ := executor.GetPerformanceMonitor().GetCLIMetrics()
	if execs != 1 {
		t.Errorf("expected 1 recorded execution, got %d", execs)
	}
}

// ========================
// 整合測試
// ========================

func TestExecutionModeSelector_Integration(t *testing.T) {
	selector := NewExecutionModeSelector()

	// 添加規則：高優先級任務使用 SDK
	selector.AddRule(SelectionRule{
		Name:     "high-priority-sdk",
		Priority: 1,
		Condition: func(task *Task) bool {
			return task != nil && task.Priority >= 8
		},
		Mode: ModeSDK,
	})

	// 添加規則：帶特定標籤的任務使用 CLI
	selector.AddRule(SelectionRule{
		Name:     "quick-tag-cli",
		Priority: 2,
		Condition: func(task *Task) bool {
			if task == nil {
				return false
			}
			for _, tag := range task.Tags {
				if tag == "quick" {
					return true
				}
			}
			return false
		},
		Mode: ModeCLI,
	})

	// 測試高優先級任務
	highPriorityTask := NewTask("1", "prompt").WithPriority(9)
	if selector.Choose(highPriorityTask) != ModeSDK {
		t.Error("high priority task should use SDK")
	}

	// 測試帶標籤的任務
	quickTask := NewTask("2", "prompt").WithTags("quick")
	if selector.Choose(quickTask) != ModeCLI {
		t.Error("quick task should use CLI")
	}

	// 測試普通任務
	normalTask := NewTask("3", "prompt")
	if selector.Choose(normalTask) != ModeCLI {
		t.Error("normal simple task should use CLI")
	}
}

func TestHybridExecutor_Integration(t *testing.T) {
	selector := NewExecutionModeSelector()
	executor := NewHybridExecutor(selector)

	cliResults := []string{"cli1", "cli2", "cli3"}
	sdkResults := []string{"sdk1", "sdk2", "sdk3"}
	cliIndex := 0
	sdkIndex := 0

	executor.SetCLIExecutor(func(ctx context.Context, prompt string) (string, error) {
		time.Sleep(1 * time.Millisecond) // 添加小延遲以確保記錄時間
		result := cliResults[cliIndex%len(cliResults)]
		cliIndex++
		return result, nil
	})
	executor.SetSDKExecutor(func(ctx context.Context, prompt string) (string, error) {
		time.Sleep(1 * time.Millisecond) // 添加小延遲以確保記錄時間
		result := sdkResults[sdkIndex%len(sdkResults)]
		sdkIndex++
		return result, nil
	})

	// 執行多個不同類型的任務
	tasks := []*Task{
		NewTask("1", "prompt").WithComplexity(ComplexitySimple),
		NewTask("2", "prompt").WithComplexity(ComplexityComplex),
		NewTask("3", "prompt").WithPreferredMode(ModeCLI),
	}

	for _, task := range tasks {
		_, err := executor.Execute(context.Background(), task)
		if err != nil {
			t.Errorf("unexpected error for task %s: %v", task.ID, err)
		}
	}

	metrics := executor.GetPerformanceMonitor().GetPerformanceMetrics()
	if metrics.CLITime == 0 && metrics.SDKTime == 0 {
		t.Error("expected some execution times to be recorded")
	}
}

func TestExecutionModeSelector_Concurrent(t *testing.T) {
	selector := NewExecutionModeSelector()

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			task := NewTask("task", "prompt")
			if id%2 == 0 {
				task.WithComplexity(ComplexitySimple)
			} else {
				task.WithComplexity(ComplexityComplex)
			}
			_ = selector.Choose(task)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	metrics := selector.GetMetrics()
	if metrics.TotalSelections != 10 {
		t.Errorf("expected 10 selections, got %d", metrics.TotalSelections)
	}
}
