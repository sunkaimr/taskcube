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

func checkTaskTemplateSteps(db *gorm.DB, task *TaskTemplate) (common.ServiceCode, error) {
	for i, step := range task.Spec.Steps {
		// 名字不能为空
		if step.Name == "" {
			return common.CodeTaskTemplateStepNameEmpty, fmt.Errorf("TaskTemplate.Spec.Steps[%d].Name can not be empty", i)
		}

		// 镜像不能为空
		if step.Image == "" {
			return common.CodeTaskTemplateStepImageEmpty, fmt.Errorf("TaskTemplate.Spec.Steps[%d].Image can not be empty", i)
		}

		// 脚本不能为空
		if step.Script == "" && step.Source == "" {
			return common.CodeTaskTemplateScriptSourceEmpty, fmt.Errorf("TaskTemplate.Spec.Steps[%d].Script Task.Spec.Steps[%d].Source can not both empty", i, i)
		} else if step.Script != "" && step.Source != "" {
			// Script和Source同时存在以Source为主
			task.Spec.Steps[i].Script = ""
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

		// Input可以引用Task.Metadata 和 Task.Spec里的静态数据 {{Spec.Steps.2.Output.step2_out}}
		// Input如果引用了其他的Task.Spec.Steps[].Output则其必须已事先声明且要符合逻辑先后顺序
		for k, v := range step.Input {
			if !IsReferenceValue(v) {
				continue
			}

			if _, err := GetReferenceValue(v, task); err != nil {
				return common.CodeTaskReferenceValueNotExist, fmt.Errorf("check TaskTemplate.Spec.Steps[%d].Input.%s reference value failed, %s", i, k, err)
			}
		}
	}

	return common.CodeOK, nil
}

func checkTaskTemplateInput(task *TaskTemplate) (common.ServiceCode, error) {
	for k, v := range task.Spec.Input {
		if !IsReferenceValue(v) {
			continue
		}

		if _, err := GetReferenceValue(v, task); err != nil {
			return common.CodeTaskReferenceValueNotExist, fmt.Errorf("check TaskTemplate.Spec.Input.%s reference value failed, %s", k, err)
		}
	}
	return common.CodeOK, nil
}

func checkTaskTemplateOutput(task *TaskTemplate) (common.ServiceCode, error) {
	for k, v := range task.Spec.Output {
		if !IsReferenceValue(v) {
			continue
		}

		if _, err := GetReferenceValue(v, task); err != nil {
			return common.CodeTaskReferenceValueNotExist, fmt.Errorf("check TaskTemplate.Spec.Output.%s reference value failed, %s", k, err)
		}
	}
	return common.CodeOK, nil
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

	// 校验steps合法性
	code, err := checkTaskTemplateSteps(db, &c.TaskTemplate)
	if err != nil {
		return false, code, fmt.Errorf("check TaskTemplate.Spec.Steps failed, %s", err)
	}

	// 校验Spec.Input 和 Spec.Output 合法性
	code, err = checkTaskTemplateInput(&c.TaskTemplate)
	if err != nil {
		return false, code, fmt.Errorf("check TaskTemplate.Spec.Input failed, %s", err)
	}

	code, err = checkTaskTemplateOutput(&c.TaskTemplate)
	if err != nil {
		return false, code, fmt.Errorf("check TaskTemplate.Spec.Output failed, %s", err)
	}

	return true, common.CodeOK, nil
}

func (c *TaskTemplateService) CreateTaskTemplate(ctx *gin.Context) (common.ServiceCode, error) {
	log, db := common.ExtractContext(ctx)

	b, code, err := c.CheckParameters(ctx)
	if !b {
		log.Errorf("CheckParameters not pass, %s", err)
		return code, err
	}

	c.Metadata.Version = common.GenerateVersion()
	c.Metadata.CreateAt = time.Now().Format(time.RFC3339)
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

	// 校验steps合法性
	code, err := checkTaskTemplateSteps(db, &c.TaskTemplate)
	if err != nil {
		return false, code, fmt.Errorf("check TaskTemplate.Spec.Steps failed, %s", err)
	}

	// 校验Spec.Input 和 Spec.Output 合法性
	code, err = checkTaskTemplateInput(&c.TaskTemplate)
	if err != nil {
		return false, code, fmt.Errorf("check TaskTemplate.Spec.Input failed, %s", err)
	}

	code, err = checkTaskTemplateOutput(&c.TaskTemplate)
	if err != nil {
		return false, code, fmt.Errorf("check TaskTemplate.Spec.Output failed, %s", err)
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
	c.Metadata.CreateAt = time.Now().Format(time.RFC3339)
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
