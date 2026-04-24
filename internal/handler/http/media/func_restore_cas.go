package media

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/tickstep/cloudpan189-api/cloudpan"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	"github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
	casrestoreSvi "github.com/xxcheng123/cloudpan189-share/internal/services/casrestore"
)

type restoreCasRequest struct {
	StorageID       int64                         `json:"storageId" binding:"omitempty" example:"1"`
	MountPointID    int64                         `json:"mountPointId" binding:"omitempty" example:"1"`
	CasFileID       string                        `json:"casFileId" binding:"omitempty" example:"123456789"`
	CasFileName     string                        `json:"casFileName" binding:"omitempty" example:"movie.cas"`
	CasVirtualID    int64                         `json:"casVirtualId" binding:"omitempty" example:"1001"`
	CasPath         string                        `json:"casPath" binding:"omitempty" example:"/电影库/movie.cas"`
	UploadRoute     casrestoreSvi.UploadRoute     `json:"uploadRoute" binding:"omitempty,oneof=family person" example:"family"`
	DestinationType casrestoreSvi.DestinationType `json:"destinationType" binding:"required,oneof=family person" example:"family"`
	TargetFolderID  string                        `json:"targetFolderId" binding:"omitempty" example:"-11"`
}

func (h *handler) RestoreCas() httpcontext.HandlerFunc {
	return func(ctx *httpcontext.Context) {
		req := new(restoreCasRequest)
		if err := ctx.ShouldBindJSON(req); err != nil {
			ctx.AbortWithInvalidParams(err)
			return
		}
		if req.CasVirtualID == 0 && req.CasPath == "" {
			ctx.AbortWithInvalidParams(fmt.Errorf("casVirtualId 和 casPath 至少传一个"))
			return
		}
		if req.UploadRoute == casrestoreSvi.UploadRoutePerson && req.DestinationType == casrestoreSvi.DestinationTypeFamily {
			ctx.AbortWithInvalidParams(fmt.Errorf("不支持的操作: 当前仅支持 reference-backed 的 restore 组合，person -> family 暂未实现"))
			return
		}

		restoreReq, err := h.buildRestoreRequest(ctx, req)
		if err != nil {
			ctx.Fail(codeRestoreCasFailed.WithError(err))
			return
		}
		if restoreReq.UploadRoute == "" {
			restoreReq.UploadRoute = casrestoreSvi.UploadRouteFamily
		}

		resp, err := h.casRestoreService.EnsureRestored(ctx.GetContext(), restoreReq)
		if err != nil {
			ctx.Fail(codeRestoreCasFailed.WithError(err))
			return
		}

		ctx.Success(resp)
	}
}

