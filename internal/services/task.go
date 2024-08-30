package services

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sunkaimr/taskcube/internal/pkg/common"
	. "github.com/sunkaimr/taskcube/internal/services/types"
	"github.com/sunkaimr/taskcube/pkg/utils"
	"sort"
	"time"
)

type TaskService struct {
	Task
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

	// 校验script必须存在
	for i, step := range c.Spec.Steps {
		code, err := checkTaskStep(db, &step)
		if err != nil {
			return false, code, fmt.Errorf("spec.steps[%d] check failed, %s", i, err)
		}
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
	c.Metadata.CreateAt = time.Now().Format(time.DateTime)
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

	// 校验script必须存在
	for i, step := range c.Spec.Steps {
		code, err := checkTaskStep(db, &step)
		if err != nil {
			return false, code, fmt.Errorf("spec.steps[%d] check failed, %s", i, err)
		}
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
