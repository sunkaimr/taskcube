package task_controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/mount"
	"github.com/sunkaimr/taskcube/internal/pkg/common"
	. "github.com/sunkaimr/taskcube/internal/services/types"
	"github.com/sunkaimr/taskcube/pkg/docker"
	"github.com/sunkaimr/taskcube/pkg/utils"
	"sort"
	"time"
)

const (
	ParaPrefix = "EXECUTE_PARA_"
	ScriptName = "EXECUTE_SCRIPT_CONTENT"
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
		AgentImage: "registry.cn-beijing.aliyuncs.com/data-loom/taskcube-agent",
		APIVersion: "1.41",
	}
}

func (c *TaskController) Start() {

	go c.RunTaskCubeAgent()
	//go c.RunCreate()
	//go c.RunTaskStepsLifeCycle()

	<-c.ctx.Context.Done()
	c.ctx.Log.Info("shutdown TaskController")
}

// RunCreate 选择节点运行任务
func (c *TaskController) RunCreate() {
	c.ctx.Wg.Add(1)
	defer c.ctx.Wg.Done()

	tick := time.Tick(time.Second * 10)
	for {
		select {
		case <-tick:
			c.createHandler()
		case <-c.ctx.Context.Done():
			c.ctx.Log.Info("shutdown TaskController.RunCreate")
		}
	}
}

