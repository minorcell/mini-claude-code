package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// Message 是发给模型的最小消息结构。
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Config 保存模型调用所需的配置。
type Config struct {
	BaseURL string
	APIKey  string
	Model   string
}

var (
	actionRe = regexp.MustCompile(`(?s)<action tool="([^"]+)">(.*?)</action>`)
	finalRe  = regexp.MustCompile(`(?s)<final>(.*?)</final>`)
)

// loadConfig 从环境变量读取模型配置。
func loadConfig() (Config, error) {
	cfg := Config{
		BaseURL: strings.TrimRight(firstEnv(os.Getenv("LLM_BASE_URL"), "https://api.deepseek.com/v1"), "/"),
		APIKey:  firstEnv(os.Getenv("LLM_API_KEY"), os.Getenv("DEEPSEEK_API_KEY")),
		Model:   firstEnv(os.Getenv("LLM_MODEL"), "deepseek-chat"),
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return Config{}, fmt.Errorf("请设置 LLM_API_KEY 或 DEEPSEEK_API_KEY")
	}
	return cfg, nil
}

// firstEnv 返回第一个非空字符串。
func firstEnv(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// callModel 调用兼容 OpenAI 的 /chat/completions 接口。
func callModel(cfg Config, messages []Message) (string, error) {
	reqBody, err := json.Marshal(map[string]any{
		"model":    cfg.Model,
		"messages": messages,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, cfg.BaseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("模型调用失败：%s %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var data struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	if len(data.Choices) == 0 || strings.TrimSpace(data.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("模型返回为空")
	}
	return strings.TrimSpace(data.Choices[0].Message.Content), nil
}

// parseReply 从模型输出里提取 action 或 final。
func parseReply(text string) (tool string, input string, final string) {
	if m := finalRe.FindStringSubmatch(text); len(m) == 2 {
		final = strings.TrimSpace(m[1])
	}
	if m := actionRe.FindStringSubmatch(text); len(m) == 3 {
		tool = strings.TrimSpace(m[1])
		input = strings.TrimSpace(m[2])
	}
	return
}

// wrapObservation 把工具结果包装成 XML，避免特殊字符破坏协议。
func wrapObservation(text string) string {
	var buf bytes.Buffer
	buf.WriteString("<observation>")
	_ = xml.EscapeText(&buf, []byte(text))
	buf.WriteString("</observation>")
	return buf.String()
}

// run 是最小 loop：模型决定要不要调工具，工具结果再喂回模型。
func run(question string, cfg Config, root string) (string, error) {
	prompt, err := os.ReadFile("prompt.md")
	if err != nil {
		return "", err
	}

	history := []Message{
		{Role: "system", Content: string(prompt)},
		{Role: "user", Content: question},
	}

	for step := 1; step <= 8; step++ {
		reply, err := callModel(cfg, history)
		if err != nil {
			return "", err
		}

		fmt.Printf("\n[第 %d 轮]\n%s\n", step, reply)
		history = append(history, Message{Role: "assistant", Content: reply})

		tool, input, final := parseReply(reply)
		if final != "" {
			return final, nil
		}
		if tool == "" {
			return "", fmt.Errorf("模型输出不符合约定")
		}

		observation := wrapObservation(callTool(root, tool, input))
		fmt.Println(observation)
		history = append(history, Message{Role: "user", Content: observation})
	}

	return "", fmt.Errorf("超过最大轮数，仍未得到最终答案")
}

func main() {
	question := strings.TrimSpace(strings.Join(os.Args[1:], " "))
	if question == "" {
		question = "请列出当前目录，并解释这个最小 Agent 的工作流程。"
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("问题:", question)
	answer, err := run(question, cfg, root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "运行失败:", err)
		os.Exit(1)
	}

	fmt.Println("\n=== 最终回答 ===")
	fmt.Println(answer)
}
