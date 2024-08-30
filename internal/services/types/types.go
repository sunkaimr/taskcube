package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
)

var ErrRecordExisted = errors.New("record existed")
var ErrMultipleRecord = errors.New("multiple record found")

type TaskStepStatusType string

const (
	TaskStepStatusCreating     TaskStepStatusType = "Creating"
	TaskStepStatusInitializing TaskStepStatusType = "Initializing"
	TaskStepStatusRunning      TaskStepStatusType = "Running"
	TaskStepStatusPaused       TaskStepStatusType = "Paused"
	TaskStepStatusExited       TaskStepStatusType = "Exited"
)

type TaskStatusType string

const (
	TaskStatusCreating    TaskStatusType = "Creating"
	TaskStatusPending     TaskStatusType = "Pending"
	TaskStatusRunning     TaskStatusType = "Running"
	TaskStatusPausing     TaskStatusType = "Pausing"
	TaskStatusPaused      TaskStatusType = "Paused"
	TaskStatusSucceeded   TaskStatusType = "Succeeded"
	TaskStatusFailed      TaskStatusType = "Failed"
	TaskStatusUnknown     TaskStatusType = "Unknown"
	TaskStatusTerminating TaskStatusType = "Terminating"
	TaskStatusTerminated  TaskStatusType = "Terminated"
)

// TaskStatusCanUpdate 以下状态的任务还没有进入运行状态可以更改
var TaskStatusCanUpdate = []TaskStatusType{"", TaskStatusCreating, TaskStatusPending}

// TaskStatusCanPauseStop 以下状态的任务可以暂停或停止
var TaskStatusCanPauseStop = []TaskStatusType{"", TaskStatusCreating, TaskStatusPending, TaskStatusRunning, TaskStatusPausing, TaskStatusPaused}

type KindType string

const (
	ScriptKind       KindType = "Script"
	TaskKind         KindType = "Task"
	TaskTemplateKind KindType = "TaskTemplate"
)

type ScriptType string

const (
	ScriptTypeShell  ScriptType = "shell"
	ScriptTypePython ScriptType = "python"
)

type Script struct {
	Kind     KindType       `json:"kind" yaml:"kind"`
	Metadata ScriptMetadata `json:"metadata" yaml:"metadata"`
	Source   string         `json:"source" yaml:"source"`
}

type ScriptMetadata struct {
	Name     string     `json:"name" yaml:"name"`
	Version  string     `json:"version" yaml:"version"`
	Type     ScriptType `json:"type" yaml:"type"`
	CreateAt string     `json:"createAt,omitempty" yaml:"createAt,omitempty"`
}

type ScriptModel struct {
	gorm.Model
	Kind    KindType   `gorm:"type:varchar(64);not null;index:kind_idx;"`
	Name    string     `gorm:"type:varchar(64);not null;index:name_idx;"`
	Version string     `gorm:"type:varchar(64);not null;index:version_idx;"`
	Type    ScriptType `gorm:"type:varchar(64);not null;index:type_idx;"`
	Data    string     `gorm:"type:longtext;not null;"`
}

func (c *ScriptModel) TableName() string {
	return "script"
}

func (c *Script) Get(db *gorm.DB) ([]Script, error) {
	var res []ScriptModel
	err := db.Model(&ScriptModel{}).
		Select("data").
		Scopes(
			filter("kind", string(c.Kind)),
			filter("name", c.Metadata.Name),
			filter("version", c.Metadata.Version),
			filter("type", string(c.Metadata.Type)),
		).
		Find(&res).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	resScript := make([]Script, len(res))
	for i, r := range res {
		err = json.Unmarshal([]byte(r.Data), &resScript[i])
		if err != nil {
			return nil, err
		}
	}
	return resScript, nil
}

func (c *Script) Exist(db *gorm.DB) (bool, error) {
	var count int64
	err := db.Model(&ScriptModel{}).
		Select("data").
		Scopes(
			filter("kind", string(c.Kind)),
			filter("name", c.Metadata.Name),
			filter("version", c.Metadata.Version),
			filter("type", string(c.Metadata.Type)),
		).
		Count(&count).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	return count != 0, nil
}

