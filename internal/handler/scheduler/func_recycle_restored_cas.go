package scheduler

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	appsessionSvi "github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
	casrecordSvi "github.com/xxcheng123/cloudpan189-share/internal/services/casrecord"
	cloudtokenSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudtoken"
	mountpointSvi "github.com/xxcheng123/cloudpan189-share/internal/services/mountpoint"
	"go.uber.org/zap"
)

type RecycleRestoredCASScheduler struct {
	casRecordService  casrecordSvi.Service
	appSessionService appsessionSvi.Service
	mountPointService mountpointSvi.Service
	cloudTokenService cloudtokenSvi.Service
	quit              chan struct{}
	done              chan struct{}
	running           bool
}

func NewRecycleRestoredCASScheduler(casRecordService casrecordSvi.Service, appSessionService appsessionSvi.Service, mountPointService mountpointSvi.Service, cloudTokenService cloudtokenSvi.Service) Scheduler {
	return &RecycleRestoredCASScheduler{
		casRecordService:  casRecordService,
		appSessionService: appSessionService,
		mountPointService: mountPointService,
		cloudTokenService: cloudTokenService,
		quit:              make(chan struct{}),
		done:              make(chan struct{}),
	}
}

func (s *RecycleRestoredCASScheduler) Start(ctx appctx.Context) error {
	if s.running {
		return ErrSchedulerRunning
	}
	s.running = true
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		s.doJob(ctx)
		for {
			select {
			case <-ticker.C:
				s.doJob(ctx)
			case <-s.quit:
				return
			}
		}
	}()
	return nil
}

func (s *RecycleRestoredCASScheduler) Stop() {
	if !s.running {
		return
	}
	close(s.quit)
	<-s.done
	s.running = false
}

func (s *RecycleRestoredCASScheduler) doJob(ctx appctx.Context) {
	list, err := s.casRecordService.ListDueRecycle(ctx, time.Now(), 50)
	if err != nil {
		ctx.Error("查询到期CAS恢复文件失败")
		return
	}
	for _, record := range list {
		if record == nil || strings.TrimSpace(record.RestoredFileID) == "" {
			continue
		}
		_ = s.casRecordService.Update(ctx, record.ID, map[string]any{"restore_status": models.CasRestoreStatusRecycling})
		if err := s.recycleOne(ctx, record); err != nil {
			_ = s.casRecordService.Update(ctx, record.ID, map[string]any{
				"restore_status": models.CasRestoreStatusFailed,
				"last_error":     err.Error(),
			})
			ctx.Error("回收到期CAS恢复文件失败", zap.Int64("record_id", record.ID), zap.Error(err))
			continue
		}
		now := time.Now()
		_ = s.casRecordService.Update(ctx, record.ID, map[string]any{
			"restore_status":   models.CasRestoreStatusRecycled,
			"restored_file_id": "",
			"recycle_after_at": nil,
			"last_access_at":   &now,
			"last_error":       "",
		})
	}
}

func (s *RecycleRestoredCASScheduler) recycleOne(ctx appctx.Context, record *models.CasMediaRecord) error {
	targetTokenID := record.TargetTokenID
	if targetTokenID <= 0 {
		return fmt.Errorf("缺少回收目标token")
	}
	session, err := s.appSessionService.GetByTokenID(ctx, targetTokenID)
	if err != nil {
		return err
	}
	accessToken := strings.TrimSpace(session.Token.AccessToken)
	if accessToken == "" {
		return fmt.Errorf("无法获取AccessToken")
	}
	fileName := strings.TrimSpace(record.RestoredFileName)
	if fileName == "" {
		fileName = strings.TrimSpace(record.OriginalFileName)
	}
	if fileName == "" {
		fileName = record.RestoredFileID
	}
	if strings.EqualFold(strings.TrimSpace(record.DestinationType), "family") {
		familyID := strings.TrimSpace(record.TargetFamilyID)
		if familyID == "" {
			return fmt.Errorf("缺少家庭回收目标familyId")
		}
		deleteResp := new(batchTaskCreateResp)
		if err := doAccessTokenFormJSONRequest(accessToken, familyBatchAPIBase+"/open/batch/createBatchTask.action", map[string]string{
			"type":           "DELETE",
			"taskInfos":      fmt.Sprintf(`[{"fileId":"%s","fileName":"%s","isFolder":0}]`, record.RestoredFileID, escapeJSONString(fileName)),
			"targetFolderId": "",
			"familyId":       familyID,
		}, 30*time.Second, deleteResp); err != nil {
			return errors.Wrap(err, "提交DELETE任务失败")
		}
		if batchRespError(deleteResp.ResCode, deleteResp.ResMessage) {
			return fmt.Errorf("提交DELETE任务失败: %s", deleteResp.ResMessage)
		}
		if err := waitForRecycleBatchTask(accessToken, "DELETE", deleteResp.TaskID, 2*time.Minute); err != nil {
			return err
		}

		clearResp := new(batchTaskCreateResp)
		if err := doAccessTokenFormJSONRequest(accessToken, familyBatchAPIBase+"/open/batch/createBatchTask.action", map[string]string{
			"type":           "CLEAR_RECYCLE",
			"taskInfos":      "[]",
			"targetFolderId": "",
			"familyId":       familyID,
		}, 30*time.Second, clearResp); err != nil {
			return errors.Wrap(err, "提交CLEAR_RECYCLE任务失败")
		}
		if batchRespError(clearResp.ResCode, clearResp.ResMessage) {
			return fmt.Errorf("提交CLEAR_RECYCLE任务失败: %s", clearResp.ResMessage)
		}
		if err := waitForRecycleBatchTask(accessToken, "CLEAR_RECYCLE", clearResp.TaskID, 2*time.Minute); err != nil {
			return err
		}
		return nil
	}
	panClient, err := s.newPanClientBySession(ctx, session)
	if err != nil {
		return err
	}
	ok, apiErr := panClient.AppDeleteFile([]string{record.RestoredFileID})
	if apiErr != nil {
		return apiErr
	}
	if !ok {
		return fmt.Errorf("删除个人恢复文件失败")
	}
	if apiErr := panClient.RecycleDelete(0, []string{record.RestoredFileID}); apiErr != nil {
		return apiErr
	}
	if apiErr := panClient.RecycleClear(0); apiErr != nil {
		return apiErr
	}
	return nil
}

