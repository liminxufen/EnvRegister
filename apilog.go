package httputil

import (
	"net/http"
	"strings"
	"auth/acl"
	"../log"
)

// APILOG Handler.
// 打印API的访问日志
type i_APILOG int

var logApi = log.NewLogger("httputil.apilog")

func (f i_APILOG) ServeHTTP(w http.ResponseWriter, req *http.Request) { //定义APILOG Handler
	loginName := ""
	ui := acl.GetUserInfo(req)
	if ui != nil {
		loginName = ui.LoginName
	}

	logApi.Infof("url=%s, loginName=%s, remoteAddr=%s", req.URL.RequestURI(), loginName, getRequestAddress(req))
}

func getRequestAddress(req *http.Request) string { //获取请求地址
	address := ""
	forwardedfor := req.Header.Get("X-Forwarded-For")
	if forwardedfor != "" {
		parts := strings.Split(forwardedfor, ",")
		if len(parts) >= 1 {
			address = parts[0]
		}
	}
	if address == "" {
		address = req.RemoteAddr
		i := strings.LastIndex(address, ":")
		if i != -1 {
			address = address[:i]
		}
	}
	return address
}

const (
	APILOG = i_APILOG(0) // APILOG Handler
)
