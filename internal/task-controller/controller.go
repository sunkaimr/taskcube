package task_controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/mount"
	"github.com/sunkaimr/taskcube/internal/pkg/common"
	"github.com/sunkaimr/taskcube/internal/services"
	. "github.com/sunkaimr/taskcube/internal/services/types"
	"github.com/sunkaimr/taskcube/pkg/docker"
	"github.com/sunkaimr/taskcube/pkg/utils"
	"sort"
	"time"
)

const (
	ParaPrefix = "EXECUTE_PARA_"
	ScriptName = "EXECUTE_SCRIPT_CONTENT"
	OutputPath = "/agent/output"
)

type TaskController struct {
	NodePool   []string // 节点池，从节点池选择节点用于容器调度
	AgentImage string
	APIVersion string // 1.41
	ctx        *common.Context
}

func NewTaskController(ctx *common.Context) *TaskController {
	return &TaskController{
		ctx:        ctx,
		NodePool:   []string{"tcp://192.168.198.128:2375"},
		AgentImage: "registry.cn-beijing.aliyuncs.com/data-loom/taskcube-agent:latest",
		APIVersion: "1.41",
	}
}

func (c *TaskController) Start() {
	go c.RunTaskCubeAgent()
	go c.RunCreate()
	go c.RunTaskStepsLifeCycle()

	<-c.ctx.Context.Done()
	c.ctx.Log.Info("shutdown TaskController")
}

// RunCreate 选择节点运行任务
func (c *TaskController) RunCreate() {
	c.ctx.Wg.Add(1)
	defer c.ctx.Wg.Done()

	tick := time.Tick(time.Second * 3)
	for {
		select {
		case <-tick:
			c.createHandler()
		case <-c.ctx.Context.Done():
			c.ctx.Log.Info("shutdown TaskController.RunCreate")
			return
		}
	}
}

// RunTaskStepsLifeCycle 根据任务steps逐步执行
func (c *TaskController) RunTaskStepsLifeCycle() {
	c.ctx.Wg.Add(1)
	defer c.ctx.Wg.Done()

	tick := time.Tick(time.Second * 3)
	for {
		select {
		case <-tick:
			c.taskLifeCycleHandler()
		case <-c.ctx.Context.Done():
			c.ctx.Log.Info("shutdown TaskController.RunTaskStepsLifeCycle")
			return
		}
	}
}

// createHandler 任务创建后初始化任务的status信息给任务调度节点
func (c *TaskController) createHandler() {
	db, log := c.ctx.DB, c.ctx.Log

	statusFilter := []TaskStatusType{"", TaskStatusCreated, TaskStatusPending}
	tasks, err := TaskStatusFilter(c.ctx, statusFilter...)
	if err != nil {
		log.Errorf("query model.TaskModel(status IN (%v)) failed, %s", statusFilter, err)
		return
	}

	for _, task := range tasks {
		// 找到合适的节点
		err = scheduleTask(c, &task)
		if err != nil {
			log.Errorf("schedule task(%s) node failed, %s", task.Metadata.Name, err)
			continue
		}

		err = (&task).Update(db)
		if err != nil {
			log.Errorf("update task(%+v) failed, %s", task, err)
			continue
		}
	}
}

func TaskStatusFilter(ctx *common.Context, status ...TaskStatusType) ([]Task, error) {
	statusList := make([]string, 0, len(status))
	for _, s := range status {
		statusList = append(statusList, string(s))
	}

	res := make([]TaskModel, 0, 10)
	err := ctx.DB.Model(&TaskModel{}).
		Scopes(
			common.FilterMultiCondition("status", statusList),
		).
		Find(&res).Error
	if err != nil {
		return nil, err
	}

	taskList := make([]Task, 0, len(res))
	for i := range res {
		t := Task{}
		err = json.Unmarshal([]byte(res[i].Data), &t)
		if err != nil {
			ctx.Log.Errorf("json.Unmarshal %s/%s/%s failed, %s", res[i].Kind, res[i].Name, res[i].Version, err)
			continue
		}

		taskList = append(taskList, t)
	}

	sort.Sort(TaskList(taskList))
	return taskList, nil
}

// taskStatusInit 初始化task.Status信息
func taskStatusInit(task *Task) {
	task.Status.Status = TaskStatusCreated
	task.Status.Message = ""
	task.Status.Progress = fmt.Sprintf("0/%d", len(task.Spec.Steps))
	task.Status.Steps = make([]TaskStatusStep, len(task.Spec.Steps), len(task.Spec.Steps))

	setTaskStatusInput(task)

	for i, _ := range task.Spec.Steps {
		task.Status.Steps[i].Name = task.Spec.Steps[i].Name
		task.Status.Steps[i].ContainerID = ""
		task.Status.Steps[i].Status = ""
		task.Status.Steps[i].Message = ""
		task.Status.Steps[i].ExitCode = 0
		task.Status.Steps[i].Input = utils.CopyMap(task.Spec.Steps[i].Input)
	}
}

