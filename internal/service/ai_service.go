package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const deepseekAPIURL = "https://api.deepseek.com/v1/chat/completions"

type AIService struct {
	apiKey string
}

func NewAIService(apiKey string) *AIService {
	return &AIService{apiKey: apiKey}
}

// deepseek 请求体
type deepseekRequest struct {
	Model       string        `json:"model"`
	Messages    []DeepseekMsg `json:"messages"`
	Stream      bool          `json:"stream"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type DeepseekMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// deepseek 流式响应中的 data 块
type deepseekStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// ChatStream 调用 DeepSeek 流式 API，将结果通过回调函数逐块返回
// messages: 完整的消息列表（由前端维护，包含 system/user/assistant 历史）
// onChunk: 每次收到内容片段时调用
// 返回完整内容（用于记录历史）
func (s *AIService) ChatStream(messages []DeepseekMsg, onChunk func(text string)) (string, error) {

	reqBody := deepseekRequest{
		Model:       "deepseek-chat",
		Messages:    messages,
		Stream:      true,
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求体失败: %v", err)
	}

	req, err := http.NewRequest("POST", deepseekAPIURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用 DeepSeek API 失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("DeepSeek API 返回错误 (%d): %s", resp.StatusCode, string(respBody))
	}

	// 读取 SSE 流
	scanner := bufio.NewScanner(resp.Body)
	// 增加 buffer 大小以处理长行
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var fullContent string

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk deepseekStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // 跳过无法解析的块
		}

		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta.Content
			if delta != "" {
				fullContent += delta
				onChunk(delta)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fullContent, fmt.Errorf("读取流响应失败: %v", err)
	}

	return fullContent, nil
}

// Chat 非流式版本（备选）
func (s *AIService) Chat(messages []DeepseekMsg) (string, error) {
	var fullContent string
	_, err := s.ChatStream(messages, func(text string) {
		fullContent += text
	})
	return fullContent, err
}