func waitForRecycleBatchTask(accessToken, taskType, taskID string, maxWait time.Duration) error {
	deadline := time.Now().Add(maxWait)
	lastStatus := 0
	for time.Now().Before(deadline) {
		time.Sleep(1 * time.Second)
		resp := new(batchTaskCheckResp)
		if err := doAccessTokenFormJSONRequest(accessToken, familyBatchAPIBase+"/open/batch/checkBatchTask.action", map[string]string{
			"type":   taskType,
			"taskId": taskID,
		}, 15*time.Second, resp); err != nil {
			return errors.Wrap(err, "批量任务查询失败")
		}
		if batchRespError(resp.ResCode, resp.ResMessage) {
			return fmt.Errorf("批量任务查询失败: %s", resp.ResMessage)
		}
		lastStatus = resp.TaskStatus
		if lastStatus == 4 {
			if resp.FailedCount > 0 && resp.SuccessedCount == 0 {
				return fmt.Errorf("批量任务失败 type=%s failed=%d successed=%d", taskType, resp.FailedCount, resp.SuccessedCount)
			}
			return nil
		}
	}
	return fmt.Errorf("批量任务超时 type=%s status=%d", taskType, lastStatus)
}

const familyBatchAPIBase = "https://api.cloud.189.cn"

type batchTaskCreateResp struct {
	ResCode    any    `json:"res_code"`
	ResMessage string `json:"res_message"`
	TaskID     string `json:"taskId"`
}

type batchTaskCheckResp struct {
	ResCode        any    `json:"res_code"`
	ResMessage     string `json:"res_message"`
	TaskStatus     int    `json:"taskStatus"`
	TaskID         string `json:"taskId"`
	FailedCount    int    `json:"failedCount"`
	SuccessedCount int    `json:"successedCount"`
	SkipCount      int    `json:"skipCount"`
}

func batchRespError(code any, msg string) bool {
	if code == nil {
		return false
	}
	return fmt.Sprint(code) != "0"
}

func escapeJSONString(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return replacer.Replace(s)
}

func doAccessTokenFormJSONRequest(accessToken string, targetURL string, params map[string]string, timeout time.Duration, out any) error {
	timestamp, signature := buildAccessTokenSignature(strings.TrimSpace(accessToken), params)
	req, err := http.NewRequest(http.MethodPost, targetURL, strings.NewReader(formURLEncode(params)))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Sign-Type", "1")
	req.Header.Set("Signature", signature)
	req.Header.Set("Timestamp", timestamp)
	req.Header.Set("AccessToken", strings.TrimSpace(accessToken))
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return jsonNewDecoder(resp, out)
}

func jsonNewDecoder(resp *http.Response, out any) error {
	return json.NewDecoder(resp.Body).Decode(out)
}

func buildAccessTokenSignature(accessToken string, params map[string]string) (string, string) {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	raw := accessToken + strings.Join(parts, "&") + timestamp
	sum := md5.Sum([]byte(raw))
	return timestamp, strings.ToUpper(hex.EncodeToString(sum[:]))
}

func formURLEncode(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(params[k]))
	}
	return strings.Join(parts, "&")
}
