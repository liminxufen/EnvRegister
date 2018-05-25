// 读取配置文件。可读取TOML格式的配置文件，采用注册配置结构体的机制，
// 自动将配置项注入配置变量。并可以调用配置结构体的初始化方法。
package env

import (
	"expvar"
	"fmt"
	std_log "log"
	"sync"
	// log "github.com/cihub/seelog"
	"github.com/echou/toml"
	"net"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
)

// 配置结构体注册

// 初始化接口。如果配置结构体实现该接口，在配置载入后就调用此接口
type Initializer interface {
	Init() error
}

// 初始化函数，实现了Initializer接口
type InitFunc func() error

func (f InitFunc) Init() error {
	return f()
}

var (
	tomlConfigMaps = make(map[string]interface{})
	initializers   []Initializer

	once sync.Once
)

/*
注册一个配置结构体。

如何读取配置？

1. 定义一个配置结构体
  type FooConfig struct {
     Foo string `desc:"foo"`
     Bar []int  `desc:"bar"`
  }
2. 定义配置文件读取后的初始化方法（可选）
  func (this *FooConfig) Init() error {
      ...
  }
3. 定义配置变量，并可设置缺省值
  var fooConfig = &FooConfig {
      Foo: "a",
      Bar: []int{1,2,3}
  }
4. 在init函数中向env注册该配置变量。注册名就是配置文件中的section名
  func init() {
      env.Register("foo", fooConfig)
  }

*/
func Register(sectionName string, config interface{}) {
	tomlConfigMaps[sectionName] = config
	if initializer, ok := config.(Initializer); ok {
		initializers = append(initializers, initializer)
	}
}

func RegisterInitializer(initializer Initializer) {
	initializers = append(initializers, initializer)
}

func RegisterInitFunc(fun InitFunc) {
	RegisterInitializer(fun)
}

// 替换字符串中出现的${..}路径变量。
func PathReplace(s string) string {
	return PathReplacer.Replace(s)
}

var ( // 配置参数
	StartType string
	LocalIP   string

	DeviceString string
)

var ( // 全局变量
	BasePath     string            // 基准路径
	PathReplacer *strings.Replacer // 用于替换配置中的${...}变量

	Eth0IP string // eth0 IP地址
	Eth1IP string // eth1 IP地址
)

const (
	PATH_ETC  = "etc"
	PATH_VAR  = "var"
	PATH_LOGS = "logs"
)

// Seelog配置定义

// type seelogConfigType struct {
// 	ConfigFile string `desc:"seelog配置文件名"`
// }

// var seelogConfig = &seelogConfigType{
// 	ConfigFile: "seelog.xml",
// }

// func (this *seelogConfigType) Init() error {
// 	// 读取seelog配置
// 	realConfigFile := path.Clean(path.Join(BasePath, PATH_ETC, this.ConfigFile))
// 	log.Debug("Seelog config file is ", realConfigFile)
// 	data, err := ioutil.ReadFile(realConfigFile)
// 	if err != nil {
// 		return err
// 	}
// 	content := PathReplace(string(data)) // 替换路径变量
// 	logger, err := log.LoggerFromConfigAsString(content)
// 	if err != nil {
// 		return err
// 	}
// 	log.ReplaceLogger(logger)
// 	return nil
// }

func indirect(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Ptr {
		return v
	}
	if v.IsNil() {
		v.Set(reflect.New(v.Type().Elem()))
	}
	return indirect(reflect.Indirect(v))
}

// 显示当前所有配置项
func Help(serverName, configName string) {

	fullPath, _ := filepath.Abs(os.Args[0]) //获取程序运行绝对路径
	curDir := path.Dir(fullPath)
	// 检查当前目录或上级目录是否有etc子目录
	BasePath = path.Clean(path.Join(curDir, "..")) //获取程序运行上级目录

	Eth0IP = getInterfaceIPv4Addr("eth0", "en0")
	Eth1IP = getInterfaceIPv4Addr("eth1", "en1")
	LocalIP = Eth1IP

	PathReplacer = strings.NewReplacer( //生成替换器
		"${BASE_PATH}", BasePath, //[old, new]...
		"${CONFIG_PATH}", path.Join(BasePath, PATH_ETC),
		"${LOG_PATH}", path.Join(BasePath, PATH_LOGS),
		"${VAR_PATH}", path.Join(BasePath, PATH_VAR),
		// ip地址
		"${ETH0_IP}", Eth0IP,
		"${ETH1_IP}", Eth1IP,
		"${LOCAL_IP}", Eth1IP,
		"${SERVER_NAME}", serverName,
		"${CONFIG_NAME}", configName,
	)

	// 载入配置
	configFilename := fmt.Sprintf("%s_%s.toml", serverName, configName) //拼凑配置文件全名
	configRealPath := path.Join(BasePath, PATH_ETC, configFilename)
	_, err := toml.DecodeFile(configRealPath, tomlConfigMaps) //解析配置文件内容到配置变量
	if err != nil {
		fmt.Println("Config file syntax error!", err)
		return
	}

	for key, obj := range tomlConfigMaps {
		v := indirect(reflect.ValueOf(obj))
		rt := v.Type()
		fmt.Printf("\n%-20s (定义于%s)\n\n", "["+key+"]", rt.PkgPath()) //打印配置变量定义包名
		for i := 0; i < rt.NumField(); i++ {                         //遍历结构体字段
			f := rt.Field(i) //字段属性为结构体类型
			desc := f.Tag.Get("desc")
			if desc == "" {
				desc = "<无描述>"
			}
			val := v.Field(i).Interface() //Value.Field仅用于结构体
			var real_val interface{}
			switch val0 := val.(type) {
			case string:
				real_val = PathReplace(val0) //替换配置文件中指定字符串
			default:
				real_val = val0
			}

			fmt.Printf("    %-20s %-15s %s: \"%v\"\n", f.Name, f.Type, desc, real_val)
		}
	}
	fmt.Println()

}

