package services

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sunkaimr/taskcube/internal/pkg/common"
	. "github.com/sunkaimr/taskcube/internal/services/types"
	"github.com/sunkaimr/taskcube/pkg/utils"
	"github.com/tidwall/gjson"
	"gorm.io/gorm"
	"regexp"
	"sort"
	"strings"
	"time"
)

type TaskService struct {
	Task
}

func checkTaskSteps(db *gorm.DB, task *Task) (common.ServiceCode, error) {
	for i, step := range task.Spec.Steps {
		// 名字不能为空
		if step.Name == "" {
			return common.CodeTaskTemplateStepNameEmpty, fmt.Errorf("Task.Spec.Steps[%d].Name can not be empty", i)
		}

		// 镜像不能为空
		if step.Image == "" {
			return common.CodeTaskTemplateStepImageEmpty, fmt.Errorf("Task.Spec.Steps[%d].Image can not be empty", i)
		}

		// 脚本不能为空
		if step.Script == "" && step.Source == "" {
			return common.CodeTaskTemplateScriptSourceEmpty, fmt.Errorf("Task.Spec.Steps[%d].Script or Task.Spec.Steps[%d].Source can not both empty", i, i)
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
				return common.CodeServerErr, fmt.Errorf("query Task.Spec.Steps[%d].Script(%s/%s) from db failed, %s", i, f.Kind, f.Metadata.Name, err)
			}
			if !exist {
				return common.CodeScriptNotExist, fmt.Errorf("Task.Spec.Steps[%d].Script(%s/%s) not exist", i, f.Kind, f.Metadata.Name)
			}
		}

		// Input可以引用Task.Metadata 和 Task.Spec里的静态数据 {{Spec.Steps.2.Output.step2_out}}
		// Input如果引用了其他的Task.Spec.Steps[].Output则其必须已事先声明且要符合逻辑先后顺序
		for k, v := range step.Input {
			if !IsReferenceValue(v) {
				continue
			}

			if _, err := GetReferenceValue(v, task); err != nil {
				return common.CodeTaskReferenceValueNotExist, fmt.Errorf("check Task.Spec.Steps[%d].Input.%s reference value failed, %s", i, k, err)
			}
		}
	}

	return common.CodeOK, nil
}

func checkTaskInput(task *Task) (common.ServiceCode, error) {
	for k, v := range task.Spec.Input {
		if !IsReferenceValue(v) {
			continue
		}

		if _, err := GetReferenceValue(v, task); err != nil {
			return common.CodeTaskReferenceValueNotExist, fmt.Errorf("check Task.Spec.Input.%s reference value failed, %s", k, err)
		}
	}
	return common.CodeOK, nil
}

func checkTaskOutput(task *Task) (common.ServiceCode, error) {
	for k, v := range task.Spec.Output {
		if !IsReferenceValue(v) {
			continue
		}

		if _, err := GetReferenceValue(v, task); err != nil {
			return common.CodeTaskReferenceValueNotExist, fmt.Errorf("check Task.Spec.Output.%s reference value failed, %s", k, err)
		}
	}
	return common.CodeOK, nil
}

func IsReferenceValue(s string) bool {
	return strings.HasPrefix(s, "{{") && strings.HasSuffix(s, "}}")
}

func GetReferenceValue(key string, data any) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	key, _ = strings.CutPrefix(key, "{{")
	key, _ = strings.CutSuffix(key, "}}")

	value := gjson.Get(string(b), key)
	// 未找到这个节点
	if value.Index == 0 {
		return "", fmt.Errorf("cannot found the reference value: %s", key)
	}
	return value.String(), nil
}

func SetReferenceValue(data map[string]string, task *Task) {
	for k, v := range data {
		if !IsReferenceValue(v) {
			data[k] = v
			continue
		}

		// 替换"Spec"为"Status"
		v1 := regexp.MustCompile(`{{Spec\.Steps\.\d+\.Output\..+}}`).ReplaceAllStringFunc(v, func(match string) string {
			return strings.Replace(match, "Spec", "Status", 1)
		})

		v2, err := GetReferenceValue(v1, task)
		if err != nil {
			data[k] = v
		} else {
			data[k] = v2
		}
	}
}

