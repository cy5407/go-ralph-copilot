# 模擬 GitHub Copilot CLI 的行為
# 用於測試 Ralph Loop 的整合邏輯

param(
    [Parameter(Mandatory=$true)]
    [string]$Command,
    
    [Parameter(Mandatory=$false)]
    [string]$Prompt
)

# 模擬處理延遲（AI 思考時間）
Start-Sleep -Milliseconds (Get-Random -Minimum 800 -Maximum 2000)

switch ($Command) {
    "what-the-shell" {
        Write-Host "Suggestion:"
        Write-Host ""
        
        if ($Prompt -match "列出.*go.*檔案") {
            Write-Host "  find . -name '*.go' -type f"
            Write-Host ""
            Write-Host "or"
            Write-Host ""
            Write-Host "  Get-ChildItem -Recurse -Filter *.go"
        }
        elseif ($Prompt -match "修正.*錯誤|fix.*error") {
            Write-Host "Based on the error 'undefined: fmt.Printl', it looks like you have a typo."
            Write-Host ""
            Write-Host "The correct function is:"
            Write-Host "  fmt.Println()"
            Write-Host ""
            Write-Host "Change line 10 in main.go from:"
            Write-Host "  fmt.Printl(`"Hello`")"
            Write-Host "to:"
            Write-Host "  fmt.Println(`"Hello`")"
        }
        else {
            Write-Host "  # Simulated shell command for: $Prompt"
            Write-Host "  echo 'This is a mock response'"
        }
    }
    
    "git-assist" {
        Write-Host "Git command suggestion:"
        Write-Host ""
        Write-Host "  git $Prompt"
    }
    
    "gh-assist" {
        Write-Host "GitHub CLI command suggestion:"
        Write-Host ""
        Write-Host "  gh $Prompt"
    }
    
    "--version" {
        Write-Host "0.1.36-mock"
    }
    
    default {
        Write-Host "Unknown command: $Command"
        exit 1
    }
}

exit 0
