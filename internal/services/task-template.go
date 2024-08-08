package services

import (
	"github.com/gin-gonic/gin"
	"github.com/sunkaimr/taskcube/internal/pkg/common"
)

type TaskTemplateService struct {
	// 归档任务元数据信息

}

func (c *TaskTemplateService) UpdateTask(ctx *gin.Context) (common.ServiceCode, error) {

	return common.CodeOK, nil
}

func (c *TaskTemplateService) QueryTask(ctx *gin.Context, queryMap map[string]string) (any, common.ServiceCode, error) {

	return nil, common.CodeOK, nil
}

func (c *TaskTemplateService) DeleteTask(ctx *gin.Context) (common.ServiceCode, error) {

	return common.CodeOK, nil
}
