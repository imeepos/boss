package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ai 命令组:OpenAI 能力网关(config/chat/embed)。
// 用法:
//
//	bossctl ai config [--url URL] [--key KEY] [--model MODEL]   查看/更新集中配置
//	bossctl ai chat [--model MODEL] "prompt"                    对话补全
//	bossctl ai embed [--model MODEL] "text" ...                 文本向量化
func (c *CLI) ai(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("用法: bossctl ai config|chat|embed")
	}
	switch args[0] {
	case "config":
		return c.aiConfig(args[1:])
	case "chat":
		return c.aiChat(args[1:])
	case "embed":
		return c.aiEmbed(args[1:])
	default:
		return fmt.Errorf("未知 ai 命令: %s", args[0])
	}
}

// aiConfig 无参数=查看脱敏配置;带 --url/--key/--model=更新(仅传要改的键)。
func (c *CLI) aiConfig(args []string) error {
	upd := map[string]any{}
	for i := 0; i < len(args); i++ {
		v := ""
		if i+1 < len(args) {
			v = args[i+1]
		}
		switch args[i] {
		case "--url":
			upd["apiUrl"] = v
			i++
		case "--key":
			upd["apiKey"] = v
			i++
		case "--model":
			upd["model"] = v
			i++
		}
	}
	if len(upd) == 0 {
		resp, err := c.do("GET", "/api/v1/ai/openai/config", nil, nil)
		if err != nil {
			return err
		}
		if resp.Code != 0 {
			return fmt.Errorf("读取配置失败: code=%d msg=%s", resp.Code, resp.Msg)
		}
		printJSON(resp.Data)
		return nil
	}
	resp, err := c.do("PUT", "/api/v1/ai/openai/config", upd, nil)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("更新配置失败: code=%d msg=%s", resp.Code, resp.Msg)
	}
	fmt.Println("配置已更新(立即生效)")
	return nil
}

// aiChat 对话补全,输出回答文本与用量。
func (c *CLI) aiChat(args []string) error {
	model, prompt, err := parseAIArgs("chat", args)
	if err != nil {
		return err
	}
	body := map[string]any{
		"messages": []map[string]string{{"role": "user", "content": prompt}},
	}
	if model != "" {
		body["model"] = model
	}
	resp, err := c.do("POST", "/api/v1/ai/chat/completions", body, nil)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("chat 失败: code=%d msg=%s", resp.Code, resp.Msg)
	}
	var data struct {
		Model   string `json:"model"`
		Content string `json:"content"`
		Usage   struct {
			TotalTokens int64 `json:"totalTokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return fmt.Errorf("解析响应: %w", err)
	}
	fmt.Println(data.Content)
	fmt.Fprintf(os.Stderr, "[model=%s tokens=%d]\n", data.Model, data.Usage.TotalTokens)
	return nil
}

// aiEmbed 文本向量化,输出 JSON(向量与输入顺序一一对应)。
func (c *CLI) aiEmbed(args []string) error {
	model, text, err := parseAIArgs("embed", args)
	if err != nil {
		return err
	}
	body := map[string]any{"input": strings.Fields(text)}
	if model != "" {
		body["model"] = model
	}
	resp, err := c.do("POST", "/api/v1/ai/embeddings", body, nil)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("embed 失败: code=%d msg=%s", resp.Code, resp.Msg)
	}
	printJSON(resp.Data)
	return nil
}

// parseAIArgs 解析 [--model M] 与剩余位置参数(至少一个)。
func parseAIArgs(cmd string, args []string) (model, text string, err error) {
	var pos []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--model" && i+1 < len(args) {
			model = args[i+1]
			i++
			continue
		}
		pos = append(pos, args[i])
	}
	if len(pos) == 0 {
		return "", "", fmt.Errorf("用法: bossctl ai %s [--model MODEL] <文本>", cmd)
	}
	return model, strings.Join(pos, " "), nil
}
