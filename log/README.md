# NewLog
## 说明
基于zap包封装的日志包
## 用途
模块化记录日志
## 使用方法
### 初始化
调用InitLog，传入service和equipID，这样整个微服务实例的日志全部携带service和equipID，便于在排查问题的时候得知具体实例的信息
### gateway
#### 日志middleware
使用模板
```go
func Log(rawLogger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		beginTime := time.Now()
		trace := uuid.NewString()
		logger := newlog.AddTraceID(rawLogger, trace)
		c.Set("logger", logger)
		c.Set("trace", trace)

		c.Next()

		message := c.GetString("log_message")
		loglevel := c.GetInt("log_level")
		RawLogger, exist := c.Get("logger")
		if !exist {
			newlog.Log(rawLogger, newerror.LevelFatal, "Middleware Can't Get Logger")
		}
		logger, ok := RawLogger.(*zap.Logger)
		if !ok {
			newlog.Log(rawLogger, newerror.LevelFatal, "Middleware Can't Get Logger")
		}
		logger = newlog.AddLatencyAndTime(logger, beginTime)
		newlog.Log(logger, zapcore.Level(loglevel), message)
	}
}

```
#### handler
使用模板
```text
    a, _ := c.Get("logger")
	logger := a.(*zap.Logger)
	defer c.Set("logger",logger)
	……
	if err!=nil{
	    ……
	    logger=logger.AddHttpCode(HttpCode).
            AddLevel(LogLevel).
            AddError(err,StatusCode)
		return
	}
	……
	logger=logger.AddHttpCode(StatusOK).
        AddLevel(LogLevel)
	return
```
### 微服务Handler
```text
    logger:=rawLogger.AddTraceID(TraceID)
    ……
    if err!=nil{
        logger = logger.AddError(err, StatusCode).
            AddOperation(operation).
            AddLevel(LogLevel).
            Log()
		return nil, err
    }
    logger =AddOperation(operation).
         AddLevel(LogLevel).
         Log()
	return resp, nil
```