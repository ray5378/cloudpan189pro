package casrestore

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
)

func BuildRefSDKAccessToken(session *appsession.Session, cloudToken *models.CloudToken) (string, error) {
	if session == nil {
		return "", fmt.Errorf("session为空")
	}
	if cloudToken == nil {
		return "", fmt.Errorf("cloudToken为空")
	}
	refClient := newRefSDKClient()
	sessionKey, _ := cloudToken.Addition[models.CloudTokenAdditionSessionKey].(string)
	sessionSecret, _ := cloudToken.Addition[models.CloudTokenAdditionSessionSecret].(string)
	familySessionKey, _ := cloudToken.Addition[models.CloudTokenAdditionFamilySessionKey].(string)
	familySessionSecret, _ := cloudToken.Addition[models.CloudTokenAdditionFamilySessionSecret].(string)
	pcAccessToken, _ := cloudToken.Addition[models.CloudTokenAdditionAppAccessToken].(string)
	if strings.TrimSpace(sessionKey) != "" && strings.TrimSpace(sessionSecret) != "" && strings.TrimSpace(familySessionKey) != "" && strings.TrimSpace(familySessionSecret) != "" {
		_, accessToken, err := refClient.buildSessionFromStoredTokens(session, sessionKey, sessionSecret, familySessionKey, familySessionSecret, "")
		return accessToken, err
	}
	if strings.TrimSpace(pcAccessToken) == "" {
		pcAccessToken = strings.TrimSpace(cloudToken.AccessToken)
	}
	if strings.TrimSpace(pcAccessToken) == "" {
		return "", fmt.Errorf("缺少可用access token，无法构建参考SDK会话")
	}
	_, accessToken, err := refClient.buildSessionFromAppAccessToken(session, pcAccessToken)
	return accessToken, err
}

func SafeDeleteFamilyFileByAccessToken(accessToken, familyID, fileID, fileName string) error {
	return SafeDeleteFamilyNodeByAccessToken(accessToken, familyID, fileID, fileName, false)
}

func SafeDeleteFamilyNodeByAccessToken(accessToken, familyID, fileID, fileName string, isFolder bool) error {
	fid, err := strconv.ParseInt(familyID, 10, 64)
	if err != nil {
		return err
	}
	if err := newRefSDKClient().safeDeleteFamilyNode(accessToken, fid, fileID, fileName, isFolder); err != nil {
		if strings.Contains(err.Error(), "批量任务超时 type=CLEAR_RECYCLE") {
			return nil
		}
		return err
	}
	return nil
}

func ClearFamilyRecycleByAccessToken(accessToken, familyID string) error {
	fid, err := strconv.ParseInt(familyID, 10, 64)
	if err != nil {
		return err
	}
	clearResp := new(refSDKBatchCreateResp)
	client := newRefSDKClient()
	if err := client.doBatchFormRequest(accessToken, familyBatchAPIBase+"/open/batch/createBatchTask.action", map[string]string{
		"type":           "CLEAR_RECYCLE",
		"taskInfos":      "[]",
		"targetFolderId": "",
		"familyId":       strconv.FormatInt(fid, 10),
	}, 30*time.Second, clearResp); err != nil {
		return err
	}
	if batchRespError(clearResp.ResCode, clearResp.ResMessage) {
		return fmt.Errorf("CLEAR_RECYCLE失败: %s", clearResp.ResMessage)
	}
	if clearResp.TaskID != "" {
		if err := client.waitForBatchTask(accessToken, "CLEAR_RECYCLE", clearResp.TaskID, 2*time.Minute); err != nil {
			if strings.Contains(err.Error(), "批量任务超时 type=CLEAR_RECYCLE") {
				return nil
			}
			return err
		}
	}
	return nil
}
