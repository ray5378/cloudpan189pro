package casrestore

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tickstep/cloudpan189-api/cloudpan/apiutil"
	cpcrypto "github.com/tickstep/library-go/crypto"
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

var uploadRSAKeyCache = struct {
	sync.Mutex
	items map[string]*rsaKeyInfo
}{items: map[string]*rsaKeyInfo{}}

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

type blacklistedError struct {
	URI string
}

func (e blacklistedError) Error() string {
	return fmt.Sprintf("CAS秒传被天翼云盘风控拦截(文件MD5黑名单): %s", e.URI)
}

func (e blacklistedError) IsBlacklisted() bool {
	return true
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

func getSessionKeyForUpload(session *appsession.Session) (string, error) {
	if session == nil {
		return "", fmt.Errorf("AppSession不能为空")
	}
	sessionKey := strings.TrimSpace(session.Token.SessionKey)
	if sessionKey == "" {
		return "", fmt.Errorf("获取上传SessionKey失败")
	}
	return sessionKey, nil
}

// uploadRequest 是 upload.cloud.189.cn 参考链的固定入口。
// 注意：这里已经按参考实现对齐了 sessionKey 获取、RSA key cache、AES/RSA/HMAC、黑名单识别、403 时清 RSA cache。
// 这些行为属于“已对齐项”，不要再改回看起来等价的 SDK/简化实现。
func uploadRequest(session *appsession.Session, requestURI string, params map[string]string) (*uploadResponse, error) {
	if session == nil {
		return nil, fmt.Errorf("AppSession不能为空")
	}
	sessionKey, err := getSessionKeyForUpload(session)
	if err != nil {
		return nil, err
	}
	rsaKey, err := getUploadRSAKeyWithCache(session, sessionKey)
	if err != nil {
		return nil, err
	}
	return doUploadRequest(session, sessionKey, requestURI, params, rsaKey)
}

func doUploadRequest(session *appsession.Session, sessionKey, requestURI string, params map[string]string, rsaKey *rsaKeyInfo) (*uploadResponse, error) {
	urlStr, headers, err := buildUploadRequest(params, requestURI, rsaKey, sessionKey, http.MethodGet)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	parsed := new(uploadResponse)
	if len(body) > 0 {
		if err := json.Unmarshal(body, parsed); err != nil {
			if isBlacklistBody(string(body)) {
				return nil, blacklistedError{URI: requestURI}
			}
			if resp.StatusCode >= 400 {
				return nil, httpError{StatusCode: resp.StatusCode, Body: string(body)}
			}
			return nil, err
		}
	}
	if resp.StatusCode >= 400 {
		if resp.StatusCode == http.StatusForbidden {
			clearUploadRSAKeyCache(session)
		}
		if isBlacklistResp(parsed) {
			return nil, blacklistedError{URI: requestURI}
		}
		if parsed != nil && (parsed.Code != "" || parsed.Msg != "" || parsed.ErrorCode != "" || parsed.ErrorMsg != "") {
			return nil, fmt.Errorf("CAS上传请求失败 %s: %s", requestURI, strings.TrimSpace(strings.TrimSpace(parsed.Code+" "+parsed.Msg+" "+parsed.ErrorCode+" "+parsed.ErrorMsg)))
		}
		return nil, httpError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	if isBlacklistResp(parsed) {
		return nil, blacklistedError{URI: requestURI}
	}
	if parsed.Code != "" && parsed.Code != "SUCCESS" {
		return nil, fmt.Errorf("CAS上传请求失败 %s: %s", requestURI, firstNonEmpty(parsed.Msg, parsed.Code))
	}
	if parsed.ErrorCode != "" {
		return nil, fmt.Errorf("CAS上传请求失败 %s: %s", requestURI, firstNonEmpty(parsed.ErrorMsg, parsed.ErrorCode))
	}
	return parsed, nil
}

func getUploadRSAKeyWithCache(session *appsession.Session, sessionKey string) (*rsaKeyInfo, error) {
	key := uploadAccountKey(session)
	now := time.Now().UnixMilli()
	uploadRSAKeyCache.Lock()
	cached := uploadRSAKeyCache.items[key]
	if cached != nil && cached.Expire > now {
		copyKey := *cached
		uploadRSAKeyCache.Unlock()
		return &copyKey, nil
	}
	uploadRSAKeyCache.Unlock()

	rsaKey, err := getUploadRSAKey(sessionKey)
	if err != nil {
		return nil, err
	}
	maxExpire := time.Now().Add(rsaKeyTTL).UnixMilli()
	if rsaKey.Expire == 0 || rsaKey.Expire > maxExpire {
		rsaKey.Expire = maxExpire
	}
	uploadRSAKeyCache.Lock()
	uploadRSAKeyCache.items[key] = rsaKey
	uploadRSAKeyCache.Unlock()
	return rsaKey, nil
}

func clearUploadRSAKeyCache(session *appsession.Session) {
	key := uploadAccountKey(session)
	uploadRSAKeyCache.Lock()
	delete(uploadRSAKeyCache.items, key)
	uploadRSAKeyCache.Unlock()
}

func uploadAccountKey(session *appsession.Session) string {
	if session == nil {
		return "default"
	}
	if token := strings.TrimSpace(session.Token.AccessToken); token != "" {
		return token
	}
	if key := strings.TrimSpace(session.Token.SessionKey); key != "" {
		return key
	}
	return "default"
}

func getUploadRSAKey(sessionKey string) (*rsaKeyInfo, error) {
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	signParams := map[string]string{"AppKey": uploadAppKey, "Timestamp": ts}
	sig := apiutil.SignatureOfMd5(signParams)
	noCache := fmt.Sprintf("0.%d", rand.Int63())
	urlStr := fmt.Sprintf("%s/api/security/generateRsaKey.action?sessionKey=%s&noCache=%s", cloudWebAPIBase, url.QueryEscape(sessionKey), noCache)
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Sign-Type", "1")
	req.Header.Set("Signature", sig)
	req.Header.Set("Timestamp", ts)
	req.Header.Set("AppKey", uploadAppKey)
	req.Header.Set("SessionKey", sessionKey)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("User-Agent", defaultUploadUA)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	parsed := new(rsaKeyInfo)
	if err := json.Unmarshal(body, parsed); err != nil {
		return nil, err
	}
	if parsed.PubKey == "" {
		return nil, fmt.Errorf("RSA 密钥无效")
	}
	if parsed.Expire == 0 {
		parsed.Expire = time.Now().Add(rsaKeyTTL).UnixMilli()
	} else if parsed.Expire < 1e12 {
		parsed.Expire = time.Now().Add(time.Duration(parsed.Expire) * time.Second).UnixMilli()
	}
	return parsed, nil
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
	return uploadAPIBase + requestURI + "?params=" + encryptedParams, map[string]string{
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
	items := make([]string, 0, len(params))
	for k, v := range params {
		items = append(items, k+"="+v)
	}
	joined := strings.Join(items, "&")
	var aesKey [16]byte
	copy(aesKey[:], []byte(key[:16]))
	cipherText, err := cpcrypto.Aes128ECBEncrypt(aesKey, []byte(joined))
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

type httpError struct {
	StatusCode int
	Body       string
}

func (e httpError) Error() string {
	return fmt.Sprintf("http %d: %s", e.StatusCode, e.Body)
}

func isBlacklistResp(resp *uploadResponse) bool {
	if resp == nil {
		return false
	}
	if resp.Code == "InfoSecurityErrorCode" || resp.ErrorCode == "InfoSecurityErrorCode" {
		return true
	}
	msg := strings.ToLower(firstNonEmpty(resp.Msg, resp.ErrorMsg))
	return strings.Contains(msg, "black list")
}

func isBlacklistBody(body string) bool {
	text := strings.ToLower(body)
	return strings.Contains(text, "black list") || strings.Contains(text, "infosecurityerrorcode")
}
