package ghcopilot

import "strings"

// System Prompt 機制
// 參考自 doggy8088/copilot-ralph 專案的 System Prompt 設計
// https://github.com/doggy8088/copilot-ralph
//
// 在使用者的 prompt 前面注入系統指令，告訴 AI：
// 1. 它正在一個自動迭代迴圈中運行
// 2. 任務完成時必須輸出特定的 <promise> 標籤
// 3. 不可以為了逃脫迴圈而假裝完成

const systemPromptTemplate = `# Ralph Loop System Instructions

You are operating inside the **Ralph Loop** auto-iteration system:
- The user prompt will be fed back to you *unchanged* after each response.
- You will see the repo state and any files you modified from previous iterations.
- Your job is to keep iterating until the task is fully complete, then exit with the
  completion phrase (see below).

## Working Rules

1. **Continue from the current repo state** each iteration. Do not repeat the same
   actions if they already happened.
2. **Always make progress**: change files, run checks, or ask for specific missing
   info. If blocked, say exactly what is missing and what you need.
3. **Be concrete**: report what you changed, what you verified, and what remains.
4. **No hallucinations**: never claim edits or test results that did not happen.

## Completion Signal

Only when the task is completely finished:
1. Write a **short summary of all changes** (include files touched).
2. As the **very last text** in your response, output this *exact* phrase:
   "<promise>{{PROMISE}}</promise>"

Requirements for the completion phrase:
- It must be the final characters of the response (no trailing whitespace or text).
- Do not wrap it in a code block or quotes.
- Do not output it unless the task is **fully and verifiably** done.

## Critical Rule

Never output the completion phrase to escape the loop. If you are stuck, blocked,
or waiting on the user, explain the blocker and keep the loop going.`

// BuildSystemPrompt 將 promisePhrase 嵌入 system prompt 模板
func BuildSystemPrompt(promisePhrase string) string {
	return strings.ReplaceAll(systemPromptTemplate, "{{PROMISE}}", promisePhrase)
}

// WrapPromptWithSystemInstructions 在使用者 prompt 前面加上系統指令
func WrapPromptWithSystemInstructions(userPrompt string, promisePhrase string, iteration int, maxIterations int) string {
	var sb strings.Builder
	sb.WriteString(BuildSystemPrompt(promisePhrase))
	sb.WriteString("\n\n---\n\n")
	if maxIterations > 0 {
		sb.WriteString("[Iteration ")
		sb.WriteString(itoa(iteration))
		sb.WriteString("/")
		sb.WriteString(itoa(maxIterations))
		sb.WriteString("]\n\n")
	}
	sb.WriteString(userPrompt)
	return sb.String()
}

// itoa 簡易整數轉字串（避免引入 strconv）
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}
