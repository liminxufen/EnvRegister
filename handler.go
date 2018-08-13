package httputil

import (
	"fmt"
	"net/http"
	"../acl"
)

type HandlerChain []http.Handler

/*定义链式Handler*/
func (chain HandlerChain) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if r := recover(); r != nil {

			switch v := r.(type) {
			case http.Handler: // 如果err对象实现了http.Handler接口，则调用其输出函数。
				v.ServeHTTP(w, req)
			case error: // 普通err对象，打印出stack trace.
				msg := fmt.Sprintf("Internal Server Error: %s", v.Error())
				http.Error(w, msg, http.StatusInternalServerError)
			}
		}
	}()

	for _, filter := range chain {
		filter.ServeHTTP(w, req)
	}
}

func JsonAuthApify(fun interface{}) http.Handler { //创建鉴权Json格式链式Handler
	return HandlerChain{
		acl.APIAUTH,
		APILOG,
		JsonRPC(fun),
		JSON,
	}
}

func JsonApify(fun interface{}) http.Handler { //创建免鉴权Json格式链式Handler
	return HandlerChain{
		APILOG,
		JsonRPC(fun),
		JSON,
	}
}
