package newlog

import (
	"log"
	"runtime/debug"
)

type LogLabel string

func (l LogLabel) String() string {
	return string(l)
}

const (
	LackCode LogLabel = "Lack Code"
	LackLog  LogLabel = "Lack Log"
)

var SetLabel func(LogLabel) = SetLabel_Debug

func SetLabel_Ignore(label LogLabel) {
	return
}
func SetLabel_Debug(label LogLabel) {
	LogGreenf("%s :\n%s", true, label.String(), debug.Stack())
}
func SetLabel_Test(label LogLabel) {
	LogYellowf("%s :\n%s", true, label.String(), debug.Stack())
}
func SetLabel_Production(label LogLabel) {
	LogRedf("%s :\n%s", true, label.String(), debug.Stack())
}
func SetIgnoreModel() {
	SetLabel = SetLabel_Ignore
}
func SetDebugModel() {
	SetLabel = SetLabel_Debug
}
func SetTestModel() {
	SetLabel = SetLabel_Test
}
func SetProductionModel() {
	SetLabel = SetLabel_Production
}

func LogRedf(fmtMessage string, isBright bool, input ...any) {
	if isBright {
		log.Printf("\033[91m "+fmtMessage+" \033[0m", input...)
	} else {
		log.Printf("\033[31m "+fmtMessage+" \033[0m", input...)
	}
}
func LogGreenf(fmtMessage string, isBright bool, input ...any) {
	if isBright {
		log.Printf("\033[92m "+fmtMessage+" \033[0m", input...)
	} else {
		log.Printf("\033[32m "+fmtMessage+" \033[0m", input...)
	}
}
func LogYellowf(fmtMessage string, isBright bool, input ...any) {
	if isBright {
		log.Printf("\033[93m "+fmtMessage+" \033[0m", input...)
	} else {
		log.Printf("\033[33m "+fmtMessage+" \033[0m", input...)
	}
}
func LogBluef(fmtMessage string, isBright bool, input ...any) {
	if isBright {
		log.Printf("\033[94m "+fmtMessage+" \033[0m", input...)
	} else {
		log.Printf("\033[34m "+fmtMessage+" \033[0m", input...)
	}
}
