package ext

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

func getAppSecret(app string) (secret string, err error) {
	sql := "select secret from xxxx_db where app=?"
	rows, err := db.Query(sql, app)
	if err != nil {
		return
	}
	defer rows.Close()

	if !rows.Next() {
		err = fmt.Errorf("not exits")
		return
	}
	rows.Scan(&secret)
	return
}

func md5Str(str string) (result string) {
	hasher := md5.New()
	hasher.Write([]byte(str))
	result = hex.EncodeToString(hasher.Sum(nil))
	return
}

func sign(params map[string]string) (result string, err error) {
	defer func() {
		if rErr := recover(); rErr != nil {
			err = fmt.Errorf("sign|panic|%v", rErr)
		}
		return
	}()
	app := params["_app"]
	secret, err := getAppSecret(app)
	if err != nil {
		err = fmt.Errorf("deny: app not exist")
		return
	}
	var vs = []string{secret}
	var ks = []string{}
	for k, _ := range params {
		if k == "_sign" {
			continue
		}
		ks = append(ks, k)
	}
	sort.Strings(ks)
	for _, k := range ks {
		vs = append(vs, params[k])
	}
	signStr := strings.Join(vs, ":")
	result = md5Str(signStr)
	return
}

func CheckSign(params map[string]string) (err error) {
	defer func() {
		if rErr := recover(); rErr != nil {
			err = fmt.Errorf("checkSign|panic|%v", rErr)
		}
		return
	}()
	for _, k := range []string{"_t", "_app", "_sign"} {
		if _, ok := params[k]; !ok {
			err = fmt.Errorf("checkSign|not enough|%v", params)
			return
		}
	}
	ts, err := strconv.ParseInt(params["_t"], 10, 32)
	if err != nil {
		err = fmt.Errorf("checkSign|conv _t|%v|%v", params, err)
		return
	}
	diff := time.Now().Unix() - ts
	if diff > config.ExpireSeconds || diff < -1*config.ExpireSeconds {
		err = fmt.Errorf("checkSign|expire|%v|%d", params, diff)
		return
	}
	s, err := sign(params)
	if err != nil {
		err = fmt.Errorf("checkSign|get sign err|%v|%v", params, err)
		return
	}
	if params["_sign"] != s {
		err = fmt.Errorf("checkSign|sign err|%v|%v", params)
		return
	}
	return
}

type signChecker int

var SignChecker = signChecker(0)

func (s signChecker) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logExt.Infof("start SignChecker...")
	req.FormValue("") //触发form parse
	params := make(map[string]string)
	for k, vs := range req.Form {
		params[k] = vs[0]
	}
	logExt.Infof("SignChecker|%v", params)
	if e := CheckSign(params); e != nil {
		logExt.Errorf("SignChecker|签名错误|%v", e)
		panic(errutil.NewAPIError(-1, "签名错误:"+e.Error(), nil))
	}
	return
}
