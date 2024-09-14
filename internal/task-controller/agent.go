package task_controller

import (
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	. "github.com/sunkaimr/taskcube/internal/services/types"
	"github.com/sunkaimr/taskcube/pkg/docker"
	"time"
)

const (
	AgentName       = "taskcube-agent"
	AgentVolumeName = "taskcube-agent"
	ExecPath        = "/agent/exec"
)

func (c *TaskController) RunTaskCubeAgent() {
	c.ctx.Wg.Add(1)
	defer c.ctx.Wg.Done()

	go c.waitAgentUpdateFinished()

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

func (c *TaskController) waitAgentUpdateFinished() {
	for {
		for _, host := range c.NodePool {
			stat, _ := c.getAgentStatus(host)
			if stat == AgentStatusUnknown || stat == AgentStatusUpdateCheck || stat == AgentStatusUpdating {
				if err := c.UpdateAgent(host); err != nil {
					c.ctx.Log.Errorf("UpdateAgent failed, %s", err)
				}
			}
		}
		time.Sleep(time.Second * 10)
	}
}

// 在节点上创建一个agent，以共享卷的方式提供执行命令的二级制文件
func (c *TaskController) createAgent() error {
	log := c.ctx.Log

	for _, host := range c.NodePool {
		if stat, reason := c.getAgentStatus(host); stat == AgentStatusUpdateCheck || stat == AgentStatusUpdating {
			log.Infof("wait host %s update finished, %s", host, reason)
			continue
		}

		// 创建agent容器
		cli, err := docker.New(&docker.ContainerOps{ServerHost: host, APIVersion: c.APIVersion})
		if err != nil {
			c.setAgentStatus(host, AgentStatusNotReady, err.Error())
			return fmt.Errorf("new docker client failed, %s", err)
		}

		// 判断volume不存在则创建volume
		if err = cli.ExistVolume(c.ctx.Context, AgentVolumeName); err != nil {
			if errors.Is(err, docker.VolumeNotExistError) {
				v, err := cli.CreateVolume(c.ctx.Context, AgentVolumeName)
				if err != nil {
					err = fmt.Errorf("create volume(%s) failed, %s", AgentVolumeName, err)
					c.setAgentStatus(host, AgentStatusNotReady, err.Error())
					return err
				}

				if v != AgentVolumeName {
					err = fmt.Errorf("create volume failed, want create volume %s but got %s", AgentVolumeName, v)
					c.setAgentStatus(host, AgentStatusNotReady, err.Error())
					return err
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
					err = fmt.Errorf("create container(%s) failed, %s", AgentName, err)
					c.setAgentStatus(host, AgentStatusNotReady, err.Error())
					return err
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
		case TaskStepStatusRunning:
			c.setAgentStatus(host, AgentStatusReady, "")
		}
	}

	return nil
}

func (c *TaskController) setAgentStatus(host string, status AgentStatusType, reason string) {
	c.AgentStatus[host] = AgentStatus{
		Time:   time.Now(),
		Status: status,
		Reason: reason,
	}
}

func (c *TaskController) getAgentStatus(host string) (AgentStatusType, string) {
	if v, ok := c.AgentStatus[host]; !ok {
		return AgentStatusUnknown, fmt.Sprintf("unknown agent: %s", host)
	} else {
		return v.Status, v.Reason
	}
}

// UpdateAgent 升级的过程
// Controller启动时检查是否需要升级 => AgentStatusUpdateCheck
// 判断运行的Agent镜像是否一致
// 如果一致                       => AgentStatusUpdated
// 如果不一致进入                  => AgentStatusUpdating
//
//	等待使用到该卷的所有容器退出后，删除该卷后
func (c *TaskController) UpdateAgent(host string) error {
	c.setAgentStatus(host, AgentStatusUpdateCheck, "")

	// 检查agent版本是否需要升级
	cli, err := docker.New(&docker.ContainerOps{ServerHost: host, APIVersion: c.APIVersion})
	if err != nil {
		c.setAgentStatus(host, AgentStatusUpdateCheck, err.Error())
		return fmt.Errorf("new docker client failed, %s", err)
	}
	containerID, _, err := cli.State(c.ctx.Context, "name", AgentName)
	if err != nil && !errors.Is(docker.ContainerNotExistError, err) {
		err = fmt.Errorf("get agent container(%s) stat failed, %s", AgentName, err)
		c.setAgentStatus(host, AgentStatusUpdateCheck, err.Error())
		return err
	} else if err == nil {
		inspect, err := cli.Inspect(c.ctx.Context, containerID)
		if err != nil {
			err = fmt.Errorf("get agent container(%s) image failed, %s", AgentName, err)
			c.setAgentStatus(host, AgentStatusUpdateCheck, err.Error())
			return err
		}
		// 镜像一致无需升级
		if inspect.Config.Image == c.AgentImage {
			c.setAgentStatus(host, AgentStatusUpdated, "")
			return nil
		}
	}

	// 等待使用到该卷的所有容器退出后，删除该卷
	c.setAgentStatus(host, AgentStatusUpdating, "")

	_ = cli.Delete(c.ctx.Context, containerID)

	containers, err := cli.ContainerList(c.ctx.Context, container.ListOptions{All: true})
	if err != nil {
		c.setAgentStatus(host, AgentStatusUpdating, err.Error())
		return err
	}

	// 遍历容器，查找使用指定卷的容器
	var usedVolumeContainer []string
	for _, v := range containers {
		for _, m := range v.Mounts {
			if m.Name == AgentVolumeName {
				usedVolumeContainer = append(usedVolumeContainer, v.ID)
			}
		}
	}

	if len(usedVolumeContainer) != 0 {
		err = fmt.Errorf("remove volume(%s) failed, volume used for: %v", AgentVolumeName, usedVolumeContainer)
		c.setAgentStatus(host, AgentStatusUpdating, err.Error())
		return err
	}

	err = cli.DeleteVolume(c.ctx.Context, AgentVolumeName, true)
	if err != nil {
		err = fmt.Errorf("remove volume(%s) failed, %s", AgentVolumeName, err.Error())
		c.setAgentStatus(host, AgentStatusUpdating, err.Error())
		return err
	}

	c.setAgentStatus(host, AgentStatusUpdated, "")
	return nil
}
