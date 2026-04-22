package media

import (
	"fmt"
	"strconv"

	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	casrestoreSvi "github.com/xxcheng123/cloudpan189-share/internal/services/casrestore"
)

// restoreCasRequest 手动触发 CAS 恢复。
// 注意：uploadRoute 表示秒传/上传路线；destinationType 表示最终目录归属。
// 为了便于手动联调，接口支持三种模式：
// 1. 最简模式：传 casVirtualId + destinationType + targetFolderId（其余上下文自动反查）
// 2. 路径模式：传 casPath + destinationType + targetFolderId（先按路径查 VirtualFile，再自动反查）
// 3. 显式模式：把 storageId / mountPointId / casFileId / casFileName 一起传进来
//
// 这里的 storageId 兜底取挂载点根 file_id，和现有 storage/list 返回的 id 语义保持一致。
type restoreCasRequest struct {
	StorageID       int64                         `json:"storageId" binding:"omitempty" example:"1"`
	MountPointID    int64                         `json:"mountPointId" binding:"omitempty" example:"1"`
	CasFileID       string                        `json:"casFileId" binding:"omitempty" example:"123456789"`
	CasFileName     string                        `json:"casFileName" binding:"omitempty" example:"movie.cas"`
	CasVirtualID    int64                         `json:"casVirtualId" binding:"omitempty" example:"1001"`
	CasPath         string                        `json:"casPath" binding:"omitempty" example:"/电影库/movie.cas"`
	UploadRoute     casrestoreSvi.UploadRoute     `json:"uploadRoute" binding:"omitempty,oneof=family person" example:"family"`
	DestinationType casrestoreSvi.DestinationType `json:"destinationType" binding:"required,oneof=family person" example:"family"`
	TargetFolderID  string                        `json:"targetFolderId" binding:"required" example:"-11"`
}

// RestoreCas 手动触发单个 CAS 恢复。
// @Summary 手动恢复CAS文件
// @Description 根据 .cas 元数据立刻执行一次恢复。uploadRoute 表示上传路线，destinationType 表示最终目录归属。
// @Tags 媒体操作
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body restoreCasRequest true "恢复请求"
// @Success 200 {object} httpcontext.Response{data=casrestore.RestoreResult} "恢复成功"
// @Failure 400 {object} httpcontext.Response "参数验证失败，code=99998"
// @Failure 400 {object} httpcontext.Response "恢复失败"
// @Failure 401 {object} httpcontext.Response "未授权访问"
// @Failure 403 {object} httpcontext.Response "权限不足"
// @Router /api/media/restore_cas [post]
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
		// 已对齐参考实现的组合必须在接口层显式收口，避免外部误以为所有产品语义组合都已具备 reference-backed 主链。
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
		restoreReq.MountPointID = mp.ID
	}
	if restoreReq.StorageID == 0 {
		// 这里沿用 storage/list 的 ID 语义：storageId 对应 mount point root file_id。
		restoreReq.StorageID = mp.FileId
	}
	if h.settingService != nil && restoreReq.DestinationType == casrestoreSvi.DestinationTypeFamily {
		if latest, qerr := h.settingService.Query(ctx.GetContext()); qerr == nil && latest != nil {
			addition := latest.Addition
			if addition.CasTargetType == string(casrestoreSvi.DestinationTypeFamily) && addition.CasTargetFamilyId != "" {
				if parsed, perr := strconv.ParseInt(addition.CasTargetFamilyId, 10, 64); perr == nil && parsed > 0 {
					restoreReq.FamilyID = parsed
				}
			}
		}
	}

	return restoreReq, nil
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
