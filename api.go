// HTTP RESTful API库.
//
// 提供URL
package rest

import (
	"fmt"
	"net/http"
	"github.com/gorilla/context"
)

type HandlerChain []http.Handler

func (chain HandlerChain) ServeHTTP(w http.ResponseWriter, req *http.Request) { //链式调用http.Handler
	defer func() {
		if r := recover(); r != nil {

			switch v := r.(type) {
			case http.Handler: // 如果err对象实现了http.Handler接口，则调用其输出函数。
				v.ServeHTTP(w, req)
			case error: // 普通err对象，打印出stack trace.
				// buf := make([]byte, 4096)
				// n := runtime.Stack(buf, false)
				// msg := fmt.Sprintf("Internal Server Error: %s\n\n%s\n", v.Error(), string(buf[:n]))
				// log.Error(req.RequestURI, " ", v.Error())
				// log.Error(string(buf[:n]))

				msg := fmt.Sprintf("Internal Server Error: %s", v.Error())
				// hij, ok := w.(http.Hijacker)
				// if !ok {
				// 	fmt.Println(v)
				http.Error(w, msg, http.StatusInternalServerError)
				// } else {
				// 	fmt.Println(hij)
				// }

			}
		}
	}()

	for _, filter := range chain {
		filter.ServeHTTP(w, req)
	}
}

// 通用返回结果格式。
type APIReply struct {
	Retcode int         `json:"errno"`
	Retmsg  string      `json:"errmsg"`
	Data    interface{} `json:"data,omitempty"`
}


// JSON Handler.
// 将context["API.RESULT"]中的数据以JSON(或JSONP)格式输出

var logger = log.NewLogger("rest.json")

type t_JSON int

func (f t_JSON) ServeHTTP(w http.ResponseWriter, req *http.Request) { //链式处理最后一步，输出JSON响应
	obj := context.Get(req, CTX_API_RESULT)

	var err error
	switch c := obj.(type) {
	case chan interface{}: // TODO: 尚不支持server push
		for v := range c {
			err = WriteJson(w, req, v)
			if err != nil {
				panic(err)
			}
		}
	default:
		obj1 := &APIReply{Data: c}
		err = WriteJson(w, req, obj1)
		if err != nil {
			panic(err)
		}
	}
}

const (
	JSON = t_JSON(0) // JSON输出Handler
)
