package metrics

import (
	"math"
	"testing"
	"time"
)

func TestCounterMetric(t *testing.T) {
	counter := NewCounter("test_counter")
	
	if counter.Name() != "test_counter" {
		t.Errorf("期望名稱為 'test_counter'，但得到: %s", counter.Name())
	}
	
	if counter.Type() != Counter {
		t.Errorf("期望類型為 Counter，但得到: %s", counter.Type())
	}
	
	if counter.Get() != 0 {
		t.Errorf("初始值應該為 0，但得到: %d", counter.Get())
	}
	
	counter.Inc()
	if counter.Get() != 1 {
		t.Errorf("遞增後應該為 1，但得到: %d", counter.Get())
	}
	
	counter.Add(5)
	if counter.Get() != 6 {
		t.Errorf("加 5 後應該為 6，但得到: %d", counter.Get())
	}
	
	value := counter.Value().(int64)
	if value != 6 {
		t.Errorf("Value() 應該返回 6，但得到: %d", value)
	}
	
	counter.Reset()
	if counter.Get() != 0 {
		t.Errorf("重置後應該為 0，但得到: %d", counter.Get())
	}
}

func TestGaugeMetric(t *testing.T) {
	gauge := NewGauge("test_gauge")
	
	if gauge.Name() != "test_gauge" {
		t.Errorf("期望名稱為 'test_gauge'，但得到: %s", gauge.Name())
	}
	
	if gauge.Type() != Gauge {
		t.Errorf("期望類型為 Gauge，但得到: %s", gauge.Type())
	}
	
	if gauge.Get() != 0 {
		t.Errorf("初始值應該為 0，但得到: %f", gauge.Get())
	}
	
	gauge.Set(3.14)
	if gauge.Get() != 3.14 {
		t.Errorf("設置後應該為 3.14，但得到: %f", gauge.Get())
	}
	
	gauge.Inc()
	if math.Abs(gauge.Get()-4.14) > 0.0001 {
		t.Errorf("遞增後應該為 4.14，但得到: %f", gauge.Get())
	}
	
	gauge.Dec()
	if math.Abs(gauge.Get()-3.14) > 0.0001 {
		t.Errorf("遞減後應該為 3.14，但得到: %f", gauge.Get())
	}
	
	gauge.Add(-1.14)
	if math.Abs(gauge.Get()-2.0) > 0.0001 {
		t.Errorf("減 1.14 後應該為 2.0，但得到: %f", gauge.Get())
	}
	
	value := gauge.Value().(float64)
	if math.Abs(value-2.0) > 0.0001 {
		t.Errorf("Value() 應該返回 2.0，但得到: %f", value)
	}
}

func TestTimerMetric(t *testing.T) {
	timer := NewTimer("test_timer")
	
	if timer.Name() != "test_timer" {
		t.Errorf("期望名稱為 'test_timer'，但得到: %s", timer.Name())
	}
	
	if timer.Type() != Timer {
		t.Errorf("期望類型為 Timer，但得到: %s", timer.Type())
	}
	
	// 測試空計時器
	value := timer.Value().(map[string]interface{})
	if value["count"].(int) != 0 {
		t.Errorf("初始計數應該為 0，但得到: %v", value["count"])
	}
	
	// 記錄一些時間值
	timer.Record(100 * time.Millisecond)
	timer.Record(200 * time.Millisecond)
	timer.Record(150 * time.Millisecond)
	
	value = timer.Value().(map[string]interface{})
	if value["count"].(int) != 3 {
		t.Errorf("記錄 3 個值後計數應該為 3，但得到: %v", value["count"])
	}
	
	if value["min"].(int64) != 100 {
		t.Errorf("最小值應該為 100ms，但得到: %v", value["min"])
	}
	
	if value["max"].(int64) != 200 {
		t.Errorf("最大值應該為 200ms，但得到: %v", value["max"])
	}
	
	if value["avg"].(int64) != 150 {
		t.Errorf("平均值應該為 150ms，但得到: %v", value["avg"])
	}
}

func TestTimerMetric_Time(t *testing.T) {
	timer := NewTimer("test_timer_func")
	
	timer.Time(func() {
		time.Sleep(10 * time.Millisecond)
	})
	
	value := timer.Value().(map[string]interface{})
	if value["count"].(int) != 1 {
		t.Errorf("Time() 後計數應該為 1，但得到: %v", value["count"])
	}
	
	// 檢查記錄的時間至少 10ms
	if value["min"].(int64) < 10 {
		t.Errorf("記錄的時間應該至少 10ms，但得到: %v", value["min"])
	}
}

func TestTimerMetric_StartStop(t *testing.T) {
	timer := NewTimer("test_timer_start_stop")
	
	stop := timer.Start()
	time.Sleep(10 * time.Millisecond)
	stop()
	
	value := timer.Value().(map[string]interface{})
	if value["count"].(int) != 1 {
		t.Errorf("Start/Stop 後計數應該為 1，但得到: %v", value["count"])
	}
	
	if value["min"].(int64) < 10 {
		t.Errorf("記錄的時間應該至少 10ms，但得到: %v", value["min"])
	}
}

