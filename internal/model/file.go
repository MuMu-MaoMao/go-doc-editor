// Package model 定义前后端交互的数据契约（DTO）。
// 包含统一的 API 响应结构和请求结构体，所有 handler 层使用这些结构体编码响应。
package model

// Response 是统一 API 响应格式，所有接口返回此结构。
type Response struct {
	Success  bool        `json:"success"`            // 操作是否成功
	Message  string      `json:"message,omitempty"`  // 成功时的提示消息
	Error    string      `json:"error,omitempty"`    // 失败时的错误描述
	Files    []string    `json:"files,omitempty"`    // 文件列表（ListFiles 接口使用）
	Content  string      `json:"content,omitempty"`  // 文件内容（ReadFile 接口使用）
	Category *Category   `json:"category,omitempty"` // 单个分类信息
	Tree     []*Category `json:"tree,omitempty"`     // 分类树（GetCategoryTree 使用）
}

// SaveFileRequest 是保存文件接口的请求体。
type SaveFileRequest struct {
	Content string `json:"content"` // 文件内容
}

// Category 表示一个分类节点，用于前后端数据传输。
type Category struct {
	ID        int64       `json:"id"`                  // 分类ID
	Name      string      `json:"name"`                // 分类名称
	ParentID  *int64      `json:"parentId"`            // 父分类ID（null=顶级大类）
	SortOrder int         `json:"sortOrder"`            // 同级排序序号
	Children  []*Category `json:"children,omitempty"`  // 子分类列表（树形展示用）
}

// CreateCategoryRequest 是创建分类的请求体。
type CreateCategoryRequest struct {
	Name     string `json:"name"`                 // 分类名称
	ParentID *int64 `json:"parentId"`             // 父分类ID（null=创建大类）
}

// UpdateCategoryRequest 是更新分类名称的请求体。
type UpdateCategoryRequest struct {
	Name string `json:"name"` // 新分类名称
}

// SetCategoryRequest 是设置文档所属分类的请求体。
type SetCategoryRequest struct {
	CategoryID *int64 `json:"categoryId"` // 分类ID（null=取消分类）
}

// Annotation 表示一个关键语句标注。
type Annotation struct {
	ID             int64  `json:"id"`
	SourceFilename string `json:"sourceFilename"`
	SelectedText   string `json:"selectedText"`
	TargetFilename string `json:"targetFilename,omitempty"`
	Comment        string `json:"comment,omitempty"`
	PositionStart  int    `json:"positionStart"`
	PositionEnd    int    `json:"positionEnd"`
	CreatedAt      string `json:"createdAt,omitempty"`
}

// CreateAnnotationRequest 是创建标注的请求体。
type CreateAnnotationRequest struct {
	SourceFilename string `json:"sourceFilename"`
	SelectedText   string `json:"selectedText"`
	TargetFilename string `json:"targetFilename,omitempty"`
	Comment        string `json:"comment,omitempty"`
	PositionStart  int    `json:"positionStart"`
	PositionEnd    int    `json:"positionEnd"`
}

// FileInfo 包含文档的完整信息，用于列表展示分类信息。
type FileInfo struct {
	Filename   string `json:"filename"`
	CategoryID *int64 `json:"categoryId,omitempty"`
}
