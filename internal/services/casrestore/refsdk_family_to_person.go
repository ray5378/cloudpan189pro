package casrestore

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/tickstep/cloudpan189-api/cloudpan"
	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
	"github.com/xxcheng123/cloudpan189-share/internal/services/casparser"
	cloudtokenSvi "github.com/xxcheng123/cloudpan189-share/internal/services/cloudtoken"
)

func (s *service) tryRestoreFamilyToPersonByRefSDK(ctx appctx.Context, req RestoreRequest, session *appsession.Session, panClient *cloudpan.PanClient, restoreName string, info *casparser.CasInfo) (*familyRestoreResult, error) {
	if req.UploadRoute != UploadRouteFamily || req.DestinationType != DestinationTypePerson {
		return nil, nil
	}
	cloudToken, err := s.loadRestoreCloudToken(ctx, req)
	if err != nil {
		return nil, err
	}
	sessionKey, _ := cloudToken.Addition[models.CloudTokenAdditionSessionKey].(string)
	sessionSecret, _ := cloudToken.Addition[models.CloudTokenAdditionSessionSecret].(string)
	familySessionKey, _ := cloudToken.Addition[models.CloudTokenAdditionFamilySessionKey].(string)
	familySessionSecret, _ := cloudToken.Addition[models.CloudTokenAdditionFamilySessionSecret].(string)
	storedSSKAccessToken, _ := cloudToken.Addition[models.CloudTokenAdditionSskAccessToken].(string)
	pcAccessToken := strings.TrimSpace(cloudToken.AccessToken)

	legacy := &familyRestoreAdapter{}
	familyID := reqFamilyIDFromContext(session)
	if familyID <= 0 {
		familyID, err = legacy.pickFamilyID(panClient)
		if err != nil {
			return nil, err
		}
	}
	familyFolderID, err := legacy.getFamilyRootFolderID(session, familyID)
	if err != nil {
		return nil, err
	}
	familyFileID, err := legacy.familyRapidUpload(session, familyID, familyFolderID, info, restoreName)
	if err != nil {
		return nil, err
	}

	refClient := newRefSDKClient()
	var refSession *appsession.Session
	var accessToken string
	if strings.TrimSpace(sessionKey) != "" && strings.TrimSpace(sessionSecret) != "" && strings.TrimSpace(familySessionKey) != "" && strings.TrimSpace(familySessionSecret) != "" {
		refSession, accessToken, err = refClient.buildSessionFromStoredTokens(session, sessionKey, sessionSecret, familySessionKey, familySessionSecret, storedSSKAccessToken)
	} else {
		if strings.TrimSpace(pcAccessToken) == "" {
			return nil, fmt.Errorf("缺少可用access token，无法构建参考SDK会话")
		}
		refSession, accessToken, err = refClient.buildSessionFromAppAccessToken(session, pcAccessToken)
	}
	if err != nil {
		return nil, errors.Wrap(err, "构建参考SDK会话失败")
	}
	_ = refSession
	if err := refClient.copyFamilyFileToPersonal(accessToken, familyID, familyFileID, normalizePersonFolderID(req.TargetFolderID), restoreName); err != nil {
		return nil, errors.Wrap(err, "参考SDK COPY失败")
	}
	if err := refClient.safeDeleteFamilyFile(accessToken, familyID, familyFileID, restoreName); err != nil {
		return nil, errors.Wrap(err, "参考SDK清理家庭中转文件失败")
	}
	return &familyRestoreResult{
		FamilyID:         familyID,
		RestoredFileID:   familyFileID,
		RestoredFileName: restoreName,
	}, nil
}

func (s *service) loadRestoreCloudToken(ctx appctx.Context, req RestoreRequest) (*models.CloudToken, error) {
	cloudTokenSvc := cloudtokenSvi.NewService(s.svc)
	tokenID := req.TargetTokenID
	if tokenID <= 0 && req.MountPointID > 0 {
		var err error
		tokenID, err = s.inferTargetTokenID(ctx, req.MountPointID)
		if err != nil {
			return nil, err
		}
	}
	if tokenID <= 0 {
		return nil, fmt.Errorf("无法确定云盘令牌ID")
	}
	return cloudTokenSvc.Query(ctx, tokenID)
}
