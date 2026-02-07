package feishu

import (
	"encoding/json"
	"time"
)

// =============================================================================
// 飞书API通用响应
// =============================================================================

// BaseResponse 飞书API通用响应结构
type BaseResponse struct {
	Code int    `json:"code"` // 错误码，0表示成功
	Msg  string `json:"msg"`  // 错误消息
}

// =============================================================================
// 审批相关模型
// =============================================================================

// 审批表单字段类型常量
const (
	FieldTypeText     = "input"        // 单行文本
	FieldTypeTextArea = "textarea"     // 多行文本
	FieldTypeNumber   = "number"       // 数字
	FieldTypeDate     = "date"         // 日期
	FieldTypeSelect   = "radioV2"      // 单选
	FieldTypeMulti    = "checkboxV2"   // 多选
	FieldTypeContact  = "contact"      // 人员选择
	FieldTypeAttach   = "attachmentV2" // 附件
)

// ApprovalFormField 审批表单字段定义
type ApprovalFormField struct {
	ID       string   `json:"id"`                 // 字段唯一标识
	Type     string   `json:"type"`               // 字段类型（参考上方常量）
	Name     string   `json:"name"`               // 字段显示名称
	Required bool     `json:"required,omitempty"`  // 是否必填
	Options  []string `json:"options,omitempty"`   // 选项列表（单选/多选时使用）
	Placeholder string `json:"placeholder,omitempty"` // 占位提示文字
}

// ApprovalDefinition 审批定义
type ApprovalDefinition struct {
	Name        string              `json:"name"`         // 审批名称
	Description string              `json:"description"`  // 审批描述
	FormFields  []ApprovalFormField `json:"form_fields"`  // 表单字段列表
	NodeList    []ApprovalNode      `json:"node_list"`    // 审批节点列表
}

// ApprovalNode 审批节点
type ApprovalNode struct {
	ID          string   `json:"id"`                    // 节点ID
	Name        string   `json:"name,omitempty"`        // 节点名称
	Type        string   `json:"type"`                  // 节点类型：AND(会签), OR(或签)
	ApproverIDs []string `json:"approver_ids,omitempty"` // 审批人OpenID列表
}

// CreateApprovalInstanceReq 创建审批实例请求
type CreateApprovalInstanceReq struct {
	ApprovalCode     string                `json:"approval_code"`               // 审批定义code
	OpenID           string                `json:"open_id"`                     // 发起人OpenID
	FormData         string                `json:"form"`                        // 表单数据JSON字符串
	NodeApproverList []NodeApproverItem    `json:"node_approver_open_id_list,omitempty"` // 指定节点审批人
}

// NodeApproverItem 节点审批人配置
type NodeApproverItem struct {
	Key   string   `json:"key"`   // 节点ID
	Value []string `json:"value"` // 审批人OpenID列表
}

// ApprovalInstance 审批实例
type ApprovalInstance struct {
	InstanceCode string              `json:"instance_code"` // 实例code
	ApprovalCode string              `json:"approval_code"` // 审批定义code
	Status       string              `json:"status"`        // 状态：PENDING/APPROVED/REJECTED/CANCELED/DELETED
	OpenID       string              `json:"open_id"`       // 发起人OpenID
	FormData     string              `json:"form"`          // 表单数据
	Timeline     []ApprovalTimeline  `json:"timeline"`      // 审批时间线
	StartTime    string              `json:"start_time"`    // 发起时间（毫秒时间戳）
	EndTime      string              `json:"end_time"`      // 结束时间（毫秒时间戳）
}

// ApprovalTimeline 审批时间线记录
type ApprovalTimeline struct {
	Type       string `json:"type"`        // 类型：START/PASS/REJECT/CANCEL等
	OpenID     string `json:"open_id"`     // 操作人OpenID
	Comment    string `json:"comment"`     // 审批意见
	CreateTime string `json:"create_time"` // 操作时间（毫秒时间戳）
}

// =============================================================================
// 审批API响应结构
// =============================================================================

// CreateApprovalDefResponse 创建审批定义响应
type CreateApprovalDefResponse struct {
	BaseResponse
	Data struct {
		ApprovalCode string `json:"approval_code"` // 审批定义code
	} `json:"data"`
}

// CreateApprovalInstanceResponse 创建审批实例响应
type CreateApprovalInstanceResponse struct {
	BaseResponse
	Data struct {
		InstanceCode string `json:"instance_code"` // 实例code
	} `json:"data"`
}

