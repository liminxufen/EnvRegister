package toolbox

import (
	"fmt"
	"reflect"
	"runtime"
	"time"
	"../log"
)

func Routine(i interface{}, interval time.Duration, logger *log.Logger) {
	fn, ok := i.(func() (err error))
	if !ok {
		logger.Errorf("%v|not a func()(error)", i)
		return
	}
	fnName := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	logger.Infof("start routine|%v|%v", fnName, interval)

	_fn := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic|%v", r)
			}
			return
		}()
		err = fn()
		return
	}

	for {
		if err := _fn(); err != nil {
			logger.Errorf("run routine error|%v|%v", fnName, err)
		}
		time.Sleep(interval)
	}
}
