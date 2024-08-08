package logger

import (
	"github.com/sunkaimr/taskcube/configs"
)

func Init() {
	Log = NewZapLogger(configs.C.Log.Path, configs.C.Log.Level, configs.C.Log.MaxSize, configs.C.Log.MaxBackups, configs.C.Log.MaxAge, configs.C.Log.Compress)
}
