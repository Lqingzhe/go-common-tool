package newlog

import "context"

func GetCtxLogger(ctx context.Context) (logger Logger, exist bool) {
	rawLogger := ctx.Value("logger")
	if rawLogger == nil {
		return InitLog("NoService,GetCtxLogger Can't Get Logger", -1), false
	}
	logger, ok := rawLogger.(Logger)
	if !ok {
		return InitLog("NoService,GetCtxLogger Can't Type Assertion To Logger", -2), false
	}
	return logger, true
}