// scheduleTask 将任务调度到合适的节点上
func scheduleTask(c *TaskController, task *Task) error {
	// 初始化任务
	taskStatusInit(task)

	if len(c.NodePool) == 0 {
		return fmt.Errorf("no nodes available")
	}

	// TODO 未来有合适的方法找到合适的节点来运行任务
	// 如果未找到
	// task.State.State = TaskStatusPending

	if task.Spec.Host == "" {
		task.Spec.Host = c.NodePool[0]
	}

	task.Status.Status = TaskStatusRunning
	return nil
}

// taskLifeCycleHandler 控制Running状态的任务向其他状态流转
func (c *TaskController) taskLifeCycleHandler() {
	db, log := c.ctx.DB, c.ctx.Log

	statusFilter := []TaskStatusType{TaskStatusRunning}
	tasks, err := TaskStatusFilter(c.ctx, statusFilter...)
	if err != nil {
		log.Errorf("query model.TaskModel(status IN (%v)) failed, %s", statusFilter, err)
		return
	}

	for _, task := range tasks {
		state := ""
		curStepIndex := findCurStepIndex(&task)
		curContainerID := task.Status.Steps[curStepIndex].ContainerID

		cli, err := docker.New(&docker.ContainerOps{ServerHost: task.Spec.Host, APIVersion: c.APIVersion})
		if err != nil {
			log.Errorf("new docker client failed, %s", err)
			return
		}

		if curContainerID == "" {
			// 创建容器
			err = c.createContainer(curStepIndex, &task)
			if err != nil {
				log.Errorf("create container failed, %s", err)
				return
			}

			curStepIndex = findCurStepIndex(&task)
			curContainerID = task.Status.Steps[curStepIndex].ContainerID

			log.Infof("task(%s).step(%d) should be running create container(%s/%s) successed", task.Metadata.Name, curStepIndex, task.Spec.Host, curContainerID)

			goto updateTask
		}

		_, state, err = cli.State(c.ctx.Context, "id", curContainerID)
		if err != nil {
			if errors.Is(err, docker.ContainerNotExistError) {
				log.Warnf("task(%s).step(%d) has been terminated container(%s/%s) not exist", task.Metadata.Name, curStepIndex, task.Spec.Host, curContainerID)
			} else {
				log.Errorf("task(%s).step(%d) has been terminated get container(%s/%s) state failed, %s", task.Metadata.Name, curStepIndex, task.Spec.Host, curContainerID, err)
				continue
			}
		}

		if task.Metadata.DeleteAt != "" {
			task.Spec.Terminate = true
		}
		if task.Spec.Terminate {
			// 确保运行状态的容器都已停止
			if !errors.Is(err, docker.ContainerNotExistError) && state != string(TaskStepStatusExited) {
				log.Infof("task(%s).step(%d) has been terminated but container(%s/%s) exist need delete", task.Metadata.Name, curStepIndex, task.Spec.Host, curContainerID)

				// 更新steps[].message 任务被删除了
				if err = cli.Delete(c.ctx.Context, curContainerID); err != nil {
					msg := fmt.Sprintf("task(%s).step(%d) has been terminated but delete container(%s/%s) failed, %s", task.Metadata.Name, curStepIndex, task.Spec.Host, curContainerID, err)
					log.Error(err)
					task.Status.Steps[curStepIndex].Message = msg
				} else {
					task.Status.Steps[curStepIndex].Status = TaskStepStatusExited
					task.Status.Steps[curStepIndex].Message = fmt.Sprintf("task(%s).step(%d) has been terminated container(%s/%s) has been deleted", task.Metadata.Name, curStepIndex, task.Spec.Host, curContainerID)
				}

				task.Status.Status = TaskStatusTerminating
				// 等待下一次循环再删除任务确保容器一定被删除了
				goto updateTask
			}

			task.Status.Status = TaskStatusTerminated
			// 当所有容器都停止后删除该任务
			if err = task.Delete(db); err != nil {
				log.Errorf("task(%s).step(%d) has been terminated but delete container(%s/%s) failed, %s",
					task.Metadata.Name, curStepIndex, task.Spec.Host, curContainerID, err)
				continue
			}
		}

		if task.Spec.Pause {
			if state != string(TaskStepStatusPaused) {
				log.Infof("task(%s).step(%d) has been paused but container(%s/%s) state is %s", task.Metadata.Name, curStepIndex, task.Spec.Host, curContainerID, state)

				if err = cli.Pauses(c.ctx.Context, curContainerID); err != nil {
					msg := fmt.Sprintf("task(%s).step(%d) has been paused but pause container(%s/%s) failed, %s", task.Metadata.Name, curStepIndex, task.Spec.Host, curContainerID, err)
					log.Error(err)
					task.Status.Steps[curStepIndex].Message = msg
				} else {
					task.Status.Steps[curStepIndex].Status = TaskStepStatusPaused
					task.Status.Steps[curStepIndex].Message = fmt.Sprintf("task(%s).step(%d) has been paused container(%s/%s) has been paused", task.Metadata.Name, curStepIndex, task.Spec.Host, curContainerID)
				}

				goto updateTask
			}
		}

		if !task.Spec.Pause {
			// 确保容器处于运行状态
			switch TaskStepStatusType(state) {
			case "":
				msg := fmt.Sprintf("task(%s).step(%d) is running but container(%s/%s) not exist", task.Metadata.Name, curStepIndex, task.Spec.Host, curContainerID)
				log.Error(msg)
				task.Status.Status = TaskStatusFailed
				task.Status.Message = msg
			case TaskStepStatusCreating, TaskStepStatusCreated, TaskStepStatusInitializing, TaskStepStatusRunning:
				// 更新 steps[].status
				task.Status.Steps[curStepIndex].Status = TaskStepStatusCreating
				task.Status.Steps[curStepIndex].Message = fmt.Sprintf("task(%s).step(%d) should running container(%s/%s) state is %s", task.Metadata.Name, curStepIndex, task.Spec.Host, curContainerID, state)
			case TaskStepStatusPaused:
				log.Infof("task(%s).step(%d) is running but container(%s/%s) state is %s", task.Metadata.Name, curStepIndex, task.Spec.Host, curContainerID, state)
				if err = cli.Unpauses(c.ctx.Context, curContainerID); err != nil {
					msg := fmt.Sprintf("task(%s).step(%d) is running but but unpause container(%s/%s) failed, %s", task.Metadata.Name, curStepIndex, task.Spec.Host, curContainerID, err)
					log.Error(msg)
					task.Status.Steps[curStepIndex].Status = TaskStepStatusPaused
					task.Status.Steps[curStepIndex].Message = msg
				} else {
					log.Infof("task(%s).step(%d) should running and unpause container(%s/%s) success", task.Metadata.Name, curStepIndex, task.Spec.Host, curContainerID)
				}
			case TaskStepStatusExited:
				c.setTaskStepResult(cli, curContainerID, state, curStepIndex, &task)
			}
			goto updateTask
		}

	updateTask:
		err = (&task).Update(db)
		if err != nil {
			log.Errorf("update task(%+v) failed, %s", task, err)
		}
	}
}

