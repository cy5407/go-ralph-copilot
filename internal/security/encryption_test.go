package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEncryptionManager_Basic(t *testing.T) {
	password := "test-password-123"
	em := NewEncryptionManager(password)
	
	testData := "這是測試數據，包含中文和特殊符號：!@#$%^&*()"
	
	// 測試加密
	encrypted, err := em.EncryptString(testData)
	if err != nil {
		t.Fatalf("加密失敗: %v", err)
	}
	
	if encrypted == testData {
		t.Fatal("加密後的數據應該與原數據不同")
	}
	
	if !IsEncrypted(encrypted) {
		t.Fatal("IsEncrypted 應該能識別加密的數據")
	}
	
	// 測試解密
	decrypted, err := em.DecryptString(encrypted)
	if err != nil {
		t.Fatalf("解密失敗: %v", err)
	}
	
	if decrypted != testData {
		t.Fatalf("解密結果不匹配，期望: %s，實際: %s", testData, decrypted)
	}
}

func TestEncryptionManager_EmptyString(t *testing.T) {
	em := NewEncryptionManager("password")
	
	// 測試空字符串
	encrypted, err := em.EncryptString("")
	if err != nil {
		t.Fatalf("空字符串加密失敗: %v", err)
	}
	
	if encrypted != "" {
		t.Fatal("空字符串加密應該返回空字符串")
	}
	
	decrypted, err := em.DecryptString("")
	if err != nil {
		t.Fatalf("空字符串解密失敗: %v", err)
	}
	
	if decrypted != "" {
		t.Fatal("空字符串解密應該返回空字符串")
	}
}

func TestEncryptionManager_DifferentPasswords(t *testing.T) {
	testData := "敏感數據"
	
	em1 := NewEncryptionManager("password1")
	em2 := NewEncryptionManager("password2")
	
	// 使用不同密碼加密
	encrypted, err := em1.EncryptString(testData)
	if err != nil {
		t.Fatalf("加密失敗: %v", err)
	}
	
	// 嘗試用不同密碼解密（應該失敗）
	_, err = em2.DecryptString(encrypted)
	if err == nil {
		t.Fatal("使用不同密碼解密應該失敗")
	}
}

func TestEncryptionManager_FileOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ralph-security-test")
	if err != nil {
		t.Fatalf("創建臨時目錄失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	em := NewEncryptionManager("file-test-password")
	
	// 創建測試文件
	testContent := "這是測試文件內容\n包含多行數據\n以及特殊字符：!@#$%^&*()"
	inputFile := filepath.Join(tempDir, "input.txt")
	encryptedFile := filepath.Join(tempDir, "encrypted.txt")
	outputFile := filepath.Join(tempDir, "output.txt")
	
	// 寫入測試文件
	err = os.WriteFile(inputFile, []byte(testContent), 0600)
	if err != nil {
		t.Fatalf("寫入測試文件失敗: %v", err)
	}
	
	// 測試文件加密
	err = em.EncryptFile(inputFile, encryptedFile)
	if err != nil {
		t.Fatalf("文件加密失敗: %v", err)
	}
	
	// 驗證加密文件存在
	if _, err := os.Stat(encryptedFile); os.IsNotExist(err) {
		t.Fatal("加密文件不存在")
	}
	
	// 測試文件解密
	err = em.DecryptFile(encryptedFile, outputFile)
	if err != nil {
		t.Fatalf("文件解密失敗: %v", err)
	}
	
	// 驗證解密結果
	decryptedContent, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("讀取解密文件失敗: %v", err)
	}
	
	if string(decryptedContent) != testContent {
		t.Fatalf("解密文件內容不匹配，期望: %s，實際: %s", testContent, string(decryptedContent))
	}
}

