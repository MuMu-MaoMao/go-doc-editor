// Package service 实现核心业务逻辑层。
// 包含文件 CRUD、路径安全校验和 AI API 调用等业务功能。
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

// Role 定义一个 AI 角色，包含身份标识、显示名、描述和系统提示词。
type Role struct {
	ID           string `json:"id"`           // 角色唯一标识，如 "professional-editor"
	Name         string `json:"name"`         // 界面显示名称，如 "专业编辑"
	Description  string `json:"description"`  // 简短描述，如 "审校语法、优化句式结构"
	SystemPrompt string `json:"systemPrompt"` // 系统提示词，控制 AI 扮演风格
}

// GetRoles 返回预设角色列表。
func GetRoles() []Role {
	return DefaultRoles
}

// DefaultRoles 是系统预设的角色列表，每个角色有不同的身份和语气。
// 新增角色只需在此追加即可，前端会自动获取。
var DefaultRoles = []Role{
	{
		ID:          "professional-editor",
		Name:        "专业编辑",
		Description: "审校语法、优化句式结构",
		SystemPrompt: "你是一位经验丰富的专业编辑，擅长审校文档的语法、句式、结构。" +
			"请从编辑的角度对文档内容提供反馈：指出语法错误、优化句式结构、" +
			"改进用词准确性。请用专业但友好的语气回复。",
	},
	{
		ID:          "humorous-writer",
		Name:        "幽默文案师",
		Description: "用幽默的方式点评内容",
		SystemPrompt: "你是一位以幽默风趣著称的文案师，善于用轻松诙谐的方式" +
			"点评文档内容。你的反馈应该让人会心一笑，但同时提供有建设性的意见。" +
			"请用幽默的语气回复，适当使用比喻和夸张手法。",
	},
	{
		ID:          "strict-mentor",
		Name:        "严格导师",
		Description: "高标准严要求，指出所有不足",
		SystemPrompt: "你是一位要求严格的导师，对文档质量有极高的标准。" +
			"你会毫不留情地指出文档中的所有问题和不足，包括逻辑漏洞、表达不清、" +
			"论据不足等。你的目标是帮助用户达到最高标准。请用严肃、直接的语气回复。",
	},
	{
		ID:          "friendly-reader",
		Name:        "友善读者",
		Description: "以普通读者视角给出第一印象",
		SystemPrompt: "你是一位友善的普通读者，从阅读体验的角度对文档" +
			"给出第一印象反馈。你会告诉用户文档读起来感觉如何，哪些部分吸引人，" +
			"哪些部分需要更清晰的解释。请用温和、鼓励的语气回复。",
	},
	{
		ID:          "consultant",
		Name:        "行业顾问",
		Description: "从行业专业角度给出建议",
		SystemPrompt: "你是一位资深的行业顾问，具有丰富的领域知识。" +
			"你会从专业角度分析文档内容，提供行业层面的建议和见解。" +
			"指出文档中对行业理解不准确的地方，并提供改进建议。" +
			"请用专业、严谨但平易近人的语气回复。",
	},
}

// AIService 封装 DeepSeek API 的调用逻辑，支持流式和非流式两种模式。
// 调用时动态传入 API Key、URL 和模型名，不再全局绑定。
type AIService struct{}

// NewAIService 创建 AIService 实例（无状态，无需配置）。
func NewAIService() *AIService {
	return &AIService{}
}

// deepseekRequest 是 DeepSeek API 的请求体结构。
type deepseekRequest struct {
	Model       string        `json:"model"`
	Messages    []DeepseekMsg `json:"messages"`
	Stream      bool          `json:"stream"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

// DeepseekMsg 表示一条对话消息，包含角色（system/user/assistant）和内容。
type DeepseekMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// deepseekStreamChunk 是 DeepSeek 流式响应中的单个数据块。
type deepseekStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// ChatStream 调用 DeepSeek 流式 API，将结果通过回调函数逐块返回。
// 参数均动态传入，支持每个用户使用自己的 Key、URL 和模型。
// apiKey: DeepSeek API 密钥
// apiURL: API 地址，如 https://api.deepseek.com/chat/completions
// model: 模型名，如 deepseek-v4-flash / deepseek-v4-pro
// messages: 完整的消息列表（由前端维护）
// onChunk: 每次收到内容片段时调用
// 返回完整内容（用于记录历史）
// normalizeAPIURL 自动补全 API URL。
// DeepSeek 端点格式为 https://api.deepseek.com/chat/completions（无 /v1/ 前缀）。
func normalizeAPIURL(url string) string {
	if strings.HasSuffix(url, "/chat/completions") {
		return url
	}
	return strings.TrimRight(url, "/") + "/chat/completions"
}

func (s *AIService) ChatStream(apiKey, apiURL, model string, messages []DeepseekMsg, onChunk func(text string)) (string, error) {
	apiURL = normalizeAPIURL(apiURL)

	reqBody := deepseekRequest{
		Model:       model,
		Messages:    messages,
		Stream:      true,
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求体失败: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

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

// Chat 是非流式版本的 AI 对话接口，适用于不需要实时输出的场景。
func (s *AIService) Chat(apiKey, apiURL, model string, messages []DeepseekMsg) (string, error) {
	var fullContent string
	_, err := s.ChatStream(apiKey, apiURL, model, messages, func(text string) {
		fullContent += text
	})
	return fullContent, err
}
