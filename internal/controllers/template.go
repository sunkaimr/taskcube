package controllers

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/sunkaimr/taskcube/internal/pkg/common"
	"github.com/sunkaimr/taskcube/internal/services"
	"github.com/sunkaimr/taskcube/internal/services/types"
	"net/http"
)

type TaskTemplateController struct{}

// CreateTaskTemplate	创建任务模版
// @Router				/template [post]
// @Description			创建任务模版
// @Tags				任务模版
// @Param				TaskTemplate	body		services.TaskTemplateService	true	"TaskTemplateService"
// @Success				200				{object}	common.Response{data=services.TaskTemplateService}
// @Failure				500				{object}	common.Response
func (c *TaskTemplateController) CreateTaskTemplate(ctx *gin.Context) {
	req := &services.TaskTemplateService{}
	if err := ctx.ShouldBindJSON(req); err != nil {
		log.Error(err)
		ctx.JSON(http.StatusBadRequest, common.Response{ServiceCode: common.CodeBindErr, Error: err.Error()})
		return
	}

	code, err := req.CreateTaskTemplate(ctx)
	if err != nil {
		ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Error: err.Error()})
		return
	}
	ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Data: req})
	return
}

// UpdateTaskTemplate	更新任务模版
// @Router				/template [put]
// @Description			更新任务模版
// @Tags				任务模版
// @Param				TaskTemplate	body		services.TaskTemplateService	true	"TaskTemplateService"
// @Success				200				{object}	common.Response{data=services.TaskTemplateService}
// @Failure				500				{object}	common.Response
func (c *TaskTemplateController) UpdateTaskTemplate(ctx *gin.Context) {
	req := &services.TaskTemplateService{}
	if err := ctx.ShouldBindJSON(req); err != nil {
		log.Error(err)
		ctx.JSON(http.StatusBadRequest, common.Response{ServiceCode: common.CodeBindErr, Error: err.Error()})
		return
	}

	code, err := req.UpdateTaskTemplate(ctx)
	if err != nil {
		ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Error: err.Error()})
		return
	}
	ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Data: req})
	return
}

// QueryTaskTemplate	查询任务模版
// @Router				/template [get]
// @Description			查询任务模版
// @Tags				任务模版
// @Param   			page				query		int			false  	"page"
// @Param   			pageSize			query		int     	false  	"pageSize"
// @Param   			name				query		string     	false  	"name"
// @Param   			version				query		string     	false  	"version"
// @Success				200					{object}	common.Response{data=[]services.TaskTemplateService}
// @Failure				500					{object}	common.Response
func (c *TaskTemplateController) QueryTaskTemplate(ctx *gin.Context) {
	queryMap := make(map[string]string, 10)
	queryMap["name"] = ctx.Query("name")
	queryMap["version"] = ctx.Query("version")

	req := &services.TaskTemplateService{}
	data, code, err := req.QueryTaskTemplate(ctx, queryMap)
	if err != nil {
		ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Error: err.Error()})
		return
	}
	ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Data: data})
	return
}

// DeleteTaskTemplate	删除任务模版
// @Router				/template/{template} [delete]
// @Description			删除任务模版
// @Tags				任务模版
// @Param				template	path		string	true	"template"
// @Param   			version		query		string  false  	"version"
// @Success				200			{object}	common.Response
// @Failure				500			{object}	common.Response
func (c *TaskTemplateController) DeleteTaskTemplate(ctx *gin.Context) {
	req := &services.TaskTemplateService{}
	req.Kind = types.TaskTemplateKind
	req.Metadata.Name = ctx.Param("template")
	req.Metadata.Version = ctx.Query("version")

	code, err := req.DeleteTaskTemplate(ctx)
	if err != nil {
		ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Error: err.Error()})
		return
	}
	ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code})
	return
}

// SubmitTaskTemplate	提交任务模版
// @Router				/template/{template}/submit [post]
// @Description			提交任务模版
// @Tags				任务模版
// @Param				template	path		string	true	"template"
// @Success				200			{object}	common.Response
// @Failure				500			{object}	common.Response
func (c *TaskTemplateController) SubmitTaskTemplate(ctx *gin.Context) {
	req := &services.TaskTemplateService{}
	req.Kind = types.TaskTemplateKind
	req.Metadata.Name = ctx.Param("template")

	res, code, err := req.SubmitTaskTemplate(ctx)
	if err != nil {
		ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Error: err.Error()})
		return
	}
	ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Data: res})
}
