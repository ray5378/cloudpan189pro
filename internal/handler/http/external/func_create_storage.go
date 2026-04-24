package external

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/xxcheng123/cloudpan189-share/internal/consts"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/httpcontext"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/datatypes"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/utils"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	filetasklogSvi "github.com/xxcheng123/cloudpan189-share/internal/services/filetasklog"
	storagefacadeSvi "github.com/xxcheng123/cloudpan189-share/internal/services/storagefacade"
	"github.com/xxcheng123/cloudpan189-share/internal/types/topic"
)

var (
	reShareLink  = regexp.MustCompile(`cloud\.189\.cn/t/([a-zA-Z0-9]+)`)
	reAccessCode = regexp.MustCompile(`(?:\S+码|code)[:：]\s*([a-zA-Z0-9]+)`)
)

type createReq struct {
	DelayTime int    `json:"delayTime"`
	APIKey    string `json:"apiKey"`
	APIKey2   string `json:"api-key"`
	TokenId   *int64 `json:"tokenId"`
	Link      string `json:"shareLink"`
	LinkCN    string `json:"分享链接"`
	Target    string `json:"targetDir"`
	TargetCN  string `json:"目标文件夹"`
}

type createResp struct {
	TaskId int64 `json:"taskId"`
}

func (h *handler) CreateStorage() httpcontext.HandlerFunc {
	return func(ctx *httpcontext.Context) {
		var req createReq
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.AbortWithInvalidParams(err)
			return
		}

		// 每次直读 Setting
		st, err := h.settingService.Query(ctx.GetContext())
		if err != nil || st == nil {
			ctx.Response(http.StatusInternalServerError, http.StatusInternalServerError, "读取系统设置失败", nil)
			return
		}
		addition := st.Addition

		// 鉴权
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			apiKey = strings.TrimSpace(req.APIKey2)
		}
		if apiKey == "" {
			apiKey = strings.TrimSpace(ctx.GetHeader("X-API-Key"))
		}
		if addition.ExternalAPIKey == "" {
			ctx.Response(http.StatusForbidden, http.StatusForbidden, "未配置外部 API Key", nil)
			return
		}
		if apiKey == "" || apiKey != addition.ExternalAPIKey {
			ctx.Response(http.StatusUnauthorized, http.StatusUnauthorized, "invalid api key", nil)
			return
		}

		// 目标目录
		target := strings.TrimSpace(first(req.Target, req.TargetCN))
		if target == "" || !strings.HasPrefix(target, "/") || !utils.CheckIsPath(target) {
			ctx.Response(http.StatusBadRequest, http.StatusBadRequest, "目标文件夹必须以 / 开头且合法", nil)
			return
		}

		// 链接解析（对齐“存储管理-文件分享”兼容规则：支持包含分享链接/分享码/访问码的混合文本）
		clean := strings.TrimSpace(first(req.Link, req.LinkCN))
		if clean == "" {
			ctx.Response(http.StatusBadRequest, http.StatusBadRequest, "分享链接不能为空", nil)
			return
		}

		clean = strings.ReplaceAll(clean, "（", "(")
		clean = strings.ReplaceAll(clean, "）", ")")
		clean = strings.ReplaceAll(clean, "：", ":")

		// access code
		accessCode := ""
		if mm := reAccessCode.FindStringSubmatch(clean); len(mm) > 1 {
			accessCode = mm[1]
		} else {
			parts := strings.Fields(clean)
			if len(parts) > 1 {
				last := strings.Trim(parts[len(parts)-1], "()")
				if len(last) == 4 {
					accessCode = last
				}
			}
		}

		// share code
		shareCode := ""
		if mm := reShareLink.FindStringSubmatch(clean); len(mm) > 1 {
			shareCode = mm[1]
		} else {
			// allow pure code or 'code(访问码:xxxx)'
			if idx := strings.Index(clean, "("); idx > -1 {
				shareCode = strings.TrimSpace(clean[:idx])
			} else {
				fields := strings.Fields(clean)
				if len(fields) > 0 {
					shareCode = strings.Trim(fields[0], "()")
				}
			}
		}
		shareCode = strings.TrimSpace(shareCode)
		if shareCode == "" {
			ctx.Response(http.StatusBadRequest, http.StatusBadRequest, "无法识别分享码", nil)
			return
		}

		// 校验分享（使用 shareCode + accessCode；云端会校验是否匹配）
		result, err := h.cloudBridgeService.CheckShare(ctx.GetContext(), shareCode, accessCode)
		if err != nil || result == nil || result.ShareId == 0 {
			ctx.Response(http.StatusBadRequest, http.StatusBadRequest, "分享校验失败", nil)
			return
		}

		// 兼容：若云端返回的 accessCode 与我们提取的不一致，以云端返回为准
		if strings.TrimSpace(result.AccessCode) != "" {
			accessCode = result.AccessCode
		}

		// 选择 tokenId
		tokenId := addition.DefaultTokenId
		if req.TokenId != nil {
			tokenId = *req.TokenId
		}
		if tokenId <= 0 {
			ctx.Response(http.StatusBadRequest, http.StatusBadRequest, "默认云盘令牌无效", nil)
			return
		}

		// 组装 addition
		additionMap := datatypes.JSONMap{
			consts.FileAdditionKeyShareId:    result.ShareId,
			consts.FileAdditionKeyIsFolder:   result.IsFolder,
			consts.FileAdditionKeyShareMode:  result.ShareMode,
			consts.FileAdditionKeyAccessCode: accessCode,
		}

		// 创建任务日志（作为 taskId）
		tracker, err := h.fileTaskLogService.Create(ctx.GetContext(),
			"external_create_storage",
			"外部创建挂载",
			filetasklogSvi.WithDesc(target),
		)
		if err != nil {
			ctx.Response(http.StatusInternalServerError, http.StatusInternalServerError, "创建任务失败", nil)
			return
		}
		taskId := tracker.GetID()

		// 组装创建请求（让 worker 使用 storagefacade.CreateStorage 直接创建）
		enableAutoRefresh := false
		if addition.ExternalAutoRefreshEnabled != nil {
			enableAutoRefresh = *addition.ExternalAutoRefreshEnabled
		}

		payload := map[string]any{
			"taskId": taskId,
			"req": &storagefacadeSvi.CreateStorageRequest{
				LocalPath:         target,
				OsType:            models.OsTypeShareFolder,
				CloudToken:        tokenId,
				FileId:            result.FileId,
				Addition:          additionMap,
				EnableAutoRefresh: enableAutoRefresh,
				AutoRefreshDays:   addition.ExternalAutoRefreshDays,
				RefreshInterval:   addition.ExternalRefreshIntervalMin,
				EnableDeepRefresh: false,
			},
		}
		bs, _ := json.Marshal(payload)
		if err := h.taskEngine.PushMessage(ctx.GetContext(), topic.ExternalCreateStorageRequest{}.Topic(), bs); err != nil {
			_ = h.fileTaskLogService.WithErrorAndFail(ctx.GetContext(), tracker, err)
			ctx.Response(http.StatusInternalServerError, http.StatusInternalServerError, "入队失败", nil)
			return
		}

		ctx.Response(http.StatusAccepted, http.StatusAccepted, "accepted", &createResp{TaskId: taskId})
	}
}

func first(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return strings.TrimSpace(b)
}
