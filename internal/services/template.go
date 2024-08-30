package services

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sunkaimr/taskcube/internal/pkg/common"
	. "github.com/sunkaimr/taskcube/internal/services/types"
	"github.com/sunkaimr/taskcube/pkg/utils"
	"gorm.io/gorm"
	"sort"
	"time"
)

type TaskTemplateService struct {
	TaskTemplate
}

func (c *TaskTemplateService) CheckParameters(ctx *gin.Context) (bool, common.ServiceCode, error) {
	_, db := common.ExtractContext(ctx)

	// 类型校验
	if c.Kind != TaskTemplateKind {
		return false, common.CodeTaskTemplateKindErr, fmt.Errorf("unsupport kind: %s, only support: %s", c.Kind, TaskTemplateKind)
	}

	// Name校验（不允许重名）
	f := TaskTemplate{
		Kind:     TaskTemplateKind,
		Metadata: TaskMetadata{Name: c.Metadata.Name},
	}
	exist, err := f.Exist(db)
	if err != nil {
		return false, common.CodeServerErr, err
	}

	if exist {
		return false, common.CodeTaskTemplateExisted, fmt.Errorf("%s/%s existed", c.Kind, c.Metadata.Name)
	}

	// 校验script必须存在
	for i, step := range c.Spec.Steps {
		code, err := checkTaskStep(db, &step)
		if err != nil {
			return false, code, fmt.Errorf("spec.steps[%d] check failed, %s", i, err)
		}
	}

	return true, common.CodeOK, nil
}

func checkTaskStep(db *gorm.DB, step *TaskSpecStep) (common.ServiceCode, error) {
	if step.Name == "" {
		return common.CodeTaskTemplateStepNameEmpty, fmt.Errorf("step.name can not be empty")
	}

	if step.Image == "" {
		return common.CodeTaskTemplateStepImageEmpty, fmt.Errorf("step.image can not be empty")
	}

	if step.Script == "" && step.Source == "" {
		return common.CodeTaskTemplateScriptSourceEmpty, fmt.Errorf("step.script and step.source can not both empty")
	} else if step.Script != "" && step.Source != "" {
		// Script和Source同时存在以Source为主
		step.Script = ""
	} else if step.Script != "" {
		// 校验Script必须存在
		f := Script{
			Kind:     ScriptKind,
			Metadata: ScriptMetadata{Name: step.Script},
		}
		exist, err := f.Exist(db)
		if err != nil {
			return common.CodeServerErr, fmt.Errorf("query models.ScriptModel(%s/%s) failed, %s", f.Kind, f.Metadata.Name, err)
		}
		if !exist {
			return common.CodeScriptNotExist, fmt.Errorf("models.ScriptModel(%s/%s) not exist", f.Kind, f.Metadata.Name)
		}
	}

	return common.CodeOK, nil
}

func (c *TaskTemplateService) CreateTaskTemplate(ctx *gin.Context) (common.ServiceCode, error) {
	log, db := common.ExtractContext(ctx)

	b, code, err := c.CheckParameters(ctx)
	if !b {
		log.Errorf("CheckParameters not pass, %s", err)
		return code, err
	}

	c.Metadata.Version = common.GenerateVersion()
	c.Metadata.CreateAt = time.Now().Format(time.DateTime)
	err = c.Create(db)
	if err != nil {
		log.Errorf("save model.TaskTemplateModel failed, %s", err)
		return code, err
	}

	res, err := c.Get(db)
	if err != nil {
		return common.CodeServerErr, err
	}

	if len(res) != 1 {
		return common.CodeScriptNotExist, fmt.Errorf("%s/%s not exist", c.Metadata.Name, c.Metadata.Version)
	}

	// 返回创建的任务模版
	c.TaskTemplate = res[0]
	return common.CodeOK, nil
}

func (c *TaskTemplateService) CheckUpdateParameters(ctx *gin.Context) (bool, common.ServiceCode, error) {
	_, db := common.ExtractContext(ctx)

	// 类型校验
	if c.Kind != TaskTemplateKind {
		return false, common.CodeScriptKindErr, fmt.Errorf("unsupport kind: %s, only support: %s", c.Kind, TaskTemplateKind)
	}

	// Name校验
	f := TaskTemplate{
		Kind:     TaskTemplateKind,
		Metadata: TaskMetadata{Name: c.Metadata.Name},
	}
	exist, err := f.Exist(db)
	if err != nil {
		return false, common.CodeServerErr, err
	}

	if !exist {
		return false, common.CodeTaskTemplateNotExist, fmt.Errorf("%s/%s not exist", c.Metadata.Name, c.Metadata.Version)
	}

	// 校验script必须存在
	for i, step := range c.Spec.Steps {
		code, err := checkTaskStep(db, &step)
		if err != nil {
			return false, code, fmt.Errorf("spec.steps[%d] check failed, %s", i, err)
		}
	}

	return true, common.CodeOK, nil
}