func (c *Script) Create(db *gorm.DB) error {
	res, _ := c.Get(db)
	if len(res) != 0 {
		return ErrRecordExisted
	}

	m := ScriptModel{}
	m.Kind = c.Kind
	m.Name = c.Metadata.Name
	m.Version = c.Metadata.Version
	m.Type = c.Metadata.Type
	if b, err := json.Marshal(c); err != nil {
		return err
	} else {
		m.Data = string(b)
	}
	return db.Model(&ScriptModel{}).Create(&m).Error
}

func (c *Script) Delete(db *gorm.DB) error {
	err := db.Model(&ScriptModel{}).
		Scopes(
			filter("kind", string(c.Kind)),
			filter("name", c.Metadata.Name),
			filter("version", c.Metadata.Version),
			filter("type", string(c.Metadata.Type)),
		).
		Delete(&ScriptModel{}).Error
	return err
}

func (c *Script) Update(db *gorm.DB) error {
	var res []ScriptModel
	err := db.Model(&ScriptModel{}).
		Select("id").
		Scopes(
			filter("kind", string(c.Kind)),
			filter("name", c.Metadata.Name),
			filter("version", c.Metadata.Version),
			filter("type", string(c.Metadata.Type)),
		).
		Find(&res).Error
	if err != nil {
		return err
	}
	res[0].Kind = c.Kind
	res[0].Name = c.Metadata.Name
	res[0].Version = c.Metadata.Version
	res[0].Type = c.Metadata.Type
	if b, err := json.Marshal(c); err != nil {
		return err
	} else {
		res[0].Data = string(b)
	}
	return db.Model(&ScriptModel{}).Updates(&res[0]).Error
}

type ScriptList []Script

func (c ScriptList) Len() int {
	return len(c)
}

func (c ScriptList) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c ScriptList) Less(i, j int) bool {
	if c[i].Metadata.Name < c[j].Metadata.Name {
		return true
	} else if c[i].Metadata.Name > c[j].Metadata.Name {
		return false
	}

	if c[i].Metadata.Version > c[j].Metadata.Version {
		return true
	} else if c[i].Metadata.Version < c[j].Metadata.Version {
		return false
	}
	return false
}

type TaskTemplate struct {
	Kind     KindType         `json:"kind" yaml:"kind"`
	Metadata TaskMetadata     `json:"metadata" yaml:"metadata"`
	Spec     TaskTemplateSpec `json:"spec" yaml:"spec"`
}

type TaskTemplateSpec struct {
	Input map[string]string `json:"input,omitempty" yaml:"input"`
	Steps []TaskSpecStep    `json:"steps" yaml:"steps"`
}

type TaskMetadata struct {
	Name     string `json:"name" yaml:"name"`
	Version  string `json:"version,omitempty" yaml:"version,omitempty"`
	CreateAt string `json:"createAt,omitempty" yaml:"createAt,omitempty"`
	DeleteAt string `json:"deleteAt,omitempty" yaml:"deleteAt,omitempty"`
}

type TaskTemplateModel struct {
	gorm.Model
	Kind    KindType `gorm:"type:varchar(64);not null;index:kind_idx;"`
	Name    string   `gorm:"type:varchar(64);not null;index:name_idx;"`
	Version string   `gorm:"type:varchar(64);index:version_idx;"`
	Scripts string   `json:"scripts" yaml:"scripts"`
	Data    string   `gorm:"type:longtext;not null;"`
}

func (c *TaskTemplateModel) TableName() string {
	return "task_template"
}

func (c *TaskTemplate) Get(db *gorm.DB) ([]TaskTemplate, error) {
	var res []TaskTemplateModel
	err := db.Model(&TaskTemplateModel{}).
		Select("data").
		Scopes(
			filter("kind", string(c.Kind)),
			filter("name", c.Metadata.Name),
			filter("version", c.Metadata.Version),
		).
		Find(&res).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	resTaskTemplate := make([]TaskTemplate, len(res))
	for i, r := range res {
		err = json.Unmarshal([]byte(r.Data), &resTaskTemplate[i])
		if err != nil {
			return nil, err
		}
	}
	return resTaskTemplate, nil
}

func (c *TaskTemplate) Exist(db *gorm.DB) (bool, error) {
	var count int64
	err := db.Model(&TaskTemplateModel{}).
		Scopes(
			filter("kind", string(c.Kind)),
			filter("name", c.Metadata.Name),
			filter("version", c.Metadata.Version),
		).
		Count(&count).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	return count != 0, nil
}