// RunTaskStepsLifeCycle 根据任务steps逐步执行
func (c *TaskController) RunTaskStepsLifeCycle() {
	c.ctx.Wg.Add(1)
	defer c.ctx.Wg.Done()

	tick := time.Tick(time.Second * 60)
	for {
		select {
		case <-tick:
			c.taskLifeCycleHandler()
		case <-c.ctx.Context.Done():
			c.ctx.Log.Info("shutdown TaskController.RunTaskStepsLifeCycle")
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
	return nil, nil
}

// taskStatusInit 初始化task.Status信息
func taskStatusInit(task *Task) {
	task.Status.Status = TaskStatusCreated
	task.Status.Message = ""
	task.Status.Progress = fmt.Sprintf("0/%d", len(task.Spec.Steps))
	task.Status.Steps = make([]TaskStatusStep, 0, len(task.Spec.Steps))

	for i, _ := range task.Status.Steps {
		task.Status.Steps[i].Name = task.Spec.Steps[i].Name
		task.Status.Steps[i].ContainerID = ""
		task.Status.Steps[i].Status = ""
		task.Status.Steps[i].Message = ""
		task.Status.Steps[i].ExitCode = 0
		task.Status.Steps[i].Input = task.Spec.Steps[i].Input
		task.Status.Steps[i].Output = nil
	}
}

// scheduleTask 将任务调度到合适的节点上
func scheduleTask(c *TaskController, task *Task) error {
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

// taskLifeCycleHandler 控制Running状态的任务项其他状态流转
func (c *TaskController) taskLifeCycleHandler() {
	db, log := c.ctx.DB, c.ctx.Log

	statusFilter := []TaskStatusType{TaskStatusRunning}
	tasks, err := TaskStatusFilter(c.ctx, statusFilter...)
	if err != nil {
		log.Errorf("query model.TaskModel(status IN (%v)) failed, %s", statusFilter, err)
		return
	}

	for _, task := range tasks {
		curStepIndex := findCurStepIndex(&task)

		if task.Metadata.DeleteAt != "" {
			task.Spec.Terminate = true
		}
		containerID := task.Status.Steps[curStepIndex].ContainerID

		cli, err := docker.New(&docker.ContainerOps{ServerHost: task.Spec.Host, APIVersion: c.APIVersion})
		if err != nil {
			log.Errorf("new docker client failed, %s", err)
			return
		}

		_, status, err := cli.State(c.ctx.Context, "id", containerID)
		if err != nil {
			if errors.Is(err, docker.ContainerNotExistError) {
				log.Infof("task(%s).step(%d) has been terminated container(%s/%s) not exist", task.Metadata.Name, curStepIndex, task.Spec.Host, containerID)
			} else {
				log.Errorf("task(%s).step(%d) has been terminated get container(%s/%s) status failed, %s", task.Metadata.Name, curStepIndex, task.Spec.Host, containerID, err)
				continue
			}
		}

		if task.Spec.Terminate {
			// 确保运行状态的容器都已停止
			if !errors.Is(err, docker.ContainerNotExistError) && status != string(TaskStepStatusExited) {
				log.Infof("task(%s).step(%d) has been terminated but container(%s/%s) exist need delete", task.Metadata.Name, curStepIndex, task.Spec.Host, containerID)

				if err = cli.Delete(c.ctx.Context, containerID); err != nil {
					log.Errorf("task(%s).step(%d) has been terminated but delete container(%s/%s) failed, %s", task.Metadata.Name, curStepIndex, task.Spec.Host, containerID, err)
				}
				// 等待下一次循环再删除任务确保容器一定被删除了
				continue
			}

			// 当所有容器都停止后删除该任务
			if err = task.Delete(db); err != nil {
				log.Errorf("task(%s).step(%d) has been terminated but delete container(%s/%s) failed, %s",
					task.Metadata.Name, curStepIndex, task.Spec.Host, containerID, err)
				continue
			}
		}

		if task.Spec.Pause {
			if status != string(TaskStepStatusPaused) {
				log.Infof("task(%s).step(%d) has been paused but container(%s/%s) status is %s", task.Metadata.Name, curStepIndex, task.Spec.Host, containerID, status)

				if err = cli.Pauses(c.ctx.Context, containerID); err != nil {
					log.Errorf("task(%s).step(%d) has been paused but pause container(%s/%s) failed, %s", task.Metadata.Name, curStepIndex, task.Spec.Host, containerID, err)
				}
			}
		}

		if !task.Spec.Pause {
			// TODO 确保容器处于运行状态
			if containerID == "" {
				// TODO 首次启动容器
				task.Status.Steps[curStepIndex].ContainerID, err = cli.Run(c.ctx.Context)
				if err != nil {
					log.Errorf("task(%s).step(%d) is running but but create container(%s/%s) failed, %s", task.Metadata.Name, curStepIndex, task.Spec.Host, containerID, err)
				}

			} else if status == string(TaskStepStatusPaused) {
				log.Infof("task(%s).step(%d) is running but container(%s/%s) status is %s", task.Metadata.Name, curStepIndex, task.Spec.Host, containerID, status)

				if err = cli.Unpauses(c.ctx.Context, containerID); err != nil {
					log.Errorf("task(%s).step(%d) is running but but unpause container(%s/%s) failed, %s", task.Metadata.Name, curStepIndex, task.Spec.Host, containerID, err)
				}
			} else {

			}

			// TODO 如果最后一个容器已经结束则任务完成状态
		}

		err = (&task).Update(db)
		if err != nil {
			log.Errorf("update task(%+v) failed, %s", task, err)
			continue
		}
	}
}

func findCurStepIndex(task *Task) int {
	curStep, totalStep := 1, len(task.Status.Steps)
	for i, step := range task.Status.Steps {
		if step.Status == TaskStepStatusExited {
			continue
		} else {
			curStep = i + 1
			break
		}
	}

	task.Status.Progress = fmt.Sprintf("%d/%d", curStep, totalStep)
	return curStep
}

// 创建容器将参数和脚本通过环境变量传递入容器
func (c *TaskController) createContainer(curStepIndex int, task *Task) error {
	env := make([]string, 0, 10)
	name := fmt.Sprintf("%s-%s-%s", task.Metadata.Name, task.Spec.Steps[curStepIndex].Name, utils.RandStr(5))

	script, err := c.getScriptContent(&task.Spec.Steps[curStepIndex])
	if err != nil {
		return fmt.Errorf("get %s script content failed, %s", name, err)
	}
	env = append(env, fmt.Sprintf("%s=%s", ScriptName, script))
	for k, v := range task.Spec.Input {
		env = append(env, fmt.Sprintf("%s%s=%s", ParaPrefix, k, v))
	}

	cli, err := docker.New(&docker.ContainerOps{
		ServerHost: task.Spec.Host,
		APIVersion: c.APIVersion,
		Name:       name,
		Env:        env,
		Image:      task.Spec.Steps[curStepIndex].Image,
		Entrypoint: []string{fmt.Sprintf("%s/%s", ExecPath, ExecPath), "run"},
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
	task.Status.Steps[curStepIndex].Status = TaskStepStatusCreating

	return nil
}

func (c *TaskController) getScriptContent(step *TaskSpecStep) (string, error) {
	if step.Source != "" {
		return step.Source, nil
	}
	f := Script{
		Kind:     ScriptKind,
		Metadata: ScriptMetadata{Name: step.Script},
	}
	res, err := f.Get(c.ctx.DB)
	if err != nil {
		return "", err
	}

	switch len(res) {
	case 0:
		return "", fmt.Errorf("%s/%s not exist", ScriptKind, step.Script)
	case 1:
		return res[0].Source, nil
	default:
		sort.Slice(res, func(i, j int) bool {
			if res[i].Metadata.Version > res[j].Metadata.Version {
				return true
			}
			return false
		})
		return res[0].Source, nil
	}
}