func (c *TaskController) setTaskStepResult(cli *docker.Client, curContainerID string, state string, curStepIndex int, task *Task) {
	log := c.ctx.Log

	task.Status.Steps[curStepIndex].FinishedAt = time.Now().Format(time.RFC3339)
	// 判断退出码
	code, err := cli.ExitCode(c.ctx.Context, curContainerID)
	if err != nil {
		msg := fmt.Sprintf("task(%s).step(%d) is running but container(%s/%s) state is %s", task.Metadata.Name, curStepIndex, task.Spec.Host, curContainerID, state)
		log.Error(msg)
		task.Status.Steps[curStepIndex].Message = msg
		return
	}
	task.Status.Steps[curStepIndex].Status = TaskStepStatusExited
	task.Status.Steps[curStepIndex].ExitCode = code
	task.Status.Steps[curStepIndex].Message = ""
	log.Infof("task(%s).step(%d) should running but container(%s/%s) state is %s exit code is %d", task.Metadata.Name, curStepIndex, task.Spec.Host, curContainerID, state, code)

	// 从容器中获取输出值
	out, err := cli.ReadFiles(c.ctx.Context, curContainerID, OutputPath)
	if err != nil {
		msg := fmt.Sprintf("task(%s).step(%d) is running but read container(%s/%s) output failed, %s", task.Metadata.Name, curStepIndex, task.Spec.Host, curContainerID, err)
		log.Error(msg)
		task.Status.Steps[curStepIndex].Message = msg
		return
	}

	// 将容器的输出值更新至task.Status.Steps[curStepIndex].Output中
	for k, _ := range task.Spec.Steps[curStepIndex].Output {
		if task.Status.Steps[curStepIndex].Output == nil {
			task.Status.Steps[curStepIndex].Output = make(map[string]string)
		}
		task.Status.Steps[curStepIndex].Output[k] = out[k]
	}

	// 如果task.Spec.Output引用了task.Status.Steps[curStepIndex].Output中的值
	// 此时task.Status.Steps[curStepIndex].Output已更新则可以更新task.Spec.Output中的值
	setTaskStatusOutput(task)

	if code != 0 {
		task.Status.Status = TaskStatusFailed
		return
	}

	if curStepIndex+1 == len(task.Spec.Steps) {
		task.Status.Status = TaskStatusSucceeded
		log.Infof("task(%s) has finished", task.Metadata.Name)
	}
	return
}

