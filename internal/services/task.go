package services

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/sunkaimr/taskcube/internal/models"
	"github.com/sunkaimr/taskcube/internal/pkg/common"
	"github.com/sunkaimr/taskcube/pkg/utils"
	"time"
)

type TaskService struct {
	// 归档任务元数据信息
	Model
	Name               string                `json:"name"`                 // 任务名称
	Description        string                `json:"description"`          // 说明
	Enable             bool                  `json:"enable"`               // 是否生效
	PolicyID           uint                  `json:"policy_id"`            // 策略ID
	ExecuteWindow      []string              `json:"execute_window"`       // 执行窗口
	ExecuteDate        string                `json:"execute_date"`         // 预计执行日期 "2024-01-02"
	Pause              bool                  `json:"pause"`                // 执行窗口外是否需要暂停执行
	RebuildFlag        bool                  `json:"rebuild_flag"`         // 执行窗口外是否重建表(仅在治理方式是删除时有效)。true:在执行窗口外仍然走重建流程; false:执行窗口外跳过重建流程
	TaskStatus         common.TaskStatusType `json:"task_status"`          // 任务状态
	TaskReason         string                `json:"task_reason"`          // 任务失败原因
	TaskDetail         string                `json:"task_detail"`          // 任务失败详情
	TaskResultQuantity int                   `json:"task_result_quantity"` // 治理数据量
	TaskResultSize     int                   `json:"task_result_size"`     // 治理数据大小
	TaskStartTime      string                `json:"task_start_time"`      // 开始执行时间
	TaskEndTime        string                `json:"task_end_time"`        // 执行结束时间
	TaskDuration       int                   `json:"task_duration"`        // 执行时长(秒)
	WorkFlow           string                `json:"workflow"`             // 工作流
	WorkFlowURL        string                `json:"workflow_url"`         // 工作流地址

	// 源端信息
	SrcID           uint   `json:"src_id"`            // 任务ID
	SrcName         string `json:"src_name"`          // 源端名称
	SrcBu           string `json:"src_bu"`            // 资产BU
	SrcClusterName  string `json:"src_cluster_name"`  // 集群名称
	SrcClusterID    string `json:"src_cluster_id"`    // 集群ID
	SrcAddr         string `json:"src_addr"`          // 源端地址
	SrcDatabaseName string `json:"src_database_name"` // 源库名
	SrcTablesName   string `json:"src_tables_name"`   // 源表名
	SrcColumns      string `json:"src_columns"`       // 源列名

	// 目标端信息
	DestID           uint               `json:"dest_id"`            // 目标端ID
	DestName         string             `json:"dest_name"`          // 目标端名称
	DestStorage      common.StorageType `json:"dest_storage"`       // 归档介质
	DestConnectionID uint               `json:"dest_connection_id"` // 归档库连接信息
	DestDatabaseName string             `json:"dest_database_name"` // 归档库名
	DestTableName    string             `json:"dest_table_name"`    // 归档表名
	DestCompress     bool               `json:"dest_compress"`      // 是否压缩存储

	// 数据治理方式
	Govern        common.GovernType        `json:"govern" `         // 数据治理方式 清空数据:truncate, 不备份清理:delete, 备份后清理:backup-delete, 归档:archive
	Condition     string                   `json:"condition"`       // 数据治理条件
	RetainSrcData bool                     `json:"retain_src_data"` //归档时否保留源表数据
	CleaningSpeed common.CleaningSpeedType `json:"cleaning_speed"`  // 清理速度 稳定优先:steady, 速度适中:balanced, 速度优先:swift

	// 结果通知
	Relevant     []string                `json:"relevant"`      // 关注人
	NotifyPolicy common.NotifyPolicyType `json:"notify_policy"` // 通知策略 不通知:silence, 成功时通知:success, 失败时通知:failed, 成功或失败都通知:always
}

func (c *TaskService) UpdateTask(ctx *gin.Context) (common.ServiceCode, error) {

	return common.CodeOK, nil
}

func (c *TaskService) QueryTask(ctx *gin.Context, queryMap map[string]string) (any, common.ServiceCode, error) {

	return nil, common.CodeOK, nil
}

func (c *TaskService) DeleteTask(ctx *gin.Context) (common.ServiceCode, error) {

	return common.CodeOK, nil
}