func (c *TaskTemplateService) UpdateTaskTemplate(ctx *gin.Context) (common.ServiceCode, error) {
	log, db := common.ExtractContext(ctx)

	b, code, err := c.CheckUpdateParameters(ctx)
	if !b {
		log.Errorf("CheckUpdateParameters not pass, %s", err)
		return code, err
	}

	c.Metadata.Version = common.GenerateVersion()
	c.Metadata.CreateAt = time.Now().Format(time.DateTime)
	err = c.Create(db)
	if err != nil {
		log.Errorf("save model.TaskTemplateModel(%s/%s) failed, %s", c.Metadata.Name, c.Metadata.Version, err)
		return code, err
	}

	res, err := c.Get(db)
	if err != nil {
		return common.CodeServerErr, err
	}

	if len(res) != 1 {
		return common.CodeScriptExisted, fmt.Errorf("%s/%s not exist", c.Metadata.Name, c.Metadata.Version)
	}

	// 返回更新后的任务模版
	c.TaskTemplate = res[0]

	// 老化掉多余的版本
	f := TaskTemplate{
		Kind:     TaskTemplateKind,
		Metadata: TaskMetadata{Name: c.Metadata.Name},
	}
	res, err = f.Get(db)
	if err == nil && len(res) > common.ReservedVersions {
		sort.Sort(TaskTemplateList(res))
		for i := common.ReservedVersions; i < len(res); i++ {
			err = res[i].Delete(db)
			if err != nil {
				log.Errorf("delete model.TaskTemplateModel(%s/%s) failed, %s", c.Metadata.Name, c.Metadata.Version, err)
			}
		}
	}

	return common.CodeOK, nil
}

func (c *TaskTemplateService) QueryTaskTemplate(ctx *gin.Context, queryMap map[string]string) (any, common.ServiceCode, error) {
	log, db := common.ExtractContext(ctx)

	res, err := common.NewPageList[[]TaskTemplateModel](db).
		QueryPaging(ctx).
		Order("id desc").
		Query(
			common.FilterFuzzyStringMap(queryMap),
		)
	if err != nil {
		err = fmt.Errorf("query models.TaskTemplateModel failed, %s", err)
		log.Error(err)
		return nil, common.CodeServerErr, err
	}

	ret := common.NewPageList[[]TaskTemplate](db)
	ret.Page = res.Page
	ret.PageSize = res.PageSize
	ret.Total = res.Total
	for i := range res.Items {
		t := &TaskTemplate{}
		err = json.Unmarshal([]byte(res.Items[i].Data), &t)
		if err != nil {
			log.Errorf("json.Unmarshal %s/%s/%s failed, %s", res.Items[i].Kind, res.Items[i].Name, res.Items[i].Version, err)
			continue
		}

		ret.Items = append(ret.Items, *t)
	}

	sort.Sort(TaskTemplateList(ret.Items))
	return ret, common.CodeOK, nil
}

func (c *TaskTemplateService) DeleteTaskTemplate(ctx *gin.Context) (common.ServiceCode, error) {
	_, db := common.ExtractContext(ctx)

	if len(c.Metadata.Name) == 0 {
		return common.CodeTaskTemplateNameEmpty, fmt.Errorf("task template name cannot be empty")
	}

	f := TaskTemplate{
		Kind:     TaskTemplateKind,
		Metadata: TaskMetadata{Name: c.Metadata.Name, Version: c.Metadata.Version},
	}
	res, err := f.Get(db)
	if err != nil {
		return common.CodeServerErr, err
	}

	if len(res) == 0 {
		return common.CodeTaskTemplateNotExist, fmt.Errorf("%s/%s not exist", c.Metadata.Name, c.Metadata.Version)
	}

	for _, r := range res {
		err = (&r).Delete(db)
		if err != nil {
			err = fmt.Errorf("delete models.TaskTemplateModel(%s/%s) failed, %s", c.Metadata.Name, c.Metadata.Version, err)
			return common.CodeServerErr, err
		}
	}

	return common.CodeOK, nil
}

func (c *TaskTemplateService) SubmitTaskTemplate(ctx *gin.Context) (*TaskService, common.ServiceCode, error) {
	_, db := common.ExtractContext(ctx)

	if len(c.Metadata.Name) == 0 {
		return nil, common.CodeTaskTemplateNameEmpty, fmt.Errorf("task template name cannot be empty")
	}

	f := TaskTemplate{
		Kind:     TaskTemplateKind,
		Metadata: TaskMetadata{Name: c.Metadata.Name, Version: c.Metadata.Version},
	}
	res, err := f.Get(db)
	if err != nil {
		return nil, common.CodeServerErr, err
	}

	if len(res) == 0 {
		return nil, common.CodeTaskTemplateNotExist, fmt.Errorf("%s/%s not exist", c.Metadata.Name, c.Metadata.Version)
	}

	task := TaskService{
		Task{
			Kind: TaskKind,
			Metadata: TaskMetadata{
				Name:    res[0].Metadata.Name + "-" + utils.RandStr(5),
				Version: common.GenerateVersion(),
			},
			Spec: TaskSpec{
				Pause:     false,
				Terminate: false,
				Steps:     res[0].Spec.Steps,
				Input:     res[0].Spec.Input,
			},
		},
	}

	code, err := task.CreateTask(ctx)
	if err != nil {
		return nil, code, err
	}

	if tasks, err := task.Get(db); err != nil {
		return nil, common.CodeServerErr, err
	} else {
		if len(tasks) == 0 {
			return nil, common.CodeTaskTemplateNotExist, fmt.Errorf("%s/%s not exist", task.Metadata.Name, task.Metadata.Version)
		}
		return &TaskService{Task: tasks[0]}, common.CodeOK, nil
	}
}
