package errutil

import (
	"encoding/json"
	"net/http"
)

// 通用返回结果格式。
type APIReply struct {
	Retcode int         `json:"errno"`
	Retmsg  string      `json:"errmsg"`
	Data    interface{} `json:"data,omitempty"`
}

//创建新的APIError数据
func NewAPIError(code int, msg string, payload interface{}) *APIError { //创建新的APIError数据
	return &APIError{
		Code:       code,
		Message:    msg,
		Payload:    payload,
		StatusCode: 200,
	}
}

type RedirectError struct {
	StatusCode int
	Location   string
}

func (e RedirectError) Error() string {
	return e.Location
}

/*定义RedirectError Handler*/
func (e RedirectError) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// accept := r.Header.Get("accept")
	// if !strings.Contains(accept, "application/json") && !strings.Contains(accept, "text/javascript") {
	// 输出普通重定向响应
	http.Redirect(w, r, e.Location, e.StatusCode)
	// } else {
	// 输出api

	// }
}

type Forbidden struct {
	Message string
}

func (e Forbidden) Error() string {
	return e.Message
}

/*定义Forbidden Handler*/
func (e Forbidden) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	http.Error(w, e.Message, http.StatusForbidden)
}

type APIError struct {
	Code       int         //错误码
	Message    string      // 错误信息
	Payload    interface{} // 其他数据
	StatusCode int         // http状态码
}

func (e *APIError) Error() string {
	return e.Message
}

/*定义APIError Handler*/
func (e *APIError) ServeHTTP(w http.ResponseWriter, req *http.Request) { //定义APIError Handler
	ret := &APIReply{
		Retcode: e.Code,
		Retmsg:  e.Message,
		Data:    e.Payload,
	}
	statusCode := e.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusBadGateway
	}
	WriteJsonWithCodeForErr(w, req, ret, statusCode)
}

func WriteJsonWithCodeForErr(w http.ResponseWriter, req *http.Request, obj interface{}, code int) error {
	pretty := req.FormValue("_pretty_")
	var data []byte
	var err error
	if pretty != "" {
		data, err = json.MarshalIndent(obj, "", "  ")
	} else {
		data, err = json.Marshal(obj)
	}
	if err != nil {
		return err
	}
	if code != 0 {
		w.WriteHeader(code)
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Write(data)
	return err
}
