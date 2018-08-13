package httputil

import (
	"compress/gzip"
	"encoding/json"
	"github.com/gorilla/context"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"tusk/env"
	"tusk/errutil"
)

const (
	GZIP_MIN_LENGTH = 2048 // 响应body超过此大小，将会以gzip方式输出.
)

// 向URL发起HTTP GET请求，返回的JSON结果转换为相应对象.
func GetJSON(baseUrl string, params url.Values, ret interface{}) error {
	resp, err := http.Get(baseUrl + "?" + params.Encode())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	//log.Print(string(data))
	return json.Unmarshal(data, ret)
}

// JSON Handler.
// 将context["API.RESULT"]中的数据以JSON(或JSONP)格式输出
var logger = env.NewLogger("httputil.json")

// 将对象的JSON或JSONP形式输出到HTTP ResponseWriter.
// 如果请求参数中有 callback 参数，结果就以jsonp方式输出.
func WriteJson(w http.ResponseWriter, req *http.Request, obj interface{}) error {
	return WriteJsonWithCode(w, req, obj, 0)
}

func WriteJsonWithCode(w http.ResponseWriter, req *http.Request, obj interface{}, code int) error {
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

	//返回成功，并且要求设置Cache-Control
	if code == 0 {
		cache_settings := req.Form.Get("cache")
		if len(cache_settings) > 0 {
			w.Header().Set("Cache-Control", cache_settings)
		}
	}

	w.Header().Set("Content-Type", "application/javascript")
	shouldGzip := len(data) > GZIP_MIN_LENGTH && strings.Contains(req.Header.Get("Accept-Encoding"), "gzip")

	var writer io.Writer
	if shouldGzip {
		w.Header().Set("Vary", "Content-Encoding")
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		writer = gz
	} else {
		writer = w
	}

	if code != 0 {
		w.WriteHeader(code)
	}

	callback := req.FormValue("callback")
	if callback == "" {
		writer.Write(data)
		io.WriteString(writer, "\n")
		return nil
	}
	io.WriteString(writer, callback+"(\n")
	writer.Write(data)
	io.WriteString(writer, "\n)")
	return nil
}

/*定义JSON输出响应Handler*/
type i_JSON int

func (f i_JSON) ServeHTTP(w http.ResponseWriter, req *http.Request) { //定义JSON输出响应Handler
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
		obj1 := &errutil.APIReply{Data: c}
		err = WriteJson(w, req, obj1)
		if err != nil {
			panic(err)
		}
	}
}

const (
	JSON = i_JSON(0) // JSON输出Handler
)
