package controllers

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/sunkaimr/taskcube/internal/pkg/common"
	"github.com/sunkaimr/taskcube/internal/services"
	"github.com/sunkaimr/taskcube/internal/services/types"
	"net/http"
)

type TaskController struct{}

// CreateTask			创建任务
// @Router				/task [post]
// @Description			创建任务
// @Tags				任务
// @Param				Task		body		services.TaskService	true	"TaskService"
// @Success				200			{object}	common.Response{data=services.TaskService}
// @Failure				500			{object}	common.Response
func (c *TaskController) CreateTask(ctx *gin.Context) {
	req := &services.TaskService{}
	if err := ctx.ShouldBindJSON(req); err != nil {
		log.Error(err)
		ctx.JSON(http.StatusBadRequest, common.Response{ServiceCode: common.CodeBindErr, Error: err.Error()})
		return
	}

	code, err := req.CreateTask(ctx)
	if err != nil {
		ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Error: err.Error()})
		return
	}
	ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Data: req})
	return
}

// UpdateTask			更新任务
// @Router				/task [put]
// @Description			更新任务
// @Tags				任务
// @Param				Task		body		services.TaskService	true	"TaskService"
// @Success				200			{object}	common.Response{data=services.TaskService}
// @Failure				500			{object}	common.Response
func (c *TaskController) UpdateTask(ctx *gin.Context) {
	req := &services.TaskService{}
	if err := ctx.ShouldBindJSON(req); err != nil {
		log.Error(err)
		ctx.JSON(http.StatusBadRequest, common.Response{ServiceCode: common.CodeBindErr, Error: err.Error()})
		return
	}

	code, err := req.UpdateTask(ctx)
	if err != nil {
		ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Error: err.Error()})
		return
	}
	ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Data: req})
	return
}

// QueryTask			查询任务
// @Router				/task [get]
// @Description			查询任务
// @Tags				任务
// @Param   			page				query		int			false  	"page"
// @Param   			pageSize			query		int     	false  	"pageSize"
// @Param   			name				query		string     	false  	"name"
// @Param   			version				query		string     	false  	"version"
// @Param   			pause				query		bool     	false  	"pause"
// @Param   			terminate			query		bool     	false  	"terminate"
// @Param   			status				query		string     	false  	"status"	Enums(Creating,Pending,Running,Pausing,Paused,Succeeded,Failed,Unknown,Terminating,Terminated)
// @Success				200					{object}	common.Response{data=[]services.TaskService}
// @Failure				500					{object}	common.Response
func (c *TaskController) QueryTask(ctx *gin.Context) {
	queryMap := make(map[string]string, 10)
	queryMap["name"] = ctx.Query("name")
	queryMap["version"] = ctx.Query("version")
	queryMap["status"] = ctx.Query("status")

	req := &services.TaskService{}
	data, code, err := req.QueryTask(ctx, queryMap)
	if err != nil {
		ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Error: err.Error()})
		return
	}
	ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Data: data})
	return
}

// DeleteTask			删除任务
// @Router				/task/{task} [delete]
// @Description			删除任务
// @Tags				任务
// @Param				task		path		string	true	"task"
// @Success				200			{object}	common.Response
// @Failure				500			{object}	common.Response
func (c *TaskController) DeleteTask(ctx *gin.Context) {
	req := &services.TaskService{}
	req.Kind = types.TaskKind
	req.Metadata.Name = ctx.Param("task")

	code, err := req.DeleteTask(ctx)
	if err != nil {
		ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Error: err.Error()})
		return
	}
	ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code})
	return
}

// PauseTask			暂停任务
// @Router				/task/{task}/pause [post]
// @Description			暂停任务
// @Tags				任务
// @Param				task		path		string	true	"task"
// @Success				200			{object}	common.Response
// @Failure				500			{object}	common.Response
func (c *TaskController) PauseTask(ctx *gin.Context) {
	req := &services.TaskService{}
	req.Kind = types.TaskKind
	req.Metadata.Name = ctx.Param("task")

	code, err := req.PauseTask(ctx)
	if err != nil {
		ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Error: err.Error()})
		return
	}
	ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Data: req})
	return
}

// UnpauseTask			恢复任务
// @Router				/task/{task}/unpause [post]
// @Description			恢复任务
// @Tags				任务
// @Param				task		path		string	true	"task"
// @Success				200			{object}	common.Response
// @Failure				500			{object}	common.Response
func (c *TaskController) UnpauseTask(ctx *gin.Context) {
	req := &services.TaskService{}
	req.Kind = types.TaskKind
	req.Metadata.Name = ctx.Param("task")

	code, err := req.UnpauseTask(ctx)
	if err != nil {
		ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Error: err.Error()})
		return
	}
	ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Data: req})
	return
}

// StopTask				停止任务
// @Router				/task/{task}/stop [post]
// @Description			停止任务
// @Tags				任务
// @Param				task		path		string	true	"task"
// @Success				200			{object}	common.Response
// @Failure				500			{object}	common.Response
func (c *TaskController) StopTask(ctx *gin.Context) {
	req := &services.TaskService{}
	req.Kind = types.TaskKind
	req.Metadata.Name = ctx.Param("task")

	code, err := req.StopTask(ctx)
	if err != nil {
		ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Error: err.Error()})
		return
	}
	ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Data: req})
	return
}

// TaskLogs				查看任务日志
// @Router				/task/{task}/step/{step}/logs [get]
// @Description			查看任务日志
// @Tags				任务
// @Param				task		path		string	true	"task"
// @Param				step		path		string	true	"step"
// @Success				200			{object}	common.Response
// @Failure				500			{object}	common.Response
func (c *TaskController) TaskLogs(ctx *gin.Context) {
	_, _ = common.ExtractContext(ctx)

	task, step := ctx.Param("task"), ctx.Param("step")

	req := &services.TaskService{}
	req.Kind = types.TaskKind
	req.Metadata.Name = task

	req.TaskLogs(ctx, step)

	//ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Data: req})
	return
}