func (c *TaskService) CheckParameters(ctx *gin.Context) (bool, common.ServiceCode, error) {
	_, db := common.ExtractContext(ctx)

	// 类型校验
	if c.Kind != TaskKind {
		return false, common.CodeTaskKindErr, fmt.Errorf("unsupport kind: %s, only support: %s", c.Kind, TaskKind)
	}

	// Name校验（不允许重名）
	f := Task{
		Kind:     TaskKind,
		Metadata: TaskMetadata{Name: c.Metadata.Name},
	}

	if exist, err := f.Exist(db); err != nil {
		return false, common.CodeServerErr, err
	} else if exist {
		return false, common.CodeTaskExisted, fmt.Errorf("%s/%s existed", c.Kind, c.Metadata.Name)
	}

	// 校验steps合法性
	code, err := checkTaskSteps(db, &c.Task)
	if err != nil {
		return false, code, fmt.Errorf("check Task.Spec.Steps failed, %s", err)
	}

	// 校验Spec.Input 和 Spec.Output 合法性
	code, err = checkTaskInput(&c.Task)
	if err != nil {
		return false, code, fmt.Errorf("check Task.Spec.Input failed, %s", err)
	}

	code, err = checkTaskOutput(&c.Task)
	if err != nil {
		return false, code, fmt.Errorf("check Task.Spec.Output failed, %s", err)
	}

	return true, common.CodeOK, nil
}

func (c *TaskService) CreateTask(ctx *gin.Context) (common.ServiceCode, error) {
	log, db := common.ExtractContext(ctx)

	b, code, err := c.CheckParameters(ctx)
	if !b {
		log.Errorf("CheckParameters not pass, %s", err)
		return code, err
	}

	c.Metadata.Version = utils.Ternary[string](c.Metadata.Version == "", common.GenerateVersion(), c.Metadata.Version)
	c.Metadata.CreateAt = time.Now().Format(time.RFC3339)
	c.Status.Status = TaskStatusCreated
	err = c.Create(db)
	if err != nil {
		log.Errorf("save model.TaskModel failed, %s", err)
		return code, err
	}

	res, err := c.Get(db)
	if err != nil {
		return common.CodeServerErr, err
	} else if len(res) != 1 {
		return common.CodeScriptNotExist, fmt.Errorf("%s/%s not exist", c.Metadata.Name, c.Metadata.Version)
	} else {
		// 返回创建的任务
		c.Task = res[0]
	}

	return common.CodeOK, nil
}

func (c *TaskService) CheckUpdateParameters(ctx *gin.Context) (bool, common.ServiceCode, error) {
	_, db := common.ExtractContext(ctx)

	// 类型校验
	if c.Kind != TaskKind {
		return false, common.CodeTaskKindErr, fmt.Errorf("unsupport kind: %s, only support: %s", c.Kind, TaskKind)
	}

	// Name校验
	f := Task{
		Kind:     TaskKind,
		Metadata: TaskMetadata{Name: c.Metadata.Name},
	}
	res, err := f.Get(db)
	if err != nil {
		return false, common.CodeServerErr, err
	} else if len(res) == 0 {
		return false, common.CodeTaskNotExist, fmt.Errorf("%s/%s not exist", c.Kind, c.Metadata.Name)
	}

	if !utils.ElementExist(res[0].Status.Status, TaskStatusCanUpdate) {
		return false, common.CodeTaskImmutable, fmt.Errorf("%s/%s status %s is immutable", c.Kind, c.Metadata.Name, res[0].Status.Status)
	}

	// 校验steps合法性
	code, err := checkTaskSteps(db, &c.Task)
	if err != nil {
		return false, code, fmt.Errorf("check Task.Spec.Steps failed, %s", err)
	}

	// 校验Spec.Input 和 Spec.Output 合法性
	code, err = checkTaskInput(&c.Task)
	if err != nil {
		return false, code, fmt.Errorf("check Task.Spec.Input failed, %s", err)
	}

	code, err = checkTaskOutput(&c.Task)
	if err != nil {
		return false, code, fmt.Errorf("check Task.Spec.Output failed, %s", err)
	}
	return true, common.CodeOK, nil
}