func envConfig() interface{} { return tomlConfigMaps }

func init() {
	// Register("seelog", seelogConfig)
	Register("log", logConfig)

	expvar.Publish("env", expvar.Func(envConfig))
}

func InitEnvForUT(config string) {
	PathReplacer = strings.NewReplacer(
		"${BASE_PATH}", "/tmp",
		"${CONFIG_PATH}", "/etc",
		"${LOG_PATH}", "/tmp/log",
		"${VAR_PATH}", "/tmp/var",
		// ip地址
		"${ETH0_IP}", "127.0.0.1",
		"${ETH1_IP}", "127.0.0.1",
		"${LOCAL_IP}", "127.0.0.1",
		"${SERVER_NAME}", "test",
		"${CONFIG_NAME}", "",
	)

	_, err := toml.Decode(config, tomlConfigMaps)
	if err != nil {
		panic(err)
	}

	// Initializers
	for _, initializer := range initializers {
		err = initializer.Init()
		if err != nil {
			panic(err)
		}
	}
}

func InitEnv(serverName string) {
	configName := "config"
	if len(os.Args) > 1 {
		configName = os.Args[1]
	}

	if configName == "config" {
		configName = "dev"
		if len(os.Args) > 2 {
			configName = os.Args[2]
			Help(serverName, configName)
			os.Exit(0)
		}
	}

	once.Do(func() {
		fullPath, _ := filepath.Abs(os.Args[0])
		BasePath = path.Clean(path.Join(path.Dir(fullPath), ".."))
		std_log.Println("BasePath is ", BasePath)
		std_log.Println("Server name is ", serverName)
		std_log.Println("Config name is ", configName)

		Eth0IP = getInterfaceIPv4Addr("eth0", "en0")
		Eth1IP = getInterfaceIPv4Addr("eth1", "en1")

		PathReplacer = strings.NewReplacer(
			"${BASE_PATH}", BasePath,
			"${CONFIG_PATH}", path.Join(BasePath, PATH_ETC),
			"${LOG_PATH}", path.Join(BasePath, PATH_LOGS),
			"${VAR_PATH}", path.Join(BasePath, PATH_VAR),
			// ip地址
			"${ETH0_IP}", Eth0IP,
			"${ETH1_IP}", Eth1IP,
			"${LOCAL_IP}", Eth1IP,
			"${SERVER_NAME}", serverName,
			"${CONFIG_NAME}", configName,
		)

		// 载入配置
		configFilename := fmt.Sprintf("%s_%s.toml", serverName, configName)
		configRealPath := path.Join(BasePath, PATH_ETC, configFilename)
		_, err := toml.DecodeFile(configRealPath, tomlConfigMaps)
		if err != nil {
			panic(err)
		}

		std_log.Println("Config file is ", configRealPath)
		std_log.Println("Log Path is ", path.Join(BasePath, PATH_LOGS))

		// Initializers
		for _, initializer := range initializers {
			err = initializer.Init()
			if err != nil {
				panic(err)
			}
		}
	})
}

func InitEnv4Test(basePath, serverName, configName string) { //for test
	BasePath = basePath
	once.Do(func() {
		//fullPath, _ := filepath.Abs(os.Args[0])
		//BasePath = path.Clean(path.Join(path.Dir(fullPath), ".."))
		std_log.Println("BasePath is ", BasePath)
		std_log.Println("Server name is ", serverName)
		std_log.Println("Config name is ", configName)

		Eth0IP = getInterfaceIPv4Addr("eth0", "en0")
		Eth1IP = getInterfaceIPv4Addr("eth1", "en1")

		PathReplacer = strings.NewReplacer(
			"${BASE_PATH}", BasePath,
			"${CONFIG_PATH}", path.Join(BasePath, PATH_ETC),
			"${LOG_PATH}", path.Join(BasePath, PATH_LOGS),
			"${VAR_PATH}", path.Join(BasePath, PATH_VAR),
			// ip地址
			"${ETH0_IP}", Eth0IP,
			"${ETH1_IP}", Eth1IP,
			"${LOCAL_IP}", Eth1IP,
			"${SERVER_NAME}", serverName,
			"${CONFIG_NAME}", configName,
		)

		// 载入配置
		configFilename := fmt.Sprintf("%s_%s.toml", serverName, configName)
		configRealPath := path.Join(BasePath, PATH_ETC, configFilename)
		_, err := toml.DecodeFile(configRealPath, tomlConfigMaps)
		if err != nil {
			panic(err)
		}

		std_log.Println("Config file is ", configRealPath)

		// Initializers
		for _, initializer := range initializers {
			err = initializer.Init()
			if err != nil {
				panic(err)
			}
		}
	})
}

func getInterfaceIPv4Addr(expected_intfs ...string) string { //获取本机IPv4地址
	intfs, _ := net.Interfaces()
	for _, intf := range intfs {
		name := intf.Name
		for _, intf0 := range expected_intfs {
			if intf0 != name {
				continue
			}
			addrs, _ := intf.Addrs()
			if len(addrs) > 0 {
				for _, addr := range addrs {
					ip0, _, _ := net.ParseCIDR(addr.String())
					ip := ip0.To4()
					if ip != nil {
						return ip.String()
					}
				}
			}
		}
	}

	return ""
}

func NewLogger(id string) *log.Logger {
	return log.NewLogger(id)
}
