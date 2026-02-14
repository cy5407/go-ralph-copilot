package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// 加密參數
	saltLength  = 16
	keyLength   = 32 // AES-256
	nonceLength = 12 // GCM nonce
	iterations  = 100000
)

// EncryptionManager 管理加密操作
type EncryptionManager struct {
	masterKey []byte
}

// NewEncryptionManager 創建新的加密管理器
func NewEncryptionManager(password string) *EncryptionManager {
	// 從系統生成或讀取 salt
	salt := getOrCreateSalt()
	
	// 使用 PBKDF2 生成主密鑰
	masterKey := pbkdf2.Key([]byte(password), salt, iterations, keyLength, sha256.New)
	
	return &EncryptionManager{
		masterKey: masterKey,
	}
}

// EncryptString 加密字符串
func (em *EncryptionManager) EncryptString(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	
	// 創建 AES cipher
	block, err := aes.NewCipher(em.masterKey)
	if err != nil {
		return "", fmt.Errorf("創建 cipher 失敗: %w", err)
	}
	
	// 創建 GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("創建 GCM 失敗: %w", err)
	}
	
	// 生成隨機 nonce
	nonce := make([]byte, nonceLength)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成 nonce 失敗: %w", err)
	}
	
	// 加密
	ciphertext := aesGCM.Seal(nil, nonce, []byte(plaintext), nil)
	
	// 組合 nonce + ciphertext 並 base64 編碼
	encrypted := append(nonce, ciphertext...)
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptString 解密字符串
func (em *EncryptionManager) DecryptString(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	
	// Base64 解碼
	encrypted, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("base64 解碼失敗: %w", err)
	}
	
	// 檢查長度
	if len(encrypted) < nonceLength {
		return "", fmt.Errorf("加密數據過短")
	}
	
	// 分離 nonce 和密文
	nonce := encrypted[:nonceLength]
	cipherData := encrypted[nonceLength:]
	
	// 創建 AES cipher
	block, err := aes.NewCipher(em.masterKey)
	if err != nil {
		return "", fmt.Errorf("創建 cipher 失敗: %w", err)
	}
	
	// 創建 GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("創建 GCM 失敗: %w", err)
	}
	
	// 解密
	plaintext, err := aesGCM.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", fmt.Errorf("解密失敗: %w", err)
	}
	
	return string(plaintext), nil
}

// EncryptFile 加密文件
func (em *EncryptionManager) EncryptFile(inputPath, outputPath string) error {
	// 讀取原文件
	plaintext, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("讀取文件失敗: %w", err)
	}
	
	// 加密內容
	encrypted, err := em.EncryptString(string(plaintext))
	if err != nil {
		return fmt.Errorf("加密內容失敗: %w", err)
	}
	
	// 確保輸出目錄存在
	if err := os.MkdirAll(filepath.Dir(outputPath), 0700); err != nil {
		return fmt.Errorf("創建目錄失敗: %w", err)
	}
	
	// 寫入加密文件
	if err := os.WriteFile(outputPath, []byte(encrypted), 0600); err != nil {
		return fmt.Errorf("寫入加密文件失敗: %w", err)
	}
	
	return nil
}

// DecryptFile 解密文件
func (em *EncryptionManager) DecryptFile(inputPath, outputPath string) error {
	// 讀取加密文件
	encrypted, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("讀取加密文件失敗: %w", err)
	}
	
	// 解密內容
	plaintext, err := em.DecryptString(string(encrypted))
	if err != nil {
		return fmt.Errorf("解密內容失敗: %w", err)
	}
	
	// 確保輸出目錄存在
	if err := os.MkdirAll(filepath.Dir(outputPath), 0700); err != nil {
		return fmt.Errorf("創建目錄失敗: %w", err)
	}
	
	// 寫入解密文件
	if err := os.WriteFile(outputPath, []byte(plaintext), 0600); err != nil {
		return fmt.Errorf("寫入解密文件失敗: %w", err)
	}
	
	return nil
}

