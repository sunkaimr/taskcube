package models

import (
	"database/sql/driver"
	"errors"
	l "github.com/sunkaimr/taskcube/internal/pkg/logger"
	"github.com/sunkaimr/taskcube/internal/pkg/mysql"
	"gorm.io/gorm"
	"time"
)

type Model struct {
	ID        uint           `json:"id" gorm:"primary_key;AUTO_INCREMENT;comment:ID"`
	CreatedAt time.Time      `json:"created_at" gorm:"type:DATETIME;comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"type:DATETIME;comment:修改时间"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"type:DATETIME;index;comment:删除时间"`
	Creator   string         `json:"creator" gorm:"type:varchar(64);not null;comment:创建人"`
	Editor    string         `json:"editor" gorm:"type:varchar(64);comment:修改人"`
}

func UpdateModels() {
	err := mysql.DB.AutoMigrate(
		&Task{},
	)
	if err != nil {
		l.Log.Error(err)
	}

}

type JSON []byte

func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	s, ok := value.([]byte)
	if !ok {
		return errors.New("invalid scan source")
	}
	*j = append((*j)[0:0], s...)
	return nil
}

func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

func (j JSON) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return j, nil
}

func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("null point exception")
	}
	*j = append((*j)[0:0], data...)
	return nil
}

func (j JSON) String() string {
	if j == nil {
		return ""
	}
	return string(j)
}
