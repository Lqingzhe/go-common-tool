package newlog

import (
	newerror "demo/pkg/error"
	"log"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	logger  *zap.Logger
	level   zapcore.Level
	message string
}

func InitLog(service string, equipID int) Logger {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.TimeKey = "time"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.CallerKey = "caller"
	cfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	cfg.EncoderConfig.StacktraceKey = "stacktrace"
	cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	logger, err := cfg.Build()
	if err != nil {
		log.Fatalf("Init Logger error: %v\n", err)
	}
	logger = logger.With(zap.String("service", service), zap.Int("equip_id", equipID)) // 全局使用
	return Logger{logger: logger}                                                      //设置默认值
} //注册logger,gateway就传入gateway,调用完成后defer logger.Sync()
func (i Logger) AddLatencyAndTime(beginTime time.Time) Logger {
	now := time.Now()
	i.logger = i.logger.With(zap.Duration("latency_time", now.Sub(beginTime)))
	return i
}
func (i Logger) AddTraceID(traceID string) Logger {
	i.logger = i.logger.With(zap.String("trace_id", traceID))
	return i
} //在每个handler开头调用
func (i Logger) AddError[t int | newerror.ErrorStatue](err error, statueCode t) Logger {
	i.logger = i.logger.With(zap.Int("status_code", int(statueCode)), zap.Error(err))
	return i
} //用来在发生错误时添加错误，在AddInfo前调用
func (i Logger) AddHttpCode(httpStatue int) Logger {
	i.logger = i.logger.With(zap.Int("http_code", httpStatue))
	return i
}
func (i Logger) AddOperation(operation string) Logger {
	i.logger = i.logger.With(zap.String("operation", operation))
	return i
}
func (i Logger) AddUserID(userID int64) Logger {
	i.logger = i.logger.With(zap.Int64("user_id", userID))
	return i
}
func (i Logger) AddIp(ip string) Logger {
	i.logger = i.logger.With(zap.String("ip", ip))
	return i
}
func (i Logger) AddLevel[t int8 | zapcore.Level](level t) Logger {
	i.level = zapcore.Level(level)
	return i
}
func (i Logger) Log() {
	logState := i.level
	switch logState {
	case zapcore.DebugLevel:
		i.logger.Debug(i.message)
	case zapcore.InfoLevel:
		i.logger.Info(i.message)
	case zapcore.WarnLevel:
		i.logger.Warn(i.message)
	case zapcore.ErrorLevel:
		i.logger.Error(i.message)
	case zapcore.FatalLevel:
		i.logger.Fatal(i.message)
	default:
		i.logger.Warn("Lack Set Log Level")
	}
} //日志等级见newerror包

func (i Logger) LogInitInfo(message string) {
	i.logger.Info(message)
}
func (i Logger) LogInitWarn(err error, message string) {
	i.logger.Warn(message, zap.Error(err))
} //打印初始化阶段的Warn级别日志
func (i Logger) LogInitError(err error, message string) {
	i.logger.Error(message, zap.Error(err))
} //打印初始化阶段的Error级别日志
func (i Logger) LogInitFatal(err error, message string) {
	i.logger.Fatal(message, zap.Error(err))
} //打印初始化阶段的Fatal级别日志