func TestLoopMetrics(t *testing.T) {
	metrics := NewLoopMetrics()
	
	// 測試計數器
	if metrics.TotalLoops.Get() != 0 {
		t.Error("初始 TotalLoops 應該為 0")
	}
	
	metrics.TotalLoops.Inc()
	if metrics.TotalLoops.Get() != 1 {
		t.Error("TotalLoops 遞增後應該為 1")
	}
	
	// 測試錯誤率計算
	metrics.FailedLoops.Add(2)
	metrics.UpdateErrorRate()
	
	expectedRate := float64(2) / float64(1) * 100 // 200%
	if metrics.ErrorRate.Get() != expectedRate {
		t.Errorf("期望錯誤率為 %f，但得到: %f", expectedRate, metrics.ErrorRate.Get())
	}
}

func TestMetricsCollector(t *testing.T) {
	collector := NewCollector()
	
	// 測試註冊指標
	counter := NewCounter("test_counter")
	collector.Register(counter)
	
	retrieved := collector.Get("test_counter")
	if retrieved == nil {
		t.Error("註冊後應該能夠獲取指標")
	}
	
	if retrieved.Name() != "test_counter" {
		t.Errorf("期望指標名稱為 'test_counter'，但得到: %s", retrieved.Name())
	}
	
	// 測試取消註冊
	collector.Unregister("test_counter")
	retrieved = collector.Get("test_counter")
	if retrieved != nil {
		t.Error("取消註冊後不應該能夠獲取指標")
	}
	
	// 測試 GetAll
	allMetrics := collector.GetAll()
	if len(allMetrics) == 0 {
		t.Error("應該至少有一些預設的 LoopMetrics")
	}
	
	// 測試摘要
	summary := collector.GetSummary()
	if summary.TotalMetrics == 0 {
		t.Error("摘要應該包含一些指標")
	}
	
	if summary.Uptime == "" {
		t.Error("摘要應該包含執行時間")
	}
}

func TestSummary_ToJSON(t *testing.T) {
	collector := NewCollector()
	summary := collector.GetSummary()
	
	json, err := summary.ToJSON()
	if err != nil {
		t.Errorf("轉換為 JSON 失敗: %v", err)
	}
	
	if json == "" {
		t.Error("JSON 輸出不應該為空")
	}
	
	if !contains(json, "timestamp") {
		t.Error("JSON 應該包含 timestamp 字段")
	}
}

func TestSummary_ToText(t *testing.T) {
	collector := NewCollector()
	
	// 添加一些測試數據
	collector.GetLoopMetrics().TotalLoops.Add(5)
	collector.GetLoopMetrics().SuccessfulLoops.Add(3)
	collector.GetLoopMetrics().FailedLoops.Add(2)
	collector.GetLoopMetrics().UpdateErrorRate()
	
	summary := collector.GetSummary()
	text := summary.ToText()
	
	if text == "" {
		t.Error("文字輸出不應該為空")
	}
	
	if !contains(text, "Ralph Loop 指標摘要") {
		t.Error("文字輸出應該包含標題")
	}
	
	if !contains(text, "📊 計數器") {
		t.Error("文字輸出應該包含計數器部分")
	}
	
	if !contains(text, "📈 標量") {
		t.Error("文字輸出應該包含標量部分")
	}
}

func TestGlobalFunctions(t *testing.T) {
	// 重置全域指標
	ResetGlobalMetrics()
	
	// 測試記錄迴圈
	stopTimer := RecordLoopStart()
	time.Sleep(1 * time.Millisecond)
	RecordLoopSuccess(stopTimer)
	
	// 檢查指標
	if GlobalCollector.GetLoopMetrics().TotalLoops.Get() != 1 {
		t.Error("應該記錄一個迴圈")
	}
	
	if GlobalCollector.GetLoopMetrics().SuccessfulLoops.Get() != 1 {
		t.Error("應該記錄一個成功迴圈")
	}
	
	// 測試失敗迴圈
	stopTimer2 := RecordLoopStart()
	RecordLoopFailure(stopTimer2)
	
	if GlobalCollector.GetLoopMetrics().FailedLoops.Get() != 1 {
		t.Error("應該記錄一個失敗迴圈")
	}
	
	// 測試其他記錄函數
	RecordRetryAttempt()
	if GlobalCollector.GetLoopMetrics().RetryAttempts.Get() != 1 {
		t.Error("應該記錄一次重試")
	}
	
	RecordCLIExecution(100 * time.Millisecond)
	if GlobalCollector.GetLoopMetrics().CLIExecutions.Get() != 1 {
		t.Error("應該記錄一次 CLI 執行")
	}
	
	RecordSDKExecution(50 * time.Millisecond)
	if GlobalCollector.GetLoopMetrics().SDKExecutions.Get() != 1 {
		t.Error("應該記錄一次 SDK 執行")
	}
	
	RecordCircuitBreakerTrip()
	if GlobalCollector.GetLoopMetrics().CircuitBreakerTrips.Get() != 1 {
		t.Error("應該記錄一次熔斷器觸發")
	}
	
	SetCircuitBreakerState(1)
	if GlobalCollector.GetLoopMetrics().CircuitBreakerState.Get() != 1.0 {
		t.Error("應該設置熔斷器狀態為 1")
	}
}

// 輔助函數
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
			(s[:len(substr)] == substr || 
			 s[len(s)-len(substr):] == substr ||
			 containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}