// IsEncrypted 檢查字符串是否為加密格式
func IsEncrypted(data string) bool {
	// 加密的數據應該是有效的 base64 且長度合理
	if len(data) < 32 { // 至少需要容納 nonce(12) + 最小密文(1) + 認證標籤(16) = 29字節，base64約40字符
		return false
	}
	
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return false
	}
	
	// 檢查解碼後的長度是否符合 AES-GCM 加密結構
	// nonce(12) + 至少1字節密文 + GCM標籤(16) = 至少29字節
	if len(decoded) < nonceLength+1+16 {
		return false
	}
	
	// 簡單檢查：解碼後的內容不應該是可讀的 ASCII 文字
	// 檢查前16字節，如果大部分是可打印的ASCII字符，可能不是加密數據
	printableCount := 0
	sampleSize := 16
	if len(decoded) < sampleSize {
		sampleSize = len(decoded)
	}
	
	for i := 0; i < sampleSize; i++ {
		if decoded[i] >= 32 && decoded[i] <= 126 {
			printableCount++
		}
	}
	
	// 如果超過75%的字節都是可打印字符，可能是普通文字而非加密數據
	if float64(printableCount)/float64(sampleSize) > 0.75 {
		return false
	}
	
	return true
}

// hasRandomDistribution 檢查數據是否具有隨機分布特徵
func hasRandomDistribution(data []byte) bool {
	if len(data) < 8 { // 降低最小要求
		return false
	}
	
	// 計算字節值的分布
	counts := make([]int, 256)
	for _, b := range data {
		counts[b]++
	}
	
	// 檢查是否有過多重複的字節值（較寬鬆的熵檢查）
	// 對於較短的數據，允許更高的重複率
	maxCount := len(data) / 2 // 允許某個字節值出現最多50%
	if len(data) > 32 {
		maxCount = len(data) / 3 // 較長數據更嚴格，最多33%
	}
	if len(data) > 64 {
		maxCount = len(data) / 4 // 長數據最嚴格，最多25%
	}
	
	for _, count := range counts {
		if count > maxCount {
			return false
		}
	}
	
	// 額外檢查：不能有太多連續的相同字節
	consecutiveCount := 1
	maxConsecutive := 3 // 最多3個連續相同字節
	
	for i := 1; i < len(data); i++ {
		if data[i] == data[i-1] {
			consecutiveCount++
			if consecutiveCount > maxConsecutive {
				return false
			}
		} else {
			consecutiveCount = 1
		}
	}
	
	return true
}

// MaskSensitiveInfo 遮罩敏感資訊
func MaskSensitiveInfo(data string) string {
	// 使用正則表達式來匹配和替換敏感資訊
	sensitivePatterns := []string{
		"password", "secret", "key", "token", "auth", "credential",
		"api_key", "access_key", "private_key", "bearer",
	}
	
	result := data
	
	// 對每個模式進行全域替換
	for _, pattern := range sensitivePatterns {
		// 匹配 pattern=value 格式
		re1 := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(pattern) + `\s*=\s*[^\s]+`)
		result = re1.ReplaceAllStringFunc(result, func(match string) string {
			parts := strings.SplitN(match, "=", 2)
			if len(parts) == 2 {
				return parts[0] + "=***MASKED***"
			}
			return match
		})
		
		// 匹配 pattern: value 格式
		re2 := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(pattern) + `\s*:\s*[^\s]+`)
		result = re2.ReplaceAllStringFunc(result, func(match string) string {
			parts := strings.SplitN(match, ":", 2)
			if len(parts) == 2 {
				return parts[0] + ": ***MASKED***"
			}
			return match
		})
	}
	
	return result
}

// getOrCreateSalt 獲取或創建系統 salt
func getOrCreateSalt() []byte {
	// 使用固定的系統位置儲存 salt
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// 如果無法獲取 home 目錄，使用當前目錄
		homeDir = "."
	}
	
	saltPath := filepath.Join(homeDir, ".ralph-loop", "salt")
	
	// 嘗試讀取現有的 salt
	if salt, err := os.ReadFile(saltPath); err == nil && len(salt) == saltLength {
		return salt
	}
	
	// 生成新的 salt
	salt := make([]byte, saltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		// 如果隨機生成失敗，使用確定性的 salt（不太安全，但仍可用）
		copy(salt, []byte("ralph-loop-salt-"))
	}
	
	// 確保目錄存在
	os.MkdirAll(filepath.Dir(saltPath), 0700)
	
	// 儲存 salt（忽略錯誤，因為可能是權限問題）
	os.WriteFile(saltPath, salt, 0600)
	
	return salt
}

// GetDefaultPassword 獲取預設加密密碼
func GetDefaultPassword() string {
	// 優先使用環境變數
	if password := os.Getenv("RALPH_ENCRYPTION_PASSWORD"); password != "" {
		return password
	}
	
	// 使用機器特定的識別符作為預設密碼
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}
	
	// 結合主機名和固定字符串
	return fmt.Sprintf("ralph-loop-%s-default", hostname)
}