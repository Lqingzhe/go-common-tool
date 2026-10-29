package newerror

import (
	"errors"
	"fmt"
	"testing"
)

// 防止编译器优化，包级变量接收结果
var (
	benchmarkResult error
	benchmarkErr    error
	benchmarkStr    string
)

// ========== 基础构造 ==========

// 构造一个带 4 层 trace 的错误
func newBenchError() *Error {
	return MakeError(CodeInternalError, errors.New("A Error"), LevelInfo).
		AddErrorTrace("trace 1").
		AddErrorTrace("trace 2").
		AddErrorTrace("trace 3").
		AddErrorTrace("trace 4")
}

// ========== 1. 纯 Marshal ==========

func BenchmarkMarshal(b *testing.B) {
	rawErr := newBenchError()
	b.ResetTimer()
	for range b.N {
		benchmarkErr = TranslateError(rawErr).MarshalError()
	}
}

// ========== 2. 纯 Unmarshal ==========

func BenchmarkUnmarshal(b *testing.B) {
	rawErr := newBenchError()
	marshaled := TranslateError(rawErr).MarshalError()
	b.ResetTimer()
	for range b.N {
		benchmarkResult = UnMarshalError(marshaled)
	}
}

// ========== 3. Marshal + Unmarshal（组合，你原来的场景） ==========

func BenchmarkMarshalUnmarshal(b *testing.B) {
	rawErr := newBenchError()
	b.ResetTimer()
	for range b.N {
		err := TranslateError(rawErr).MarshalError()
		benchmarkResult = UnMarshalError(err)
	}
}

// ========== 4. 原生 errors.New 基准（对照组） ==========

func BenchmarkNativeErrorsNew(b *testing.B) {
	b.ResetTimer()
	for range b.N {
		benchmarkErr = errors.New("A Error")
	}
}

// ========== 5. 原生 fmt.Errorf + %w（对照组） ==========

func BenchmarkNativeFmtErrorf(b *testing.B) {
	baseErr := errors.New("A Error")
	b.ResetTimer()
	for range b.N {
		benchmarkErr = fmt.Errorf("wrap: %w", baseErr)
	}
}

// ========== 6. 不同 trace 层数的影响 ==========

func BenchmarkMarshal_Trace1(b *testing.B) {
	rawErr := MakeError(CodeInternalError, errors.New("A Error"), LevelInfo).
		AddErrorTrace("trace 1")
	b.ResetTimer()
	for range b.N {
		benchmarkErr = TranslateError(rawErr).MarshalError()
	}
}

func BenchmarkMarshal_Trace4(b *testing.B) {
	rawErr := newBenchError()
	b.ResetTimer()
	for range b.N {
		benchmarkErr = TranslateError(rawErr).MarshalError()
	}
}

func BenchmarkMarshal_Trace10(b *testing.B) {
	rawErr := MakeError(CodeInternalError, errors.New("A Error"), LevelInfo)
	for i := range 10 {
		rawErr = rawErr.AddErrorTrace(fmt.Sprintf("trace %d", i+1))
	}
	b.ResetTimer()
	for range b.N {
		benchmarkErr = TranslateError(rawErr).MarshalError()
	}
}

// ========== 7. 不同错误消息长度的影响 ==========

func BenchmarkMarshal_ShortMsg(b *testing.B) {
	rawErr := MakeError(CodeInternalError, errors.New("short"), LevelInfo).
		AddErrorTrace("trace 1")
	b.ResetTimer()
	for range b.N {
		benchmarkErr = TranslateError(rawErr).MarshalError()
	}
}

func BenchmarkMarshal_LongMsg(b *testing.B) {
	longMsg := string(make([]byte, 1024)) // 1KB 消息
	rawErr := MakeError(CodeInternalError, errors.New(longMsg), LevelInfo).
		AddErrorTrace("trace 1")
	b.ResetTimer()
	for range b.N {
		benchmarkErr = TranslateError(rawErr).MarshalError()
	}
}

// ========== 8. 并发场景 ==========

func BenchmarkMarshalUnmarshal_Parallel(b *testing.B) {
	rawErr := newBenchError()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := TranslateError(rawErr).MarshalError()
			benchmarkResult = UnMarshalError(err)
		}
	})
}
