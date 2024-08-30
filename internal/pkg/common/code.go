package common

// 通用错误码
var (
	CodeOK           = ServiceCode{0, ""}
	CodeBindErr      = ServiceCode{4000000, "参数解析错误"}
	CodeInvalidID    = ServiceCode{4000001, "缺少ID或ID无效"}
	CodeMissAuth     = ServiceCode{4010001, "No Authorization"}
	CodeTokenErr     = ServiceCode{4010002, "token格式错误"}
	CodeTokenExpired = ServiceCode{4010003, "token已过期"}
	CodeDenied       = ServiceCode{4030000, "权限不足"}
	CodeAdminOnly    = ServiceCode{4030001, "权限不足，需要管理员权限"}
	CodeNotFound     = ServiceCode{4040000, "not found"}
	CodeServerErr    = ServiceCode{5000000, "服务器内部错误"}
)

// Script模块错误码范围: 4xx1xx - 5xx1xx
var (
	CodeScriptKindErr   = ServiceCode{4000101, "Kind必须是Script"}
	CodeScriptTypeErr   = ServiceCode{4000102, "不支持Metadata.Type"}
	CodeScriptNameEmpty = ServiceCode{4000103, "Script名字不能为空"}
	CodeScriptNotExist  = ServiceCode{4040101, "Script不存在"}
	CodeScriptExisted   = ServiceCode{4090101, "存在同名的Script"}
)

// TaskTemplate模块错误码范围: 4xx2xx - 5xx2xx
var (
	CodeTaskTemplateKindErr           = ServiceCode{4000201, "Kind必须是TaskTemplate"}
	CodeTaskTemplateNameEmpty         = ServiceCode{4000203, "TaskTemplate名字不能为空"}
	CodeTaskTemplateStepNameEmpty     = ServiceCode{4000203, "step.name不能为空"}
	CodeTaskTemplateStepImageEmpty    = ServiceCode{4000203, "step.image不能为空"}
	CodeTaskTemplateScriptSourceEmpty = ServiceCode{4000203, "step.script和step.source不能同时为空"}
	CodeTaskTemplateNotExist          = ServiceCode{4040201, "TaskTemplate不存在"}
	CodeTaskTemplateExisted           = ServiceCode{4090201, "存在同名的TaskTemplate"}
)

// Task模块错误码范围: 4xx3xx - 5xx3xx
var (
	CodeTaskKindErr   = ServiceCode{4000301, "Kind必须是TaskTemplate"}
	CodeTaskNameEmpty = ServiceCode{4000302, "Task名字不能为空"}
	CodeTaskImmutable = ServiceCode{4000303, "当前状态的任务不支持该操作"}
	CodeTaskNotExist  = ServiceCode{4040301, "Task不存在"}
	CodeTaskExisted   = ServiceCode{4090301, "存在同名的Task"}
)