// GetApprovalInstanceResponse 获取审批实例响应
type GetApprovalInstanceResponse struct {
	BaseResponse
	Data ApprovalInstance `json:"data"`
}

// =============================================================================
// 任务相关模型
// =============================================================================

// TaskMember 任务成员
type TaskMember struct {
	ID   string `json:"id"`   // 成员OpenID
	Role string `json:"role"` // 角色：assignee(执行人), follower(关注人)
}

// TaskDue 任务截止时间
type TaskDue struct {
	Time     int64 `json:"time"`       // 截止时间（毫秒时间戳）
	IsAllDay bool  `json:"is_all_day"` // 是否全天任务
}

// CreateTaskReq 创建任务请求
type CreateTaskReq struct {
	Summary     string       `json:"summary"`               // 任务标题
	Description string       `json:"description,omitempty"` // 任务描述
	Members     []TaskMember `json:"members,omitempty"`     // 任务成员
	Due         *TaskDue     `json:"due,omitempty"`         // 截止时间
	Origin      *TaskOrigin  `json:"origin,omitempty"`      // 任务来源（关联文档等）
}

// TaskOrigin 任务来源信息
type TaskOrigin struct {
	PlatformI18nName string     `json:"platform_i18n_name"` // 来源平台名称
	Href             *TaskHref  `json:"href,omitempty"`     // 关联链接
}

// TaskHref 任务关联链接
type TaskHref struct {
	URL   string `json:"url"`   // 链接URL
	Title string `json:"title"` // 链接标题
}

// UpdateTaskReq 更新任务请求
type UpdateTaskReq struct {
	Summary     *string  `json:"summary,omitempty"`     // 任务标题
	Description *string  `json:"description,omitempty"` // 任务描述
	Due         *TaskDue `json:"due,omitempty"`         // 截止时间
}

// CreateTaskResponse 创建任务响应
type CreateTaskResponse struct {
	BaseResponse
	Data struct {
		Task struct {
			Guid string `json:"guid"` // 任务全局唯一ID
		} `json:"task"`
	} `json:"data"`
}

// UpdateTaskResponse 更新任务响应
type UpdateTaskResponse struct {
	BaseResponse
}

// CompleteTaskResponse 完成任务响应
type CompleteTaskResponse struct {
	BaseResponse
}

// =============================================================================
// 会议（日历事件）相关模型
// =============================================================================

// EventTime 事件时间
type EventTime struct {
	Timestamp string `json:"timestamp"` // Unix时间戳（秒）
}

// EventAttendee 事件参会人
type EventAttendee struct {
	Type   string `json:"type"`    // 类型：user
	UserID string `json:"user_id"` // 用户ID
}

// CreateMeetingReq 创建会议请求
type CreateMeetingReq struct {
	Summary          string    `json:"summary"`           // 会议标题
	Description      string    `json:"description"`       // 会议描述（可含文档链接）
	StartTime        time.Time `json:"-"`                 // 开始时间
	EndTime          time.Time `json:"-"`                 // 结束时间
	AttendeeIDs      []string  `json:"-"`                 // 参会人UserID列表
	NeedNotification bool      `json:"need_notification"` // 是否发送通知
}

// CreateMeetingResponse 创建会议响应
type CreateMeetingResponse struct {
	BaseResponse
	Data struct {
		Event struct {
			EventID string `json:"event_id"` // 日历事件ID
		} `json:"event"`
	} `json:"data"`
}

// =============================================================================
// 消息卡片相关模型
// =============================================================================

// InteractiveCard 飞书交互式消息卡片
type InteractiveCard struct {
	Config   *CardConfig   `json:"config,omitempty"`   // 卡片配置
	Header   *CardHeader   `json:"header,omitempty"`   // 卡片标题
	Elements []CardElement `json:"elements,omitempty"` // 卡片内容元素
}

// CardConfig 卡片配置
type CardConfig struct {
	WideScreenMode bool `json:"wide_screen_mode"` // 宽屏模式
}

// CardHeader 卡片标题
type CardHeader struct {
	Title    CardText `json:"title"`              // 标题文本
	Template string   `json:"template,omitempty"` // 标题颜色模板：blue/green/red/orange/purple等
}

// CardText 卡片文本
type CardText struct {
	Tag     string `json:"tag"`     // 文本类型：plain_text / lark_md
	Content string `json:"content"` // 文本内容
}

