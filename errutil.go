package errutil

import (
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"../log"
)

func CatchErr(name string, eP *error, logger *log.Logger, vs ...interface{}) {
	if rE := recover(); rE != nil {
		*eP = fmt.Errorf("panic|%v", rE.(error))
	}

	if *eP != nil {
		s := []interface{}{name, *eP}
		segs := []string{}
		s = append(s, vs...)
		for _, _ = range s {
			segs = append(segs, "%v")
		}
		*eP = fmt.Errorf(strings.Join(segs, "|"), s...)
	}

	if logger != nil && *eP != nil {
		logger.Error((*eP).Error())
	}
	return
}

func PanicMsg(msg string) {
	if len(msg) == 0 {
		panic("")
	}
	panic(msg)
	return
}

func HandlePanicMsg(err *error) { //捕获Panic错误信息
	if rerr := recover(); rerr != nil {
		// 记录出错信息
		fmt.Println(fmt.Sprintf("%v", rerr)) // PrintPanicMsg()后可以去掉
		//Log.Error(fmt.Sprintf("%v", rerr))
		if str, ok := rerr.(string); ok {
			if len(str) == 0 {
				*err = nil
				return
			}
			*err = errors.New(str)
		} else {
			*err = errors.New("Some Err happened inside package!!! " + PadStack())
		}
	}
	return
}

func PadStack() string {
	return "\n==============================\n" + string(debug.Stack()) + "\n==============================\n"
}
func PrintPanicMsg(err error) {
	// 记录出错信息
	if err != nil {
		fmt.Println(err)
	}
	return
}
