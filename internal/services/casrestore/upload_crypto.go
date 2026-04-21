package casrestore

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tickstep/cloudpan189-api/cloudpan/apiutil"
	cpcrypto "github.com/tickstep/library-go/crypto"
	"github.com/tickstep/library-go/requester"
	"github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
)

const (
	uploadAPIBase   = "https://upload.cloud.189.cn"
	cloudWebAPIBase = "https://cloud.189.cn"
	defaultUploadUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36"
	casSliceSize    = int64(10 * 1024 * 1024)
	maxCommitRetry  = 3
	rsaKeyTTL       = 5 * time.Minute
	uploadAppKey    = "600100422"
)

type rsaKeyInfo struct {
	PubKey string `json:"pubKey"`
	PkID   string `json:"pkId"`
	Ver    string `json:"ver"`
	Expire int64  `json:"expire"`
}

type uploadResponse struct {
	Code      string                 `json:"code"`
	Msg       string                 `json:"msg"`
	ErrorCode string                 `json:"errorCode"`
	ErrorMsg  string                 `json:"errorMsg"`
	Data      map[string]interface{} `json:"data"`
	File      map[string]interface{} `json:"file"`
}

func calcCasSliceSize(fileSize int64) int64 {
	if fileSize > casSliceSize*2*999 {
		mult := fileSize / 1999 / casSliceSize
		if fileSize%(1999*casSliceSize) != 0 {
			mult++
		}
		if mult < 5 {
			mult = 5
		}
		return mult * casSliceSize
	}
	if fileSize > casSliceSize*999 {
		return casSliceSize * 2
	}
	return casSliceSize
}

func uploadRequest(session *appsession.Session, requestURI string, params map[string]string) (*uploadResponse, error) {
	if session == nil {
		return nil, fmt.Errorf("AppSession不能为空")
	}
	rsaKey, err := getUploadRSAKey(session.Token.SessionKey)
	if err != nil {
		return nil, err
	}
	urlStr, headers, err := buildUploadRequest(params, requestURI, rsaKey, session.Token.SessionKey, "GET")
	if err != nil {
		return nil, err
	}
	body, err := requester.Fetch(http.MethodGet, urlStr, nil, headers)
	if err != nil {
		return nil, err
	}
	resp := new(uploadResponse)
	if err := json.Unmarshal(body, resp); err != nil {
		return nil, err
	}
	if resp.Code != "" && resp.Code != "SUCCESS" {
		return nil, fmt.Errorf("CAS上传请求失败 %s: %s", requestURI, firstNonEmpty(resp.Msg, resp.Code))
	}
	if resp.ErrorCode != "" {
		return nil, fmt.Errorf("CAS上传请求失败 %s: %s", requestURI, firstNonEmpty(resp.ErrorMsg, resp.ErrorCode))
	}
	return resp, nil
}

func getUploadRSAKey(sessionKey string) (*rsaKeyInfo, error) {
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	signParams := map[string]string{"AppKey": uploadAppKey, "Timestamp": ts}
	sig := apiutil.SignatureOfMd5(signParams)
	noCache := fmt.Sprintf("0.%d", rand.Int63())
	urlStr := fmt.Sprintf("%s/api/security/generateRsaKey.action?sessionKey=%s&noCache=%s", cloudWebAPIBase, url.QueryEscape(sessionKey), noCache)
	body, err := requester.Fetch(http.MethodGet, urlStr, nil, map[string]string{
		"Sign-Type":  "1",
		"Signature":  sig,
		"Timestamp":  ts,
		"AppKey":     uploadAppKey,
		"SessionKey": sessionKey,
		"Accept":     "application/json;charset=UTF-8",
		"User-Agent": defaultUploadUA,
	})
	if err != nil {
		return nil, err
	}
	resp := new(rsaKeyInfo)
	if err := json.Unmarshal(body, resp); err != nil {
		return nil, err
	}
	if resp.PubKey == "" {
		return nil, fmt.Errorf("RSA 密钥无效")
	}
	if resp.Expire == 0 {
		resp.Expire = time.Now().Add(rsaKeyTTL).UnixMilli()
	}
	return resp, nil
}

func buildUploadRequest(params map[string]string, requestURI string, rsaKey *rsaKeyInfo, sessionKey, method string) (string, map[string]string, error) {
	l := randomString(16 + rand.Intn(17))
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	uuid := apiutil.Uuid()
	encryptedParams, err := aesEncryptUpperHex(params, l)
	if err != nil {
		return "", nil, err
	}
	encryptionText, err := rsaEncryptBase64(rsaKey.PubKey, l)
	if err != nil {
		return "", nil, err
	}
	signText := fmt.Sprintf("SessionKey=%s&Operate=%s&RequestURI=%s&Date=%s&params=%s", sessionKey, method, requestURI, ts, encryptedParams)
	signature := strings.ToUpper(hex.EncodeToString(cpcrypto.HmacSHA1([]byte(l), []byte(signText))))
	return uploadAPIBase + requestURI + "?params=" + url.QueryEscape(encryptedParams), map[string]string{
		"Accept":         "application/json;charset=UTF-8",
		"SessionKey":     sessionKey,
		"Signature":      signature,
		"X-Request-Date": ts,
		"X-Request-ID":   uuid,
		"EncryptionText": encryptionText,
		"PkId":           rsaKey.PkID,
		"User-Agent":     defaultUploadUA,
	}, nil
}

func aesEncryptUpperHex(params map[string]string, key string) (string, error) {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	items := make([]string, 0, len(keys))
	for _, k := range keys {
		items = append(items, k+"="+params[k])
	}
	var aesKey [16]byte
	copy(aesKey[:], []byte(key[:16]))
	cipherText, err := cpcrypto.Aes128ECBEncrypt(aesKey, []byte(strings.Join(items, "&")))
	if err != nil {
		return "", err
	}
	return strings.ToUpper(hex.EncodeToString(cipherText)), nil
}

func rsaEncryptBase64(publicKey, raw string) (string, error) {
	formatted := publicKey
	if !strings.Contains(formatted, "BEGIN PUBLIC KEY") {
		formatted = "-----BEGIN PUBLIC KEY-----\n" + formatted + "\n-----END PUBLIC KEY-----"
	}
	enc, err := cpcrypto.RsaEncrypt([]byte(formatted), []byte(raw))
	if err != nil {
		return "", err
	}
	return string(cpcrypto.Base64Encode(enc)), nil
}

func randomString(length int) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func uploadRespDataString(resp *uploadResponse, path ...string) string {
	var cur interface{}
	if len(path) == 0 {
		return ""
	}
	switch path[0] {
	case "data":
		cur = resp.Data
	case "file":
		cur = resp.File
	default:
		return ""
	}
	for _, key := range path[1:] {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return ""
		}
		cur = m[key]
	}
	switch v := cur.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return ""
	}
}

func uploadRespDataBoolInt(resp *uploadResponse, path ...string) bool {
	var cur interface{}
	if len(path) == 0 {
		return false
	}
	switch path[0] {
	case "data":
		cur = resp.Data
	case "file":
		cur = resp.File
	default:
		return false
	}
	for _, key := range path[1:] {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return false
		}
		cur = m[key]
	}
	switch v := cur.(type) {
	case float64:
		return int(v) == 1
	case int:
		return v == 1
	case int64:
		return v == 1
	case string:
		return v == "1"
	default:
		return false
	}
}
