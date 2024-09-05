package task_controller

import (
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/mount"
	. "github.com/sunkaimr/taskcube/internal/services/types"
	"github.com/sunkaimr/taskcube/pkg/docker"
	"time"
)

const (
	AgentName       = "taskcube-agent"
	AgentVolumeName = "taskcube-agent"
	ExecPath        = "/agent/exec/"
)

func (c *TaskController) RunTaskCubeAgent() {
	c.ctx.Wg.Add(1)
	defer c.ctx.Wg.Done()

	tick := time.Tick(time.Second * 10)
	for {
		select {
		case <-tick:
			err := c.createAgent()
			if err != nil {
				c.ctx.Log.Errorf("create agent failed, %s", err)
			}
		case <-c.ctx.Context.Done():
			c.ctx.Log.Info("shutdown TaskController.RunTaskCubeAgent")
			return
		}
	}
}

// 在节点上创建一个agent，以共享卷的方式提供执行命令的二级制文件
func (c *TaskController) createAgent() error {
	log := c.ctx.Log

	for _, host := range c.NodePool {
		// 创建agent容器
		cli, err := docker.New(&docker.ContainerOps{
			ServerHost: host,
			APIVersion: c.APIVersion,
		})
		if err != nil {
			return fmt.Errorf("new docker client failed, %s", err)
		}

		// 判断volume不存在则创建volume
		if err = cli.ExistVolume(c.ctx.Context, AgentVolumeName); err != nil {
			if errors.Is(err, docker.VolumeNotExistError) {
				v, err := cli.CreateVolume(c.ctx.Context, AgentVolumeName)
				if err != nil {
					return fmt.Errorf("create volume(%s) failed, %s", AgentVolumeName, err)
				}

				if v != AgentVolumeName {
					return fmt.Errorf("create volume failed, want create volume %s but got %s", AgentVolumeName, v)
				}
				log.Infof("create volume(%s) success", AgentVolumeName)
			}
		}

		containerID, status, err := cli.State(c.ctx.Context, "name", AgentName)
		if err != nil {
			if errors.Is(err, docker.ContainerNotExistError) {
				// 判断agent容器不存在则创建agent容器
				_, err = cli.
					WithContainerName(AgentName).
					WithImage(c.AgentImage).
					WithMounts(
						[]mount.Mount{
							{
								Type:     mount.TypeVolume,
								Source:   AgentVolumeName,
								Target:   ExecPath,
								ReadOnly: true,
							},
						},
					).
					Run(c.ctx.Context)
				if err != nil {
					return fmt.Errorf("create container(%s) failed, %s", AgentName, err)
				}
				log.Infof("create agent container(%s) success", AgentName)
				return nil
			}
		}

		// 判断agent容器不是Running状态则运行容器
		switch TaskStepStatusType(status) {
		case TaskStepStatusCreated:
			err = cli.Start(c.ctx.Context, containerID)
			if err != nil {
				return fmt.Errorf("container(%s) is Created start failed, %s", AgentName, err)
			}
			log.Infof("container(%s) is Created start success", AgentName)
		case TaskStepStatusPaused:
			err = cli.Unpauses(c.ctx.Context, containerID)
			if err != nil {
				return fmt.Errorf("container(%s) is Paused unpaused failed, %s", containerID, err)
			}
			log.Infof("container(%s) is Paused unpaused success", AgentName)
		case TaskStepStatusExited:
			err = cli.Restart(c.ctx.Context, containerID)
			if err != nil {
				return fmt.Errorf("container(%s) is Exited restart failed, %s", AgentName, err)
			}
			log.Infof("container(%s) is Exited restart success", AgentName)

		}
	}

	return nil
}