func TestIsEncrypted(t *testing.T) {
	testCases := []struct {
		data     string
		expected bool
	}{
		{"", false},                           // 空字符串
		{"plaintext", false},                  // 普通文本
		{"short", false},                      // 過短的字符串
		{"not-base64-data", false},           // 非base64數據
		{"dGVzdA==", false},                  // 有效base64但太短
		{"dGhpcyBpcyBhIHRlc3QgZm9yIGVuY3J5cHRpb24=", false}, // 有效base64但未加密
	}
	
	for _, tc := range testCases {
		result := IsEncrypted(tc.data)
		if result != tc.expected {
			t.Errorf("IsEncrypted(%q) = %v, 期望 %v", tc.data, result, tc.expected)
		}
	}
	
	// 測試真正的加密數據
	em := NewEncryptionManager("test")
	encrypted, err := em.EncryptString("test data")
	if err != nil {
		t.Fatalf("加密測試數據失敗: %v", err)
	}
	
	if !IsEncrypted(encrypted) {
		t.Fatal("IsEncrypted 應該能識別真正的加密數據")
	}
}

func TestMaskSensitiveInfo(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{
			"password=mysecret123",
			"password=***MASKED***",
		},
		{
			"api_key: sk-1234567890abcdef",
			"api_key: ***MASKED***",
		},
		{
			"username=john\npassword=secret\nhost=localhost",
			"username=john\npassword=***MASKED***\nhost=localhost",
		},
		{
			"普通文本，沒有敏感資訊",
			"普通文本，沒有敏感資訊",
		},
		{
			"multiple secrets: password=123 and token=abc",
			"multiple secrets: password=***MASKED*** and token=***MASKED***",
		},
	}
	
	for i, tc := range testCases {
		result := MaskSensitiveInfo(tc.input)
		if result != tc.expected {
			t.Errorf("測試案例 %d 失敗:\n輸入: %s\n期望: %s\n實際: %s", 
				i+1, tc.input, tc.expected, result)
		}
	}
}

func TestGetDefaultPassword(t *testing.T) {
	// 測試環境變數優先級
	originalEnv := os.Getenv("RALPH_ENCRYPTION_PASSWORD")
	defer func() {
		if originalEnv != "" {
			os.Setenv("RALPH_ENCRYPTION_PASSWORD", originalEnv)
		} else {
			os.Unsetenv("RALPH_ENCRYPTION_PASSWORD")
		}
	}()
	
	// 設定環境變數
	testPassword := "env-test-password"
	os.Setenv("RALPH_ENCRYPTION_PASSWORD", testPassword)
	
	password := GetDefaultPassword()
	if password != testPassword {
		t.Errorf("期望從環境變數獲取密碼 %s，實際獲取 %s", testPassword, password)
	}
	
	// 清除環境變數，測試預設行為
	os.Unsetenv("RALPH_ENCRYPTION_PASSWORD")
	
	defaultPassword := GetDefaultPassword()
	if !strings.HasPrefix(defaultPassword, "ralph-loop-") || !strings.HasSuffix(defaultPassword, "-default") {
		t.Errorf("預設密碼格式不正確: %s", defaultPassword)
	}
}

func TestEncryptionConsistency(t *testing.T) {
	// 測試相同密碼和數據的多次加密是否能正確解密
	password := "consistency-test"
	testData := "一致性測試數據"
	
	em := NewEncryptionManager(password)
	
	// 多次加密同樣的數據
	results := make([]string, 5)
	for i := 0; i < 5; i++ {
		encrypted, err := em.EncryptString(testData)
		if err != nil {
			t.Fatalf("第 %d 次加密失敗: %v", i+1, err)
		}
		results[i] = encrypted
		
		// 驗證每次加密的結果都不同（由於隨機nonce）
		for j := 0; j < i; j++ {
			if results[i] == results[j] {
				t.Fatal("相同數據的多次加密結果應該不同（隨機nonce）")
			}
		}
		
		// 驗證都能正確解密
		decrypted, err := em.DecryptString(encrypted)
		if err != nil {
			t.Fatalf("第 %d 次解密失敗: %v", i+1, err)
		}
		if decrypted != testData {
			t.Fatalf("第 %d 次解密結果不正確", i+1)
		}
	}
}