func (c *TaskTemplate) Create(db *gorm.DB) error {
	res, _ := c.Get(db)
	if len(res) != 0 {
		return ErrRecordExisted
	}

	m := TaskTemplateModel{}
	m.Kind = c.Kind
	m.Name = c.Metadata.Name
	m.Version = c.Metadata.Version
	if b, err := json.Marshal(c); err != nil {
		return err
	} else {
		m.Data = string(b)
	}
	return db.Model(&TaskTemplateModel{}).Create(&m).Error
}

func (c *TaskTemplate) Delete(db *gorm.DB) error {
	err := db.Model(&TaskTemplateModel{}).
		Scopes(
			filter("kind", string(c.Kind)),
			filter("name", c.Metadata.Name),
			filter("version", c.Metadata.Version),
		).
		Delete(&TaskTemplateModel{}).Error
	return err
}

func (c *TaskTemplate) Update(db *gorm.DB) error {
	var res []TaskModel
	err := db.Model(&TaskModel{}).
		Select("id").
		Scopes(
			filter("kind", string(c.Kind)),
			filter("name", c.Metadata.Name),
			filter("version", c.Metadata.Version),
		).
		Find(&res).Error
	if err != nil {
		return err
	}
	res[0].Kind = c.Kind
	res[0].Name = c.Metadata.Name
	res[0].Version = c.Metadata.Version
	if b, err := json.Marshal(c); err != nil {
		return err
	} else {
		res[0].Data = string(b)
	}
	return db.Model(&TaskModel{}).Updates(&res[0]).Error
}

type TaskTemplateList []TaskTemplate

func (c TaskTemplateList) Len() int {
	return len(c)
}

func (c TaskTemplateList) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c TaskTemplateList) Less(i, j int) bool {
	if c[i].Metadata.Name < c[j].Metadata.Name {
		return true
	} else if c[i].Metadata.Name > c[j].Metadata.Name {
		return false
	}

	if c[i].Metadata.Version > c[j].Metadata.Version {
		return true
	} else if c[i].Metadata.Version < c[j].Metadata.Version {
		return false
	}
	return false
}

type Task struct {
	Kind     KindType     `json:"kind" yaml:"kind"`
	Metadata TaskMetadata `json:"metadata" yaml:"metadata"`
	Spec     TaskSpec     `json:"spec" yaml:"spec"`
	Status   TaskStatus   `json:"status,omitempty" yaml:"status,omitempty"`
}

type TaskSpec struct {
	Pause     bool              `json:"pause" yaml:"pause"`
	Terminate bool              `json:"terminate" yaml:"terminate"`
	Steps     []TaskSpecStep    `json:"steps" yaml:"steps"`
	Input     map[string]string `json:"input" yaml:"input"`
}

type TaskStatus struct {
	Status   TaskStatusType   `json:"status,omitempty" yaml:"status,omitempty"`
	Progress string           `json:"progress,omitempty" yaml:"progress,omitempty"`
	Steps    []TaskStatusStep `json:"steps,omitempty" yaml:"steps,omitempty"`
}

type TaskSpecStep struct {
	Name   string            `json:"name" yaml:"name"`
	Image  string            `json:"image" yaml:"image"`
	Script string            `json:"script,omitempty" yaml:"script,omitempty"`
	Source string            `json:"source,omitempty" yaml:"source,omitempty"`
	Input  map[string]string `json:"input,omitempty" yaml:"input,omitempty"`
}

type TaskStatusStep struct {
	Name        string             `json:"name" yaml:"name"`
	Host        string             `json:"host" yaml:"host"`
	ContainerID string             `json:"containerID" yaml:"containerID"`
	Status      TaskStepStatusType `json:"status" yaml:"status"`
	Message     string             `json:"message" yaml:"message"`
	ExitCode    int                `json:"exitCode" yaml:"exitCode"`
	Input       map[string]string  `json:"input" yaml:"input"`
	Output      map[string]string  `json:"output" yaml:"output"`
}

