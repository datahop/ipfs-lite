package pkg

import (
	"reflect"
	"runtime"

	logging "github.com/ipfs/go-log/v2"
)

func CheckError(log *logging.ZapEventLogger, f func() error) {
	fn := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	if err := f(); err != nil {
		log.Debugf("%s failed : %s", fn, err)
		log.Error("%s failed :", fn)
	}
}

func CheckErrors(fs ...func() error) {
	for i := len(fs) - 1; i >= 0; i-- {
		fn := runtime.FuncForPC(reflect.ValueOf(fs).Pointer()).Name()
		if err := fs[i](); err != nil {
			log.Debugf("%s failed : %s", fn, err)
			log.Error("%s failed :", fn)
		}
	}
}