func (c *TaskService) UpdateTask(ctx *gin.Context) (common.ServiceCode, error) {
	log, db := common.ExtractContext(ctx)

	b, code, err := c.CheckUpdateParameters(ctx)
	if !b {
		log.Errorf("CheckUpdateParameters not pass, %s", err)
		return code, err
	}

	err = c.Update(db)
	if err != nil {
		log.Errorf("update task(%v) failed, %s", c, err)
		return common.CodeServerErr, err
	}

	f := Task{
		Kind:     TaskKind,
		Metadata: TaskMetadata{Name: c.Metadata.Name},
	}
	res, err := f.Get(db)
	if err != nil {
		return common.CodeServerErr, err
	} else if len(res) < 1 {
		return common.CodeTaskNotExist, fmt.Errorf("%s not exist", c.Metadata.Name)
	} else {
		// 返回更新后的脚本
		c.Task = res[0]
	}

	return common.CodeOK, nil
}

func (c *TaskService) QueryTask(ctx *gin.Context, queryMap map[string]string) (any, common.ServiceCode, error) {
	log, db := common.ExtractContext(ctx)
	pause, pauseOk := ctx.GetQuery("pause")
	terminate, terminateOk := ctx.GetQuery("terminate")

	res, err := common.NewPageList[[]TaskModel](db).
		QueryPaging(ctx).
		Order("id desc").
		Query(
			common.FilterFuzzyStringMap(queryMap),
			common.FilterCustomBool("pause", pause, pauseOk),
			common.FilterCustomBool("terminate", terminate, terminateOk),
		)
	if err != nil {
		err = fmt.Errorf("query models.TaskModel failed, %s", err)
		log.Error(err)
		return nil, common.CodeServerErr, err
	}

	ret := common.NewPageList[[]Task](db)
	ret.Page = res.Page
	ret.PageSize = res.PageSize
	ret.Total = res.Total
	for i := range res.Items {
		t := &Task{}
		err = json.Unmarshal([]byte(res.Items[i].Data), &t)
		if err != nil {
			log.Errorf("json.Unmarshal %s/%s/%s failed, %s", res.Items[i].Kind, res.Items[i].Name, res.Items[i].Version, err)
			continue
		}

		ret.Items = append(ret.Items, *t)
	}

	sort.Sort(TaskList(ret.Items))
	return ret, common.CodeOK, nil
}

func (c *TaskService) DeleteTask(ctx *gin.Context) (common.ServiceCode, error) {
	_, db := common.ExtractContext(ctx)

	if len(c.Metadata.Name) == 0 {
		return common.CodeTaskNameEmpty, fmt.Errorf("task name cannot be empty")
	}

	f := Task{
		Kind:     TaskKind,
		Metadata: TaskMetadata{Name: c.Metadata.Name, Version: c.Metadata.Version},
	}
	res, err := f.Get(db)
	if err != nil {
		return common.CodeServerErr, err
	} else if len(res) == 0 {
		return common.CodeScriptNotExist, fmt.Errorf("%s/%s not exist", c.Metadata.Name, c.Metadata.Version)
	}

	for _, r := range res {
		r.Metadata.DeleteAt = time.Now().Format(time.DateTime)
		err = (&r).Update(db)
		if err != nil {
			err = fmt.Errorf("delete models.TaskModel(%s/%s) failed, %s", c.Metadata.Name, c.Metadata.Version, err)
			return common.CodeServerErr, err
		}
	}

	return common.CodeOK, nil
}

func (c *TaskService) PauseTask(ctx *gin.Context) (common.ServiceCode, error) {
	log, db := common.ExtractContext(ctx)

	f := Task{
		Kind:     TaskKind,
		Metadata: TaskMetadata{Name: c.Metadata.Name},
	}
	res, err := f.Get(db)
	if err != nil {
		return common.CodeServerErr, err
	} else if len(res) == 0 {
		return common.CodeTaskNotExist, fmt.Errorf("%s/%s not exist", c.Kind, c.Metadata.Name)
	}

	// 只有某些状态的任务才可以暂停
	if !utils.ElementExist(res[0].Status.Status, TaskStatusCanPauseStop) {
		return common.CodeTaskImmutable, fmt.Errorf("%s/%s status is %s, not %v", c.Kind, c.Metadata.Name, res[0].Status.Status, TaskStatusCanPauseStop)
	}

	res[0].Spec.Pause = true
	err = res[0].Update(db)
	if err != nil {
		log.Errorf("update task(%v) failed, %s", c, err)
		return common.CodeServerErr, err
	}

	if res, err = f.Get(db); err != nil {
		return common.CodeServerErr, err
	} else if len(res) < 1 {
		return common.CodeTaskNotExist, fmt.Errorf("%s not exist", c.Metadata.Name)
	} else {
		// 返回更新后的脚本
		c.Task = res[0]
	}

	log.Infof("pause task %s/%s", c.Kind, c.Metadata.Name)
	return common.CodeOK, nil
}

