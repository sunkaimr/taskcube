package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/sunkaimr/taskcube/internal/pkg/common"
)

type TaskController struct{}

// CreateTask			更新任务
// @Router				/task [put]
// @Description			更新任务
// @Tags				任务
// @Param				Task		body		services.TaskService	true	"Task"
// @Success				200			{object}	common.Response{data=services.TaskService}
// @Failure				500			{object}	common.Response
func (c *TaskController) CreateTask(ctx *gin.Context) {
	_, _ = common.ExtractContext(ctx)

	//ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Data: req})
	return
}

// UpdateTask			更新任务
// @Router				/task [put]
// @Description			更新任务
// @Tags				任务
// @Param				Task		body		services.TaskService	true	"Task"
// @Success				200			{object}	common.Response{data=services.TaskService}
// @Failure				500			{object}	common.Response
func (c *TaskController) UpdateTask(ctx *gin.Context) {
	_, _ = common.ExtractContext(ctx)

	//ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Data: req})
	return
}

// QueryTask			查询任务
// @Router				/task [get]
// @Description			查询任务
// @Tags				任务
// @Param   			page				query		int			false  	"page"
// @Param   			pageSize			query		int     	false  	"pageSize"
// @Param   			id					query		uint     	false  	"任务ID"
// @Param   			creator				query		string     	false  	"创建人"
// @Param   			editor				query		string     	false  	"修改人"
// @Param   			name				query		string     	false  	"任务名称"
// @Param   			description			query		string     	false  	"描述"
// @Param   			enable				query		bool     	false  	"是否生效"
// @Param   			policy_id			query		int     	false  	"策略ID"
// @Param   			execute_date		query		string     	false  	"计划执行日期: 2024-03-01"
// @Param   			pause				query		bool     	false  	"执行窗口外是否需要暂停执行"
// @Param   			rebuild_flag		query		bool     	false  	"执行窗口外是否重建表(仅在治理方式是删除时有效)。true:在执行窗口外仍然走重建流程; false:执行窗口外跳过重建流程"
// @Param   			task_status			query		int     	false  	"任务状态"
// @Param   			task_reason			query		string     	false  	"任务失败原因"
// @Param   			task_detail			query		string     	false  	"任务失败详情"
// @Param   			workflow			query		string     	false  	"工作流"
// @Param   			src_id				query		uint     	false  	"源端ID"
// @Param   			src_name			query		string     	false  	"源端名称"
// @Param   			src_bu				query		string     	false  	"资产BU"
// @Param   			src_cluster_name	query		string     	false  	"源端集群名称"
// @Param   			src_cluster_id		query		string     	false  	"源端集群ID"
// @Param   			src_database_name	query		string     	false  	"源库名"
// @Param   			src_tables_name		query		string     	false  	"源表名"
// @Param   			src_columns			query		string     	false  	"源端归档列名"
// @Param   			govern				query		string     	false  	"数据治理方式"		Enums(truncate,delete,backup-delete,archive)
// @Param   			condition			query		string     	false  	"数据治理条件"
// @Param   			clean_src			query		bool     	false  	"是否清理源表"
// @Param   			cleaning_speed		query		string     	false  	"清理速度"			Enums(steady,balanced,swift)
// @Param   			dest_id				query		uint     	false  	"目标端ID"
// @Param   			dest_name			query		string     	false  	"目标端名称"
// @Param   			dest_storage		query		string     	false  	"存储介质"			Enums(mysql, databend)
// @Param   			dest_connection_id	query		uint     	false  	"目标端连接ID"
// @Param   			dest_database_name	query		string     	false  	"目标端数据库"
// @Param   			dest_table_name		query		string     	false  	"目标端表名字"
// @Param   			dest_compress		query		bool     	false  	"目标端是否压缩存储存储"
// @Param   			relevant			query		string     	false  	"关注人"
// @Param   			notify_policy		query		string     	false  	"通知策略"			Enums(silence,success,failed,always)
// @Success				200		{object}	common.Response{data=services.TaskService}
// @Failure				500		{object}	common.Response
func (c *TaskController) QueryTask(ctx *gin.Context) {

	//ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Data: data})
	return
}

// DeleteTask			删除任务
// @Router				/task [delete]
// @Description			删除任务
// @Tags				任务
// @Param				Task		body		services.TaskService	true	"Task"
// @Success				200			{object}	common.Response{data=services.TaskService}
// @Failure				500			{object}	common.Response
func (c *TaskController) DeleteTask(ctx *gin.Context) {
	_, _ = common.ExtractContext(ctx)

	//ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code})
	return
}

// UpdateTaskResult		上报任务执行结果
// @Router				/task/result [put]
// @Description			上报任务执行结果
// @Tags				任务
// @Param				TaskResult	body		types.TaskResultService	true	"TaskResult"
// @Success				200			{object}	common.Response{data=services.TaskService}
// @Failure				500			{object}	common.Response
func (c *TaskController) UpdateTaskResult(ctx *gin.Context) {
	_, _ = common.ExtractContext(ctx)

	//ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Data: res})
	return
}
