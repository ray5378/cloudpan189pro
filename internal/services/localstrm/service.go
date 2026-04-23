package localstrm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/xxcheng123/cloudpan189-share/internal/bootstrap"
	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"github.com/xxcheng123/cloudpan189-share/internal/services/casrecord"
	"github.com/xxcheng123/cloudpan189-share/internal/shared"
	"go.uber.org/zap"
)

type Service interface {
	EnsureForLocalCAS(ctx appctx.Context, file *models.VirtualFile, localCASPath string) (string, int64, error)
	ScanAndEnsureAll(ctx appctx.Context) (*ScanResult, error)
}

type service struct {
	svc              bootstrap.ServiceContext
	casRecordService casrecord.Service
}

func NewService(svc bootstrap.ServiceContext) Service {
	return &service{
		svc:              svc,
		casRecordService: casrecord.NewService(svc),
	}
}

func (s *service) EnsureForLocalCAS(ctx appctx.Context, file *models.VirtualFile, localCASPath string) (string, int64, error) {
	if file == nil {
		return "", 0, fmt.Errorf("file不能为空")
	}
	if strings.TrimSpace(localCASPath) == "" {
		return "", 0, fmt.Errorf("localCASPath不能为空")
	}
	if !strings.HasSuffix(strings.ToLower(file.Name), ".cas") {
		return "", 0, fmt.Errorf("目标文件不是.cas: %s", file.Name)
	}

	const localCASRoot = "/local_cas"
	const localSTRMRoot = "/cas_strm"

	relCASPath, err := filepath.Rel(localCASRoot, localCASPath)
	if err != nil {
		return "", 0, err
	}
	if strings.HasPrefix(relCASPath, "..") {
		return "", 0, fmt.Errorf("localCASPath不在/local_cas下: %s", localCASPath)
	}

	strmRelPath := strings.TrimSuffix(filepath.ToSlash(relCASPath), ".cas") + ".strm"
	strmFullPath := filepath.Join(localSTRMRoot, filepath.FromSlash(strmRelPath))
	if err := os.MkdirAll(filepath.Dir(strmFullPath), 0o755); err != nil {
		return "", 0, err
	}

	record, err := s.casRecordService.QueryByStorageAndCasFileID(ctx, file.TopId, file.CloudId)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", 0, err
		}
		record = &models.CasMediaRecord{
			StorageID:        file.TopId,
			MountPointID:     file.TopId,
			CasFileID:        file.CloudId,
			CasFileName:      file.Name,
			CasFilePath:      filepath.ToSlash(relCASPath),
			SourceParentID:   fmt.Sprintf("%d", file.ParentId),
			OriginalFileName: strings.TrimSuffix(file.Name, ".cas"),
			RestoreStatus:    models.CasRestoreStatusPending,
			StrmRelativePath: strmRelPath,
		}
		if _, err := s.casRecordService.Create(ctx, record); err != nil {
			return "", 0, err
		}
	} else {
		updates := map[string]any{
			"cas_file_name":      file.Name,
			"cas_file_path":      filepath.ToSlash(relCASPath),
			"source_parent_id":   fmt.Sprintf("%d", file.ParentId),
			"original_file_name": strings.TrimSuffix(file.Name, ".cas"),
			"strm_relative_path": strmRelPath,
		}
		if err := s.casRecordService.Update(ctx, record.ID, updates); err != nil {
			return "", 0, err
		}
		record.StrmRelativePath = strmRelPath
	}

	baseURL := strings.TrimRight(strings.TrimSpace(shared.BaseURL), "/")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:12395"
	}
	playURL := fmt.Sprintf("%s/api/cas/play/%d", baseURL, record.ID)
	if err := os.WriteFile(strmFullPath, []byte(playURL), 0o644); err != nil {
		return "", 0, err
	}

	ctx.Info("本地CAS已生成STRM",
		zap.String("file_name", file.Name),
		zap.String("cas_local_path", localCASPath),
		zap.String("strm_full_path", strmFullPath),
		zap.Int64("cas_record_id", record.ID),
	)
	return strmFullPath, record.ID, nil
}