func (h *handler) buildRestoreRequest(ctx *httpcontext.Context, req *restoreCasRequest) (casrestoreSvi.RestoreRequest, error) {
	restoreReq := casrestoreSvi.RestoreRequest{
		StorageID:       req.StorageID,
		MountPointID:    req.MountPointID,
		CasFileID:       req.CasFileID,
		CasFileName:     req.CasFileName,
		CasVirtualID:    req.CasVirtualID,
		UploadRoute:     req.UploadRoute,
		DestinationType: req.DestinationType,
		TargetFolderID:  req.TargetFolderID,
	}

	vf, err := h.queryCASVirtualFile(ctx, req)
	if err != nil {
		return casrestoreSvi.RestoreRequest{}, err
	}
	if vf == nil {
		return casrestoreSvi.RestoreRequest{}, fmt.Errorf("无法定位CAS虚拟文件")
	}
	if restoreReq.CasVirtualID == 0 {
		restoreReq.CasVirtualID = vf.ID
	}
	if restoreReq.CasFileID == "" {
		restoreReq.CasFileID = vf.CloudId
	}
	if restoreReq.CasFileName == "" {
		restoreReq.CasFileName = vf.Name
	}

	top, err := h.virtualfileService.QueryTop(ctx.GetContext(), vf.ID)
	if err != nil {
		return casrestoreSvi.RestoreRequest{}, err
	}
	if top == nil {
		return casrestoreSvi.RestoreRequest{}, fmt.Errorf("无法定位CAS文件所属挂载点")
	}

	mp, err := h.mountpointService.Query(ctx.GetContext(), top.ID)
	if err != nil {
		return casrestoreSvi.RestoreRequest{}, err
	}
	if mp == nil {
		return casrestoreSvi.RestoreRequest{}, fmt.Errorf("无法定位CAS文件挂载点记录")
	}

	if restoreReq.MountPointID == 0 {
		restoreReq.MountPointID = mp.FileId
	}
	if restoreReq.StorageID == 0 {
		restoreReq.StorageID = mp.FileId
	}

	if h.settingService == nil {
		return casrestoreSvi.RestoreRequest{}, fmt.Errorf("缺少系统设置服务，无法读取CAS指定恢复位置")
	}
	latest, qerr := h.settingService.Query(ctx.GetContext())
	if qerr != nil || latest == nil {
		return casrestoreSvi.RestoreRequest{}, fmt.Errorf("读取CAS指定恢复位置失败")
	}
	addition := latest.Addition

	if restoreReq.DestinationType == casrestoreSvi.DestinationTypeFamily {
		if strings.TrimSpace(restoreReq.TargetFolderID) == "" {
			return casrestoreSvi.RestoreRequest{}, fmt.Errorf("targetFolderID不能为空")
		}
		familyTargetFamilyID := addition.CasFamilyTargetFamilyId
		familyTargetTokenID := addition.CasFamilyTargetTokenId
		if familyTargetFamilyID == "" {
			return casrestoreSvi.RestoreRequest{}, fmt.Errorf("未配置家庭恢复目标(casFamilyTargetFamilyId)")
		}
		parsed, perr := strconv.ParseInt(familyTargetFamilyID, 10, 64)
		if perr != nil || parsed <= 0 {
			return casrestoreSvi.RestoreRequest{}, fmt.Errorf("家庭恢复目标无效: %s", familyTargetFamilyID)
		}
		restoreReq.FamilyID = parsed
		resolvedFolderID, ferr := h.resolveDefaultFamilyRestoreTargetFolder(ctx, vf.ID, familyTargetTokenID, parsed, restoreReq.TargetFolderID, addition.CasAutoCollectPreservePath)
		if ferr != nil {
			return casrestoreSvi.RestoreRequest{}, ferr
		}
		restoreReq.TargetFolderID = resolvedFolderID
		return restoreReq, nil
	}

	personTargetTokenID := addition.CasPersonTargetTokenId
	personBaseFolderID := strings.TrimSpace(restoreReq.TargetFolderID)
	if personBaseFolderID == "" {
		personBaseFolderID = addition.CasPersonTargetFolderId
	}
	targetFolderID, terr := h.resolveDefaultPersonRestoreTargetFolder(ctx, vf.ID, personTargetTokenID, personBaseFolderID, addition.CasAutoCollectPreservePath)
	if terr != nil {
		return casrestoreSvi.RestoreRequest{}, terr
	}
	restoreReq.TargetFolderID = targetFolderID
	if restoreReq.TargetTokenID == 0 {
		restoreReq.TargetTokenID = personTargetTokenID
	}
	return restoreReq, nil
}