func (c *TaskService) UnpauseTask(ctx *gin.Context) (common.ServiceCode, error) {
	log, db := common.ExtractContext(ctx)

	f := Task{
		Kind:     TaskKind,
		Metadata: TaskMetadata{Name: c.Metadata.Name},
	}
	res, err := f.Get(db)
	if err != nil {
		return common.CodeServerErr, err
	} else if len(res) == 0 {
		return common.CodeTaskNotExist, fmt.Errorf("%s/%s not exist", c.Kind, c.Metadata.Name)
	}

	// 只有某些状态的任务才可以暂停
	if !utils.ElementExist(res[0].Status.Status, TaskStatusCanPauseStop) {
		return common.CodeTaskImmutable, fmt.Errorf("%s/%s status is %s, not %v", c.Kind, c.Metadata.Name, res[0].Status.Status, TaskStatusCanPauseStop)
	}

	res[0].Spec.Pause = false
	err = res[0].Update(db)
	if err != nil {
		log.Errorf("update task(%v) failed, %s", c, err)
		return common.CodeServerErr, err
	}

	if res, err = f.Get(db); err != nil {
		return common.CodeServerErr, err
	} else if len(res) < 1 {
		return common.CodeTaskNotExist, fmt.Errorf("%s not exist", c.Metadata.Name)
	} else {
		// 返回更新后的脚本
		c.Task = res[0]
	}

	log.Infof("unpause task %s/%s", c.Kind, c.Metadata.Name)
	return common.CodeOK, nil
}

func (c *TaskService) StopTask(ctx *gin.Context) (common.ServiceCode, error) {
	log, db := common.ExtractContext(ctx)

	f := Task{
		Kind:     TaskKind,
		Metadata: TaskMetadata{Name: c.Metadata.Name},
	}
	res, err := f.Get(db)
	if err != nil {
		return common.CodeServerErr, err
	} else if len(res) == 0 {
		return common.CodeTaskNotExist, fmt.Errorf("%s/%s not exist", c.Kind, c.Metadata.Name)
	}

	// 只有某些状态的任务才可以暂停
	if !utils.ElementExist(res[0].Status.Status, TaskStatusCanPauseStop) {
		return common.CodeTaskImmutable, fmt.Errorf("%s/%s status is %s, not %v", c.Kind, c.Metadata.Name, res[0].Status.Status, TaskStatusCanPauseStop)
	}

	res[0].Spec.Terminate = true
	err = res[0].Update(db)
	if err != nil {
		log.Errorf("update task(%v) failed, %s", c, err)
		return common.CodeServerErr, err
	}

	if res, err = f.Get(db); err != nil {
		return common.CodeServerErr, err
	} else if len(res) < 1 {
		return common.CodeTaskNotExist, fmt.Errorf("%s not exist", c.Metadata.Name)
	} else {
		// 返回更新后的脚本
		c.Task = res[0]
	}

	log.Infof("stop task %s/%s", c.Kind, c.Metadata.Name)
	return common.CodeOK, nil
}

func (c *TaskService) TaskLogs(ctx *gin.Context, step string) (common.ServiceCode, error) {
	log, db := common.ExtractContext(ctx)

	f := Task{
		Kind:     TaskKind,
		Metadata: TaskMetadata{Name: c.Metadata.Name},
	}
	res, err := f.Get(db)
	if err != nil {
		return common.CodeServerErr, err
	} else if len(res) == 0 {
		return common.CodeTaskNotExist, fmt.Errorf("%s/%s not exist", c.Kind, c.Metadata.Name)
	}

	containerID := ""
	for _, s := range res[0].Status.Steps {
		if s.Name != step {
			continue
		}

		if s.ContainerID == "" {
			return common.CodeTaskStepContainerIDErr, fmt.Errorf("%s/%s.Status.Steps[%s].ContainerID is null", c.Kind, c.Metadata.Name, step)
		}

		containerID = s.ContainerID
		break
	}

	if containerID == "" {
		return common.CodeTaskStepNotExist, fmt.Errorf("%s/%s.Status.Steps[%s] not exist", c.Kind, c.Metadata.Name, step)
	}

	log.Infof("get task logs %s/%s/%s", c.Kind, c.Metadata.Name, step)
	return common.CodeOK, nil
}