type TaskModel struct {
	gorm.Model
	Kind      KindType       `gorm:"type:varchar(64);not null;index:kind_idx;"`
	Name      string         `gorm:"type:varchar(64);not null;index:name_idx;"`
	Version   string         `gorm:"type:varchar(64);index:version_idx;"`
	Pause     bool           `gorm:"type:TINYINT(1);index:pause_idx;"`
	Terminate bool           `gorm:"type:TINYINT(1);index:terminate_idx;"`
	Status    TaskStatusType `gorm:"type:varchar(64);index:status_idx;"`
	Data      string         `gorm:"type:longtext;not null;"`
}

func (c *TaskModel) TableName() string {
	return "task"
}

func (c *Task) Get(db *gorm.DB) ([]Task, error) {
	var res []TaskModel
	err := db.Model(&TaskModel{}).
		Select("data").
		Scopes(
			filter("kind", string(c.Kind)),
			filter("name", c.Metadata.Name),
			filter("version", c.Metadata.Version),
			filter("status", string(c.Status.Status)),
		).
		Find(&res).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	resWorkFlow := make([]Task, len(res))
	for i, r := range res {
		err = json.Unmarshal([]byte(r.Data), &resWorkFlow[i])
		if err != nil {
			return nil, err
		}
	}
	return resWorkFlow, nil
}

func (c *Task) Exist(db *gorm.DB) (bool, error) {
	var count int64
	err := db.Model(&TaskModel{}).
		Select("id").
		Scopes(
			filter("kind", string(c.Kind)),
			filter("name", c.Metadata.Name),
			filter("version", c.Metadata.Version),
			filter("status", string(c.Status.Status)),
		).
		Find(&count).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	return count != 0, nil
}

func (c *Task) Create(db *gorm.DB) error {
	res, _ := c.Get(db)
	if len(res) != 0 {
		return ErrRecordExisted
	}

	m := TaskModel{}
	m.Kind = c.Kind
	m.Name = c.Metadata.Name
	m.Version = c.Metadata.Version
	m.Pause = c.Spec.Pause
	m.Terminate = c.Spec.Terminate
	m.Status = c.Status.Status
	if b, err := json.Marshal(c); err != nil {
		return err
	} else {
		m.Data = string(b)
	}
	return db.Model(&TaskModel{}).Create(&m).Error
}

func (c *Task) Delete(db *gorm.DB) error {
	err := db.Model(&TaskModel{}).
		Scopes(
			filter("kind", string(c.Kind)),
			filter("name", c.Metadata.Name),
			filter("version", c.Metadata.Version),
			filter("status", string(c.Status.Status)),
		).
		Delete(&TaskModel{}).Error
	return err
}

func (c *Task) Update(db *gorm.DB) error {
	var res []TaskModel
	err := db.Model(&TaskModel{}).
		Scopes(
			filter("kind", string(c.Kind)),
			filter("name", c.Metadata.Name),
			filter("version", c.Metadata.Version),
			filter("status", string(c.Status.Status)),
		).
		Find(&res).Error
	if err != nil {
		return err
	}

	if len(res) > 1 {
		return ErrMultipleRecord
	}

	res[0].Kind = c.Kind
	res[0].Name = c.Metadata.Name
	res[0].Version = c.Metadata.Version
	res[0].Pause = c.Spec.Pause
	res[0].Terminate = c.Spec.Terminate
	res[0].Status = c.Status.Status
	if b, err := json.Marshal(c); err != nil {
		return err
	} else {
		res[0].Data = string(b)
	}
	return db.Model(&TaskModel{}).Where("id =?", res[0].ID).Save(&res[0]).Error
}

func filter(name string, val string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if name == "" || val == "" {
			return db
		}
		return db.Where(fmt.Sprintf("%s = ?", name), val)
	}
}

func filterFuzzy(name string, val string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if name == "" || val == "" {
			return db
		}
		return db.Where(fmt.Sprintf("%s LIKE ?", "%"+name+"%"), val)
	}
}

type TaskList []Task

func (c TaskList) Len() int {
	return len(c)
}

func (c TaskList) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c TaskList) Less(i, j int) bool {
	if c[i].Metadata.Name < c[j].Metadata.Name {
		return true
	} else if c[i].Metadata.Name > c[j].Metadata.Name {
		return false
	}

	if c[i].Metadata.Version < c[j].Metadata.Version {
		return true
	} else if c[i].Metadata.Version > c[j].Metadata.Version {
		return false
	}
	return false
}
