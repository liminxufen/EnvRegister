package httputil

import (
	"expvar"
	"fmt"
	rpc "github.com/gorilla/rpc"
	json "github.com/gorilla/rpc/json"
	"path"
	// "github.com/gorilla/context"
	"github.com/gorilla/mux"
	// "github.com/keep94/weblogs"
	// "github.com/keep94/weblogs/loggers"

	"mime"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"strings"
)

type httpConfigType struct {
	BaseUrl     string   `desc:"基准url"`  //HOST
	ApiBase     string   `desc:"API前缀"`  //
	Bind        string   `desc:"监听地址"`
	Https       bool     `desc:"是否启用https"`
	CrtFile     string   `desc:"https certificate file"`
	KeyFile     string   `desc:"https private key file"`
	StaticMaps  []string `desc:"静态目录映射"`
	StaticAuths []string `desc:"static auth list"`
}

var (
	HttpPathPrefix string // HTTP基准路径前缀
	Router         *mux.Router //全局路由器
	BaseUrl        string
)

var (
	logUtil    = log.NewLogger("httputil")
	httpConfig = &httpConfigType{
		BaseUrl: "https://localhost:8787/",
		Bind:    ":8787",
		Https:   true,
		CrtFile: "${CONFIG_PATH}/...",
		KeyFile: "${CONFIG_PATH}/...",
	}
)

func init() {
	env.Register("http", httpConfig)
}

func (this *httpConfigType) Init() error {
	BaseUrl = this.BaseUrl
	router := mux.NewRouter()

	// 生成HttpPathPrefix
	u, err := url.Parse(this.BaseUrl)
	if err != nil {
		return err
	}
	HttpPathPrefix = u.Path
	if !strings.HasPrefix(HttpPathPrefix, "/") {
		HttpPathPrefix = "/" + HttpPathPrefix
	}
	if !strings.HasSuffix(HttpPathPrefix, "/") {
		HttpPathPrefix += "/"
	}
	mime.AddExtensionType(".htc", "text/x-component")

	//log.Info("HttpPathPrefix=", HttpPathPrefix)
	http.Handle(HttpPathPrefix, router)

	if HttpPathPrefix == "/" {
		Router = router
	} else {
		Router = router.PathPrefix(HttpPathPrefix).Subrouter()
	}
	// http.HandleFunc(path.Join(HttpPathPrefix, "debug/vars"), expvarHandler)

	Router.HandleFunc(httpConfig.ApiBase+"/debug/vars", expvarHandler)

	return nil
}

func HandleStatic(urlPrefix, relPath string) {  //API 处理静态文件函数
	absPath := relPath
	if !strings.HasPrefix(relPath, "/") {  //若为相对路径
		absPath = path.Clean(path.Join(env.BasePath, relPath)) /env.BasePath定义为程序运行目录的上级目录
	}
	fs0 := http.FileServer(http.Dir(absPath))
	n := len(urlPrefix)
	if n == 0 || urlPrefix[n-1] != '/' {
		urlPrefix += "/"
	}

	prefix := path.Join(HttpPathPrefix, urlPrefix)
	fs := http.StripPrefix(prefix, fs0)
	if urlPrefix != "/" {
		urlPrefix = urlPrefix[0 : len(urlPrefix)-1]
		fmt.Println(urlPrefix)
	}

	// fs = rest.API{staticAUTHFilter(H_STATIC_OAAUTH), fs}
	fs = rest.HandlerChain{
		//		http.HandlerFunc(fixXUACompatible),
		//		http.HandlerFunc(modifyHeader),
		staticAUTHFilter(STATIC),
		fs,
	}

	Router.PathPrefix(urlPrefix).Handler(fs)
}

func modifyHeader(w http.ResponseWriter, r *http.Request) {
	/// P3PCP
	w.Header().Set("P3P", "CP=CAO PSA OUR")

	/// no cache
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

func fixXUACompatible(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-UA-Compatible", "IE=edge,chrome=1")
}

func staticAUTHFilter(h http.Handler) http.Handler {  //静态文件鉴权过滤Handler
	fn := func(w http.ResponseWriter, r *http.Request) {
		match := false
		for _, pa := range httpConfig.StaticAuths {
			completePath := path.Join(HttpPathPrefix, pa)
			// path join总是返回clean path，即没有最后一个'/'
			// 但是mc的path带有（静态文件不需要，如"/index.html"）
			if strings.HasSuffix(pa, "/") && !strings.HasSuffix(completePath, "/") {
				completePath += "/"
			}

			// fmt.Println("check ", completePath, r.URL.Path)
			if completePath == r.URL.Path {
				match = true
				break
			}
		}

		if match {
			// fmt.Println("do need auth :", r.URL.Path)

			h.ServeHTTP(w, r)
			modifyHeader(w, r)
			fixXUACompatible(w, r)
		} else {
			// fmt.Println("do not need auth :", r.URL.Path)
		}
	}

	return http.HandlerFunc(fn)
}

type APIMap map[string]http.Handler

func HandleAPIMap(urlPrefix string, apiMap APIMap) {
	urlPrefix = httpConfig.ApiBase + urlPrefix

	r := Router.Get(urlPrefix)
	if r == nil {
		r = Router.PathPrefix(urlPrefix).Name(urlPrefix)
	}
	sub := r.Subrouter()
	for pattern, fun := range apiMap {
		// doc := ""
		// if v, ok := fun.(APIDocer); ok {
		// 	doc = v.Doc()
		// }
		// log.Info("  ", pattern) //, "["+doc+"]")

		sub.Handle(pattern, fun)
	}
}

func HandleJsonRPC(prefix string, receivers map[string]interface{}) {
	prefix = httpConfig.ApiBase + prefix

	s := rpc.NewServer()
	s.RegisterCodec(json.NewCodec(), "application/json-rpc")
	for name, receiver := range receivers {
		err := s.RegisterService(receiver, name)
		if err != nil {
			fmt.Println(err)
		}
	}
	Router.Handle(prefix, s)
}

func expvarHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, "{\n")
	first := true
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			fmt.Fprintf(w, ",\n")
		}
		first = false
		fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprintf(w, "\n}\n")
}

func GetRequestAddress(req *http.Request) string {
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

func Listen(accessLog bool) error {
	bindAddr := env.PathReplace(httpConfig.Bind)

	var handler http.Handler
	// 处理静态文件映射
	n := len(httpConfig.StaticMaps)
	i := 0
	for i < n {
		k := httpConfig.StaticMaps[i]
		i++
		if i < n {
			v := httpConfig.StaticMaps[i]
			i++
			v = env.PathReplace(v)
			fmt.Println("Map static:", k, v)
			HandleStatic(k, v)
		}
	}

	//handler = http.DefaultServeMux
	handler = Router

	if httpConfig.Https {
		crtFile := env.PathReplace(httpConfig.CrtFile)
		keyFile := env.PathReplace(httpConfig.KeyFile)
		logUtil.Info("Start Normal HTTPS at ", bindAddr)
		return http.ListenAndServeTLS(bindAddr, crtFile, keyFile, handler)
	} else {
		logUtil.Info("Start Normal HTTP at ", bindAddr)
		return http.ListenAndServe(bindAddr, handler)
	}
	return nil
}
