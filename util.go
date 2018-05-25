package rest

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	//"net/url"
)

const (
	GZIP_MIN_LENGTH = 2048 // 响应body超过此大小，将会以gzip方式输出.
)

type RedirectError struct {
	StatusCode int
	Location   string
}

func (e RedirectError) Error() string {
	return e.Location
}

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

func (e Forbidden) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	http.Error(w, e.Message, http.StatusForbidden)
}

//////////////////////

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
