package cloudtoken

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/tickstep/cloudpan189-api/cloudpan/apiutil"
	libcrypto "github.com/tickstep/library-go/crypto"
	"github.com/tickstep/library-go/requester"
)

type appLoginProbeParams struct {
	CaptchaToken string
	Lt           string
	ReturnUrl    string
	ParamId      string
	ReqId        string
	JRsaKey      string
}

type appLoginProbeResult struct {
	Result int    `json:"result"`
	Msg    string `json:"msg"`
	ToUrl  string `json:"toUrl"`
}

func appGetLoginParamsProbe() (*appLoginProbeParams, error) {
	client := requester.NewHTTPClient()
	fullURL := fmt.Sprintf("https://cloud.189.cn/unifyLoginForPC.action?appId=%s&clientType=%s&returnURL=%s&timeStamp=%d",
		"8025431004", "10020", "https://m.cloud.189.cn/zhuanti/2020/loginErrorPc/index.html", apiutil.Timestamp())
	body, err := client.Fetch("GET", fullURL, nil, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return nil, err
	}
	content := string(body)
	pick := func(pattern string) string {
		re := regexp.MustCompile(pattern)
		m := re.FindStringSubmatch(content)
		if len(m) > 1 {
			return m[1]
		}
		return ""
	}
	params := &appLoginProbeParams{
		CaptchaToken: pick("captchaToken' value='(.+?)'"),
		Lt:           pick(`lt = "(.+?)"`),
		ReturnUrl:    pick(`returnUrl = '(.+?)'`),
		ParamId:      pick(`paramId = "(.+?)"`),
		ReqId:        pick(`reqId = "(.+?)"`),
		JRsaKey:      pick(`j_rsaKey\" value=\"(.+?)\"`),
	}
	if params.CaptchaToken == "" || params.Lt == "" || params.ReturnUrl == "" || params.ParamId == "" || params.ReqId == "" || params.JRsaKey == "" {
		return nil, errors.New("解析登录参数失败")
	}
	return params, nil
}

func probeAppLogin(username, password string) (string, error) {
	params, err := appGetLoginParamsProbe()
	if err != nil {
		return "", errors.Wrap(err, "获取登录参数失败")
	}
	rsaKey := fmt.Sprintf("-----BEGIN PUBLIC KEY-----\n%s\n-----END PUBLIC KEY-----", params.JRsaKey)
	rsaUserName, _ := libcrypto.RsaEncrypt([]byte(rsaKey), []byte(username))
	rsaPassword, _ := libcrypto.RsaEncrypt([]byte(rsaKey), []byte(password))
	formData := map[string]string{
		"appKey":       "8025431004",
		"accountType":  "02",
		"userName":     "{RSA}" + apiutil.B64toHex(string(libcrypto.Base64Encode(rsaUserName))),
		"password":     "{RSA}" + apiutil.B64toHex(string(libcrypto.Base64Encode(rsaPassword))),
		"validateCode": "",
		"captchaToken": params.CaptchaToken,
		"returnUrl":    params.ReturnUrl,
		"mailSuffix":   "@189.cn",
		"dynamicCheck": "FALSE",
		"clientType":   "10020",
		"cb_SaveName":  "1",
		"isOauth2":     "false",
		"state":        "",
		"paramId":      params.ParamId,
	}
	headers := map[string]string{
		"Content-Type":     "application/x-www-form-urlencoded",
		"Referer":          "https://open.e.189.cn/api/logbox/oauth2/unifyAccountLogin.do",
		"Cookie":           "LT=" + params.Lt,
		"X-Requested-With": "XMLHttpRequest",
		"REQID":            params.ReqId,
		"lt":               params.Lt,
	}
	client := requester.NewHTTPClient()
	body, err := client.Fetch("POST", "https://open.e.189.cn/api/logbox/oauth2/loginSubmit.do", formData, headers)
	if err != nil {
		return "", errors.Wrap(err, "提交登录请求失败")
	}
	raw := strings.TrimSpace(string(body))
	resp := &appLoginProbeResult{}
	if jsonErr := json.Unmarshal(body, resp); jsonErr != nil {
		return raw, errors.Wrap(jsonErr, "解析登录响应失败")
	}
	if resp.Result != 0 || strings.TrimSpace(resp.ToUrl) == "" {
		return raw, fmt.Errorf("登录被拒 result=%d msg=%s toUrl=%s", resp.Result, resp.Msg, resp.ToUrl)
	}
	return raw, nil
}