func (h *handler) resolveDefaultPersonRestoreTargetFolder(ctx *httpcontext.Context, casVirtualID int64, targetTokenID int64, baseTargetFolderID string, preservePath bool) (string, error) {
	folderID := strings.TrimSpace(baseTargetFolderID)
	if folderID == "" {
		folderID = "-11"
	}
	if !preservePath {
		return folderID, nil
	}
	if targetTokenID <= 0 {
		return "", fmt.Errorf("未配置个人恢复目标(casPersonTargetTokenId)")
	}
	fullPath, err := h.virtualfileService.CalFullPath(ctx.GetContext(), casVirtualID)
	if err != nil {
		return "", err
	}
	relDir := strings.Trim(strings.TrimPrefix(path.Dir(fullPath), "/"), " ")
	if relDir == "" || relDir == "." {
		return folderID, nil
	}
	session, err := h.appSessionService.GetByTokenID(ctx.GetContext(), targetTokenID)
	if err != nil {
		return "", err
	}
	panClient := buildPanClient(session)
	if panClient == nil {
		return "", fmt.Errorf("创建PanClient失败")
	}
	folder, apiErr := panClient.AppMkdirRecursive(0, folderID, relDir, 0, strings.Split(relDir, "/"))
	if apiErr != nil {
		return "", fmt.Errorf("创建个人恢复目标目录失败: %w", apiErr)
	}
	if folder == nil || strings.TrimSpace(folder.FileId) == "" {
		return "", fmt.Errorf("创建个人恢复目标目录失败: 未返回最终目标目录ID relativeDir=%s", relDir)
	}
	return strings.TrimSpace(folder.FileId), nil
}

func (h *handler) resolveDefaultFamilyRestoreTargetFolder(ctx *httpcontext.Context, casVirtualID int64, targetTokenID int64, familyID int64, baseTargetFolderID string, preservePath bool) (string, error) {
	folderID := strings.TrimSpace(baseTargetFolderID)
	if folderID == "" || !preservePath {
		return folderID, nil
	}
	if targetTokenID <= 0 {
		return "", fmt.Errorf("未配置家庭恢复目标(casFamilyTargetTokenId)")
	}
	fullPath, err := h.virtualfileService.CalFullPath(ctx.GetContext(), casVirtualID)
	if err != nil {
		return "", err
	}
	relDir := strings.Trim(strings.TrimPrefix(path.Dir(fullPath), "/"), " ")
	if relDir == "" || relDir == "." {
		return folderID, nil
	}
	session, err := h.appSessionService.GetByTokenID(ctx.GetContext(), targetTokenID)
	if err != nil {
		return "", err
	}
	panClient := buildPanClient(session)
	if panClient == nil {
		return "", fmt.Errorf("创建PanClient失败")
	}
	folder, apiErr := panClient.AppMkdirRecursive(familyID, folderID, relDir, 0, strings.Split(relDir, "/"))
	if apiErr != nil {
		return "", fmt.Errorf("创建家庭恢复目标目录失败: %w", apiErr)
	}
	if folder == nil || strings.TrimSpace(folder.FileId) == "" {
		return "", fmt.Errorf("创建家庭恢复目标目录失败: 未返回最终目标目录ID relativeDir=%s", relDir)
	}
	return strings.TrimSpace(folder.FileId), nil
}

func buildPanClient(session *appsession.Session) *cloudpan.PanClient {
	if session == nil {
		return nil
	}
	webToken := cloudpan.WebLoginToken{}
	if cookie := cloudpan.RefreshCookieToken(session.Token.SessionKey); cookie != "" {
		webToken.CookieLoginUser = cookie
	}
	return cloudpan.NewPanClient(webToken, session.Token)
}

func (h *handler) queryCASVirtualFile(ctx *httpcontext.Context, req *restoreCasRequest) (*struct {
	ID      int64
	CloudId string
	Name    string
}, error) {
	if req.CasVirtualID != 0 {
		vf, err := h.virtualfileService.Query(ctx.GetContext(), req.CasVirtualID)
		if err != nil {
			return nil, err
		}
		return &struct {
			ID      int64
			CloudId string
			Name    string
		}{ID: vf.ID, CloudId: vf.CloudId, Name: vf.Name}, nil
	}

	vf, err := h.virtualfileService.QueryByPath(ctx.GetContext(), req.CasPath)
	if err != nil {
		return nil, err
	}
	return &struct {
		ID      int64
		CloudId string
		Name    string
	}{ID: vf.ID, CloudId: vf.CloudId, Name: vf.Name}, nil
}