func UpdateTaskResult(ctx *gin.Context) (*TaskService, common.ServiceCode, error) {

	return nil, common.CodeOK, nil
}
func (c *TaskService) ServiceToModel() *models.Task {
	m := &models.Task{}
	m.ID = c.ID
	m.Creator = c.Creator
	m.Editor = c.Editor
	m.CreatedAt, _ = time.ParseInLocation(time.DateTime, c.CreatedAt, time.Now().Location())
	m.UpdatedAt, _ = time.ParseInLocation(time.DateTime, c.UpdatedAt, time.Now().Location())
	m.Name = c.Name
	m.Description = c.Description
	m.Enable = c.Enable
	m.PolicyID = c.PolicyID
	m.ExecuteWindow, _ = json.Marshal(c.ExecuteWindow)
	m.Pause = c.Pause
	m.RebuildFlag = c.RebuildFlag
	m.TaskResultQuantity = c.TaskResultQuantity
	m.TaskResultSize = c.TaskResultSize
	m.TaskDuration = c.TaskDuration
	m.WorkFlow = c.WorkFlow
	m.SrcID = c.SrcID
	m.SrcName = c.SrcName
	m.SrcBu = c.SrcBu
	m.SrcClusterName = c.SrcClusterName
	m.SrcClusterID = c.SrcClusterID
	m.SrcDatabaseName = c.SrcDatabaseName
	m.SrcTablesName = c.SrcTablesName
	m.SrcColumns = c.SrcColumns
	m.DestID = c.DestID
	m.DestName = c.DestName
	m.DestStorage = c.DestStorage
	m.DestConnectionID = c.DestConnectionID
	m.DestDatabaseName = c.DestDatabaseName
	m.DestTableName = c.DestTableName
	m.DestCompress = c.DestCompress
	m.Govern = c.Govern
	m.Condition = c.Condition
	m.RetainSrcData = c.RetainSrcData
	m.CleaningSpeed = c.CleaningSpeed
	m.NotifyPolicy = c.NotifyPolicy
	m.ExecuteDate = c.ExecuteDate
	m.TaskStartTime, _ = time.ParseInLocation(time.DateTime, c.TaskStartTime, time.Now().Location())
	m.TaskEndTime, _ = time.ParseInLocation(time.DateTime, c.TaskEndTime, time.Now().Location())
	m.Relevant, _ = json.Marshal(c.Relevant)
	return m
}

func (c *TaskService) ModelToService(m *models.Task) *TaskService {
	c.ID = m.ID
	c.Creator = m.Creator
	c.Editor = m.Editor
	c.CreatedAt = m.CreatedAt.Format(time.DateTime)
	c.UpdatedAt = m.UpdatedAt.Format(time.DateTime)
	c.Name = m.Name
	c.Description = m.Description
	c.ID = m.ID
	c.Name = m.Name
	c.Enable = m.Enable
	c.Pause = m.Pause
	c.RebuildFlag = m.RebuildFlag
	c.PolicyID = m.PolicyID
	c.ExecuteDate = m.ExecuteDate
	c.TaskStatus = m.TaskStatus
	c.TaskReason = m.TaskReason
	c.TaskDetail = m.TaskDetail
	c.TaskResultQuantity = m.TaskResultQuantity
	c.TaskResultSize = m.TaskResultSize
	c.TaskDuration = m.TaskDuration
	c.WorkFlow = m.WorkFlow
	c.SrcID = m.SrcID
	c.SrcName = m.SrcName
	c.SrcBu = m.SrcBu
	c.SrcClusterName = m.SrcClusterName
	c.SrcClusterID = m.SrcClusterID
	c.SrcDatabaseName = m.SrcDatabaseName
	c.SrcTablesName = m.SrcTablesName
	c.SrcColumns = m.SrcColumns
	c.DestID = m.DestID
	c.DestName = m.DestName
	c.DestStorage = m.DestStorage
	c.DestConnectionID = m.DestConnectionID
	c.DestDatabaseName = m.DestDatabaseName
	c.DestTableName = m.DestTableName
	c.DestCompress = m.DestCompress
	c.Govern = m.Govern
	c.Condition = m.Condition
	c.RetainSrcData = m.RetainSrcData
	c.CleaningSpeed = m.CleaningSpeed
	c.NotifyPolicy = m.NotifyPolicy
	_ = json.Unmarshal(m.Relevant, &c.Relevant)
	_ = json.Unmarshal(m.ExecuteWindow, &c.ExecuteWindow)
	c.TaskStartTime = utils.Ternary[string](m.TaskStartTime == time.UnixMilli(0), "", m.TaskStartTime.Format(time.DateTime))
	c.TaskEndTime = utils.Ternary[string](m.TaskEndTime == time.UnixMilli(0), "", m.TaskEndTime.Format(time.DateTime))
	return c
}