func setTaskStatusInput(task *Task) {
	if task.Status.Input == nil {
		task.Status.Input = make(map[string]string, len(task.Spec.Input))
	}

	for k, v := range task.Spec.Input {
		task.Status.Input[k] = v
	}

	services.SetReferenceValue(task.Status.Input, task)
}

func setTaskStatusOutput(task *Task) {
	if task.Status.Output == nil {
		task.Status.Output = make(map[string]string, len(task.Spec.Output))

		for k, v := range task.Spec.Output {
			task.Status.Output[k] = v
		}
	}

	services.SetReferenceValue(task.Status.Output, task)
}

func findCurStepIndex(task *Task) int {
	curStep, totalStep := 1, len(task.Status.Steps)
	for i, step := range task.Status.Steps {
		if step.Status == TaskStepStatusExited {
			continue
		} else {
			curStep = i
			break
		}
	}

	if curStep >= len(task.Status.Steps) {
		curStep = len(task.Status.Steps) - 1
	}

	task.Status.Progress = fmt.Sprintf("%d/%d", curStep+1, totalStep)
	return curStep
}

// 创建容器将参数和脚本通过环境变量传递入容器
func (c *TaskController) createContainer(curStepIndex int, task *Task) error {
	env := make([]string, 0, 10)
	name := fmt.Sprintf("%s-%s-%s", task.Metadata.Name, task.Spec.Steps[curStepIndex].Name, utils.RandStr(5))

	scriptType, script, err := c.getScriptContent(&task.Spec.Steps[curStepIndex])
	if err != nil {
		return fmt.Errorf("get %s script content failed, %s", name, err)
	}

	task.Status.Progress = fmt.Sprintf("%d/%d", curStepIndex+1, len(task.Spec.Input))
	task.Status.Steps[curStepIndex].ContainerID = ""
	task.Status.Steps[curStepIndex].Status = TaskStepStatusCreating
	task.Status.Steps[curStepIndex].Input = utils.CopyMap(task.Spec.Input, task.Spec.Steps[curStepIndex].Input)
	task.Status.Steps[curStepIndex].StartedAt = time.Now().Format(time.RFC3339)

	services.SetReferenceValue(task.Status.Steps[curStepIndex].Input, task)

	// 将Input参数通过环境变量注入容器
	env = append(env, fmt.Sprintf("%s=%s", ScriptName, script))
	for k, v := range task.Spec.Input {
		env = append(env, fmt.Sprintf("%s%s=%s", ParaPrefix, k, v))
	}
	for k, v := range task.Status.Steps[curStepIndex].Input {
		env = append(env, fmt.Sprintf("%s%s=%s", ParaPrefix, k, v))
	}

	cli, err := docker.New(&docker.ContainerOps{
		ServerHost: task.Spec.Host,
		APIVersion: c.APIVersion,
		Name:       name,
		Env:        env,
		Image:      task.Spec.Steps[curStepIndex].Image,
		Entrypoint: []string{fmt.Sprintf("%s/%s", ExecPath, AgentName), "run", scriptType},
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeVolume,
				Source:   AgentVolumeName,
				Target:   ExecPath,
				ReadOnly: true,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("new docker client failed, %s", err)
	}

	id, err := cli.Run(c.ctx.Context)
	if err != nil {
		return fmt.Errorf("create container(%s) failed, %s", name, err)
	}
	task.Status.Steps[curStepIndex].ContainerID = id

	return nil
}

func (c *TaskController) getScriptContent(step *TaskSpecStep) (string, string, error) {
	if step.Source != "" {
		return string(ScriptTypeBash), step.Source, nil
	}
	f := Script{
		Kind:     ScriptKind,
		Metadata: ScriptMetadata{Name: step.Script},
	}
	res, err := f.Get(c.ctx.DB)
	if err != nil {
		return "", "", err
	}

	switch len(res) {
	case 0:
		return "", "", fmt.Errorf("%s/%s not exist", ScriptKind, step.Script)
	case 1:
		return string(res[0].Metadata.Type), res[0].Source, nil
	default:
		sort.Slice(res, func(i, j int) bool {
			if res[i].Metadata.Version > res[j].Metadata.Version {
				return true
			}
			return false
		})
		return string(res[0].Metadata.Type), res[0].Source, nil
	}
}
