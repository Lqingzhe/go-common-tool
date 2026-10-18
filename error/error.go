package newerror

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"go.uber.org/zap/zapcore"
)

const (
	LevelDebug = zapcore.DebugLevel // -1  调试
	LevelInfo  = zapcore.InfoLevel  //  0  一般信息
	LevelWarn  = zapcore.WarnLevel  //  1  警告
	LevelError = zapcore.ErrorLevel //  2  错误
	LevelFatal = zapcore.FatalLevel //  5  致命（谨慎使用）
)

type Error struct {
	HttpCode    int    `json:"http_code"`
	HttpMessage string `json:"http_message"`

	StatusCode int    `json:"status_code"`
	LogMessage string `json:"log_message"`
	LogLevel   int8   `json:"log_level"`

	IsNeedInterrupt bool `json:"is_need_interrupt"`
}

func (e *Error) MarshalError() error {
	if e == nil {
		return nil
	}
	return fmt.Errorf(`{"http_code":%d,"http_message":"%s","status_code":%d,"log_level":%d,"log_message":"%s","is_need_interrupt":%t}`, e.HttpCode, e.HttpMessage, e.StatusCode, e.LogLevel, e.LogMessage, e.IsNeedInterrupt)
}
func UnMarshalError(err error) error {
	if err == nil {
		return nil
	}
	errStr := err.Error()

	// 查找 JSON 开始位置
	startIdx := strings.Index(errStr, "{")
	if startIdx == -1 {
		return MakeError(CodeInternalError, fmt.Errorf("no JSON found in error: %s", errStr), LevelError).
			SetHttpError(http.StatusInternalServerError, "Unmarshal Error Failed")
	}

	jsonPart := errStr[startIdx:]
	err2 := &Error{}
	if err3 := sonic.Unmarshal([]byte(jsonPart), err2); err3 != nil {
		return MakeError(CodeInternalError, fmt.Errorf("%s : %s", `raw error:`, err.Error()), LevelFatal).
			SetHttpError(http.StatusInternalServerError, "Unmarshal Error Failed").
			AddErrorTrace("error:Unmarshal Error")
	}
	return err2
}
func (e *Error) WithContinueError() *Error {
	e.IsNeedInterrupt = false
	return e
}
func MakeError(statueCode ErrorStatue, err error, logLevel zapcore.Level) *Error {
	newStruct := &Error{
		StatusCode:      int(statueCode),
		LogMessage:      fmt.Sprintf("-> %s", err.Error()),
		LogLevel:        int8(logLevel),
		IsNeedInterrupt: true,
	}
	return newStruct
}
func (e *Error) SetHttpError(httpCode int, httpMessage string) *Error {
	e.HttpCode = httpCode
	e.HttpMessage = httpMessage
	return e
}
func MakeKafkaError(statueCode ErrorStatue, err error, logLevel zapcore.Level) *Error {
	return &Error{
		StatusCode:      int(statueCode),
		LogMessage:      fmt.Sprintf("-> %s", err.Error()),
		LogLevel:        int8(logLevel),
		IsNeedInterrupt: true,
	}
}
func (e *Error) Error() string {
	if e == nil || e.LogMessage == "" {
		return ""
	}
	return e.LogMessage
}
func TranslateError(err error) *Error {
	if err == nil {
		return nil
	}
	var err2 *Error
	err2, ok := errors.AsType[*Error](err)
	if !ok {
		return MakeError(CodeInternalError, fmt.Errorf("%s%w", `Type Assertion To "*newerror.Error" Error`, err), LevelError).
			SetHttpError(http.StatusInternalServerError, "Type Assertion Error").
			AddErrorTrace("error:TranslateError")
	}
	return err2
}
func WhetherInterrupt(err error, finalErr *error) bool {
	if err == nil {
		return false
	}
	e := TranslateError(err)
	if e == nil {
		return false
	}
	if e.IsNeedInterrupt {
		*finalErr = e
		return true
	}
	return false
}
func (e *Error) AddErrorTrace(trace string) *Error {
	if e == nil {
		return nil
	}
	e.LogMessage = trace + " / " + e.LogMessage
	return e
}
func IsContextError(err error) (bool, *Error) {
	if errors.Is(err, context.DeadlineExceeded) {
		return true, MakeError(CodeNetworkTimeout, err, LevelWarn).SetHttpError(http.StatusGatewayTimeout, "Time Out")
	}
	if errors.Is(err, context.Canceled) {
		return true, MakeError(CodeNetworkTimeout, err, LevelWarn).SetHttpError(http.StatusGatewayTimeout, "Time Out")
	}
	return false, nil
}
