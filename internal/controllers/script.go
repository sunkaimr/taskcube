package controllers

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/sunkaimr/taskcube/internal/pkg/common"
	"github.com/sunkaimr/taskcube/internal/services"
	"github.com/sunkaimr/taskcube/internal/services/types"
	"net/http"
)

type ScriptController struct{}

// CreateScript			创建执行脚本
// @Router				/script [post]
// @Description			创建任务
// @Tags				执行脚本
// @Param				Script		body		services.ScriptService	true	"ScriptService"
// @Success				200			{object}	common.Response{data=services.ScriptService}
// @Failure				500			{object}	common.Response
func (c *ScriptController) CreateScript(ctx *gin.Context) {
	req := &services.ScriptService{}
	if err := ctx.ShouldBindJSON(req); err != nil {
		log.Error(err)
		ctx.JSON(http.StatusBadRequest, common.Response{ServiceCode: common.CodeBindErr, Error: err.Error()})
		return
	}

	code, err := req.CreateScript(ctx)
	if err != nil {
		ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Error: err.Error()})
		return
	}
	ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Data: req})
	return
}

// UpdateScript			更新执行脚本
// @Router				/script [put]
// @Description			更新执行脚本
// @Tags				执行脚本
// @Param				Script		body		services.ScriptService	true	"ScriptService"
// @Success				200			{object}	common.Response{data=services.ScriptService}
// @Failure				500			{object}	common.Response
func (c *ScriptController) UpdateScript(ctx *gin.Context) {
	req := &services.ScriptService{}
	if err := ctx.ShouldBindJSON(req); err != nil {
		log.Error(err)
		ctx.JSON(http.StatusBadRequest, common.Response{ServiceCode: common.CodeBindErr, Error: err.Error()})
		return
	}

	code, err := req.UpdateScript(ctx)
	if err != nil {
		ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Error: err.Error()})
		return
	}
	ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Data: req})
	return
}

// QueryScript			查询执行脚本
// @Router				/script [get]
// @Description			查询执行脚本
// @Tags				执行脚本
// @Param   			page				query		int			false  	"page"
// @Param   			pageSize			query		int     	false  	"pageSize"
// @Param   			name				query		string     	false  	"name"
// @Param   			version				query		string     	false  	"version"
// @Param   			type				query		string     	false  	"Script type" Enums(bash, sh, python)
// @Success				200					{object}	common.Response{data=[]services.ScriptService}
// @Failure				500					{object}	common.Response
func (c *ScriptController) QueryScript(ctx *gin.Context) {
	queryMap := make(map[string]string, 10)
	queryMap["name"] = ctx.Query("name")
	queryMap["version"] = ctx.Query("version")
	queryMap["type"] = ctx.Query("type")

	task := &services.ScriptService{}
	data, code, err := task.QueryScript(ctx, queryMap)
	if err != nil {
		ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Error: err.Error()})
		return
	}
	ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Data: data})
	return
}

// DeleteScript			删除执行脚本
// @Router				/script/{script} [delete]
// @Description			删除执行脚本
// @Tags				执行脚本
// @Param				script		path		string	true	"script"
// @Param   			version		query		string  false  	"version"
// @Success				200			{object}	common.Response
// @Failure				500			{object}	common.Response
func (c *ScriptController) DeleteScript(ctx *gin.Context) {
	req := &services.ScriptService{}
	req.Kind = types.ScriptKind
	req.Metadata.Name = ctx.Param("script")
	req.Metadata.Version = ctx.Query("version")

	code, err := req.DeleteScript(ctx)
	if err != nil {
		ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code, Error: err.Error()})
		return
	}
	ctx.JSON(common.ServiceCode2HttpCode(code), common.Response{ServiceCode: code})
	return
}