// CardElement 卡片元素（通用）
type CardElement struct {
	Tag      string        `json:"tag"`                 // 元素类型：div/hr/action/note/markdown等
	Text     *CardText     `json:"text,omitempty"`      // 文本内容
	Fields   []CardField   `json:"fields,omitempty"`    // 字段列表（div元素使用）
	Actions  []CardAction  `json:"actions,omitempty"`   // 操作按钮列表（action元素使用）
	Elements []CardElement `json:"elements,omitempty"`  // 子元素（note元素使用）
	Content  string        `json:"content,omitempty"`   // Markdown内容（markdown元素使用）
}

// CardField 卡片字段
type CardField struct {
	IsShort bool     `json:"is_short"` // 是否短字段（并排显示）
	Text    CardText `json:"text"`     // 字段文本
}

// CardAction 卡片操作按钮
type CardAction struct {
	Tag   string            `json:"tag"`             // 按钮类型：button
	Text  CardText          `json:"text"`            // 按钮文本
	Type  string            `json:"type,omitempty"`  // 按钮样式：primary/danger/default
	URL   string            `json:"url,omitempty"`   // 跳转链接
	Value map[string]string `json:"value,omitempty"` // 回调数据
}

// SendMessageRequest 发送消息请求（内部使用）
type SendMessageRequest struct {
	ReceiveIDType string `json:"receive_id_type"` // 接收者类型：chat_id/user_id
	ReceiveID     string `json:"receive_id"`      // 接收者ID
	MsgType       string `json:"msg_type"`        // 消息类型：interactive
	Content       string `json:"content"`         // 卡片JSON字符串
}

// SendMessageResponse 发送消息响应
type SendMessageResponse struct {
	BaseResponse
	Data struct {
		MessageID string `json:"message_id"` // 消息ID
	} `json:"data"`
}

// =============================================================================
// Webhook事件相关模型
// =============================================================================

// 审批事件状态常量
const (
	ApprovalStatusPending  = "PENDING"  // 审批中
	ApprovalStatusApproved = "APPROVED" // 已通过
	ApprovalStatusRejected = "REJECTED" // 已拒绝
	ApprovalStatusCanceled = "CANCELED" // 已撤回
	ApprovalStatusDeleted  = "DELETED"  // 已删除
)

// 事件类型常量
const (
	EventTypeApprovalInstance = "approval_instance"  // 审批实例事件
	EventTypeURLVerification  = "url_verification"   // URL验证事件
)

// WebhookEvent 飞书Webhook事件（通用信封）
type WebhookEvent struct {
	Schema string          `json:"schema"`          // 事件模式（2.0）
	Header *WebhookHeader  `json:"header"`          // 事件头
	Event  json.RawMessage `json:"event"`           // 事件体（根据类型解析）
	// v1 兼容字段
	Type      string `json:"type,omitempty"`      // v1事件类型
	Challenge string `json:"challenge,omitempty"` // URL验证挑战码
	Token     string `json:"token,omitempty"`     // 验证token
}

// WebhookHeader 事件头信息
type WebhookHeader struct {
	EventID    string `json:"event_id"`    // 事件ID
	EventType  string `json:"event_type"`  // 事件类型
	CreateTime string `json:"create_time"` // 事件创建时间
	Token      string `json:"token"`       // 验证token
	AppID      string `json:"app_id"`      // 应用ID
}

// ApprovalEvent 审批实例事件
type ApprovalEvent struct {
	ApprovalCode string `json:"approval_code"` // 审批定义code
	InstanceCode string `json:"instance_code"` // 审批实例code
	Status       string `json:"status"`        // 审批状态：APPROVED/REJECTED/CANCELED
	OperateTime  string `json:"operate_time"`  // 操作时间
	FormData     string `json:"form"`          // 表单数据
	// 从v2事件中解析
	OpenID       string `json:"open_id,omitempty"` // 操作人
}

// ApprovalInstanceEvent v2审批实例状态变更事件体
type ApprovalInstanceEvent struct {
	ApprovalCode string `json:"approval_code"`
	InstanceCode string `json:"instance_code"`
	Status       string `json:"status"`
	OpenID       string `json:"open_id"`
	OperatorOpenID string `json:"operator_open_id"`
}

// URLVerificationEvent URL验证事件
type URLVerificationEvent struct {
	Type      string `json:"type"`      // 固定为 url_verification
	Challenge string `json:"challenge"` // 挑战码，需原样返回
	Token     string `json:"token"`     // 验证token
}
