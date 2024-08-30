package router

import (
	"github.com/gin-gonic/gin"
	"github.com/sunkaimr/taskcube/configs"
	doc "github.com/sunkaimr/taskcube/docs"
	ctl "github.com/sunkaimr/taskcube/internal/controllers"
	"github.com/sunkaimr/taskcube/internal/middlewares"
	"github.com/sunkaimr/taskcube/internal/pkg/common"
	swgfiles "github.com/swaggo/files"
	ginswg "github.com/swaggo/gin-swagger"
	"github.com/swaggo/swag"
	"io"
	"net/http"
	"net/url"
)

var Router *gin.Engine

func Init(ctx *common.Context) *gin.Engine {
	gin.DefaultWriter = io.Discard
	Router = gin.Default()

	middlewares.LoadMiddlewares(Router)

	// 公共路由无需鉴权
	public := Router.Group("/taskcube/api/v1")
	{
		public.GET("/health", new(ctl.HealthController).Health)
	}

	// 脚本相关路由
	script := public.Group("/script")
	{
		// 创建任务模版
		script.POST("/", new(ctl.ScriptController).CreateScript)
		// 删除任务模版
		script.DELETE("/:script", new(ctl.ScriptController).DeleteScript)
		// 修改任务模版
		script.PUT("/", new(ctl.ScriptController).UpdateScript)
		// 查询任务模版
		script.GET("/", new(ctl.ScriptController).QueryScript)
	}

	// 任务模版相关路由
	tmpl := public.Group("/template")
	{
		// 创建任务模版
		tmpl.POST("/", new(ctl.TaskTemplateController).CreateTaskTemplate)
		// 删除任务模版
		tmpl.DELETE("/:template", new(ctl.TaskTemplateController).DeleteTaskTemplate)
		// 修改任务模版
		tmpl.PUT("/", new(ctl.TaskTemplateController).UpdateTaskTemplate)
		// 查询任务模版
		tmpl.GET("/", new(ctl.TaskTemplateController).QueryTaskTemplate)
		// 基于任务模版提交任务
		tmpl.POST("/:template/submit", new(ctl.TaskTemplateController).SubmitTaskTemplate)
	}

	// 任务相关路由
	task := public.Group("/task")
	{
		// 创建任务
		task.POST("/", new(ctl.TaskController).CreateTask)
		// 删除任务
		task.DELETE("/:task", new(ctl.TaskController).DeleteTask)
		// 修改任务
		task.PUT("/", new(ctl.TaskController).UpdateTask)
		// 查询任务
		task.GET("/", new(ctl.TaskController).QueryTask)
		// 暂停任务
		task.POST("/:task/pause", new(ctl.TaskController).PauseTask)
		// 终止任务
		task.POST("/:task/stop", new(ctl.TaskController).StopTask)
		// 恢复任务
		task.POST("/:task/unpause", new(ctl.TaskController).UnpauseTask)
		// 查看任务日志
		task.GET("/:task/logs", new(ctl.TaskController).TaskLogs)

		// 上报任务执行结果
		//task.PUT("/result", new(ctl.TaskController).UpdateTaskResult)
	}

	// init swagger
	if configs.C.Server.ExternalAddr != "" {
		sw := swag.GetSwagger(doc.SwaggerInfo.InfoInstanceName).(*swag.Spec)
		u, err := url.Parse(configs.C.Server.ExternalAddr)
		if err != nil {
			ctx.Log.Fatalf("config: server.externalAddr parse failed, %s", err)
		}
		sw.Schemes = []string{u.Scheme}
		sw.Host = u.Host
	}
	Router.GET("/swagger/*any", ginswg.WrapHandler(swgfiles.Handler))
	Router.NoRoute(func(ctx *gin.Context) {
		ctx.JSON(http.StatusNotFound, common.Response{ServiceCode: common.CodeNotFound})
	})

	return Router
}
