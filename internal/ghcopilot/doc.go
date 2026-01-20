// Package ghcopilot 提供 GitHub Copilot CLI 整合層
//
// 該套件封裝了所有與 GitHub Copilot CLI 的互動，包括：
// - 依賴檢查（Node.js、github-copilot-cli、GitHub CLI、認證）
// - CLI 執行與結果捕獲
// - 輸出解析
// - 上下文管理
// - 回應分析（含完成偵測和卡住偵測）
// - 熔斷機制（防止失控迴圈）
// - 優雅退出決策（雙重條件驗證）
package ghcopilot

// Version 是目前套件的版本
const Version = "0.1.0"
