package common

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Context struct {
	Context context.Context
	Wg      *sync.WaitGroup
	Cancel  context.CancelFunc
	Log     *zap.SugaredLogger
	DB      *gorm.DB
	data    map[string]any
}

func NewContext() *Context {
	return &Context{
		Wg: &sync.WaitGroup{},
	}
}

func (c *Context) WithWaitGroup(wg *sync.WaitGroup) *Context {
	c.Wg = wg
	return c
}

func (c *Context) WithContext(ctx context.Context) *Context {
	c.Context = ctx
	return c
}

func (c *Context) WithCancel(cancel context.CancelFunc) *Context {
	c.Cancel = cancel
	return c
}
func (c *Context) WithLog(log *zap.SugaredLogger) *Context {
	c.Log = log
	return c
}
func (c *Context) WithDB(db *gorm.DB) *Context {
	c.DB = db
	return c
}

func (c *Context) SetData(key string, value any) *Context {
	if c.data == nil {
		c.data = make(map[string]any)
	}
	c.data[key] = value
	return c
}

func (c *Context) GetData(key string) (any, bool) {
	v, ok := c.data[key]
	return v, ok
}

type Response struct {
	ServiceCode
	Error string `json:"error,omitempty"` // 错误信息
	Data  any    `json:"data,omitempty"`  // 返回数据
}

type ServiceCode struct {
	Code    int    `json:"code"`    // 返回码
	Message string `json:"message"` // 返回信息
}

func ServiceCode2HttpCode(r ServiceCode) int {
	if r.Code == 0 {
		return http.StatusOK
	}
	return r.Code / 10000
}

func InvalidUintID(id uint) bool {
	return id == InvalidUint
}

func InvalidIntID(id int) bool {
	return id == InvalidInt
}

func ParsingQueryUintID(id string) uint {
	if i, err := strconv.Atoi(id); err == nil {
		return uint(i)
	}
	return InvalidUint
}

func GenerateVersion() string {
	return fmt.Sprintf("%s%03d", time.Now().Format("20060102150405"), time.Now().UnixMilli()%1000)
}
