package casrestore

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
)

const (
	refSDKPCAppID    = "8025431004"
	refSDKOpenAppKey = "601102120"
	refSDKChannelID  = "web_cloud.189.cn"
	refSDKClientType = "TELEMAC"
	refSDKClientVer  = "1.0.0"
	refSDKUserAgent  = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36"
)

type refSDKClient struct {
	httpClient *http.Client
}

type refSDKSessionForPCResp struct {
	XMLName             xml.Name `xml:"userSession"`
	LoginName           string   `xml:"loginName"`
	SessionKey          string   `xml:"sessionKey"`
	SessionSecret       string   `xml:"sessionSecret"`
	FamilySessionKey    string   `xml:"familySessionKey"`
	FamilySessionSecret string   `xml:"familySessionSecret"`
}

type refSDKSskTokenResp struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int64  `json:"expiresIn"`
}

type refSDKBatchCreateResp struct {
	ResCode    any    `json:"res_code"`
	ResMessage string `json:"res_message"`
	TaskID     string `json:"taskId"`
}

type refSDKBatchCheckResp struct {
	ResCode        any    `json:"res_code"`
	ResMessage     string `json:"res_message"`
	TaskStatus     int    `json:"taskStatus"`
	TaskID         string `json:"taskId"`
	FailedCount    int    `json:"failedCount"`
	SuccessedCount int    `json:"successedCount"`
	SkipCount      int    `json:"skipCount"`
	ErrorCode      string `json:"errorCode"`
}

func newRefSDKClient() *refSDKClient {
	return &refSDKClient{httpClient: &http.Client{Timeout: 30 * time.Second}}
}

func (c *refSDKClient) refreshPCSessionByAppAccessToken(appAccessToken string) (*refSDKSessionForPCResp, error) {
	appAccessToken = strings.TrimSpace(appAccessToken)
	if appAccessToken == "" {
		return nil, errors.New("app access token为空")
	}
	targetURL := fmt.Sprintf("%s/getSessionForPC.action?appId=%s&accessToken=%s&clientSn=%d&clientType=%s&version=%s&channelId=%s", familyBatchAPIBase, refSDKPCAppID, url.QueryEscape(appAccessToken), time.Now().UnixNano(), refSDKClientType, refSDKClientVer, refSDKChannelID)
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Request-ID", "cloudpan189pro-refsdk")
	req.Header.Set("User-Agent", refSDKUserAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	out := new(refSDKSessionForPCResp)
	if err := xml.Unmarshal(body, out); err != nil {
		return nil, fmt.Errorf("刷新PC session失败: url=%s body=%s parseErr=%w", targetURL, string(body), err)
	}
	if strings.TrimSpace(out.SessionKey) == "" {
		return nil, fmt.Errorf("刷新PC session失败: url=%s body=%s", targetURL, string(body))
	}
	return out, nil
}

func (c *refSDKClient) getSskAccessTokenBySessionKey(sessionKey string) (*refSDKSskTokenResp, error) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return nil, errors.New("sessionKey为空")
	}
	targetURL := familyBatchAPIBase + "/open/oauth2/getAccessTokenBySsKey.action?sessionKey=" + url.QueryEscape(sessionKey)
	timestamp, signature := refSDKBuildOpenSignature(targetURL, nil)
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Sign-Type", "1")
	req.Header.Set("Signature", signature)
	req.Header.Set("Timestamp", timestamp)
	req.Header.Set("AppKey", refSDKOpenAppKey)
	req.Header.Set("User-Agent", refSDKUserAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}
	out := new(refSDKSskTokenResp)
	if err := json.Unmarshal(body, out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return nil, fmt.Errorf("SSK access token为空: %s", string(body))
	}
	return out, nil
}

func (c *refSDKClient) buildSessionFromAppAccessToken(session *appsession.Session, appAccessToken string) (*appsession.Session, string, error) {
	pcSession, err := c.refreshPCSessionByAppAccessToken(appAccessToken)
	if err != nil {
		return nil, "", err
	}
	ssk, err := c.getSskAccessTokenBySessionKey(pcSession.SessionKey)
	if err != nil {
		return nil, "", err
	}
	out := &appsession.Session{Token: session.Token}
	out.Token.SessionKey = pcSession.SessionKey
	out.Token.SessionSecret = pcSession.SessionSecret
	out.Token.FamilySessionKey = pcSession.FamilySessionKey
	out.Token.FamilySessionSecret = pcSession.FamilySessionSecret
	out.Token.AccessToken = ssk.AccessToken
	return out, ssk.AccessToken, nil
}

func (c *refSDKClient) buildSessionFromStoredTokens(session *appsession.Session, sessionKey, sessionSecret, familySessionKey, familySessionSecret, sskAccessToken string) (*appsession.Session, string, error) {
	if strings.TrimSpace(sessionKey) == "" || strings.TrimSpace(sessionSecret) == "" || strings.TrimSpace(familySessionKey) == "" || strings.TrimSpace(familySessionSecret) == "" {
		return nil, "", errors.New("存量session token不完整")
	}
	sskAccessToken = strings.TrimSpace(sskAccessToken)
	if sskAccessToken == "" {
		ssk, err := c.getSskAccessTokenBySessionKey(sessionKey)
		if err != nil {
			return nil, "", err
		}
		sskAccessToken = ssk.AccessToken
	}
	out := &appsession.Session{Token: session.Token}
	out.Token.SessionKey = strings.TrimSpace(sessionKey)
	out.Token.SessionSecret = strings.TrimSpace(sessionSecret)
	out.Token.FamilySessionKey = strings.TrimSpace(familySessionKey)
	out.Token.FamilySessionSecret = strings.TrimSpace(familySessionSecret)
	out.Token.AccessToken = sskAccessToken
	return out, sskAccessToken, nil
}

func refSDKBuildOpenSignature(targetURL string, params map[string]string) (string, string) {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	payload := map[string]string{
		"AppKey":    refSDKOpenAppKey,
		"Timestamp": timestamp,
	}
	if parsed, err := url.Parse(targetURL); err == nil {
		for key, values := range parsed.Query() {
			if len(values) > 0 {
				payload[key] = values[0]
			}
		}
	}
	for k, v := range params {
		payload[k] = v
	}
	return timestamp, refSDKSortedMD5(payload)
}

func refSDKBuildBatchSignature(accessToken string, params map[string]string) (string, string) {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	payload := map[string]string{
		"AccessToken": accessToken,
		"Timestamp":   timestamp,
	}
	for k, v := range params {
		payload[k] = v
	}
	return timestamp, refSDKSortedMD5(payload)
}

func refSDKSortedMD5(payload map[string]string) string {
	keys := make([]string, 0, len(payload))
	for k := range payload {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+payload[k])
	}
	sum := md5.Sum([]byte(strings.Join(parts, "&")))
	return hex.EncodeToString(sum[:])
}
