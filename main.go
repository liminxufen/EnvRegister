package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	
	"../env"
	"../httputil"
)

var (
	logger = env.NewLogger("main")

	_VERSION_ = "Unknown"
)

func panicUnless(err error) {
	if err != nil {
		logger.Fatal(err.Error())
		os.Exit(2)
	}
}

func main() {
	var version = flag.Bool("v", false, "")
	flag.Parse()
	if *version {
		// start proc with -v
		fmt.Println("Version [", _VERSION_, "]")
		return
	}
	fmt.Println("Starting xxx_server...")
	runtime.GOMAXPROCS(runtime.NumCPU())

	env.InitEnv("xxx_svr")

	xx.Init()

	//httputil.HandleAPIMap("/api/xxx", xxx.APIMap)
	httputil.HandleJsonRPC("/xxxrpc", map[string]interface{}{
		"": &httputil.xxxRpc{},
	})
	//httputil.HandleStatic("/download/", xxxx.DIR)
	panicUnless(httputil.Listen(false))
}
