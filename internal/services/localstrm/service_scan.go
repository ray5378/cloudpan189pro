package localstrm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
	"go.uber.org/zap"
)

type ScanResult struct {
	Scanned int `json:"scanned"`
	Created int `json:"created"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
}

func (s *service) ScanAndEnsureAll(ctx appctx.Context) (*ScanResult, error) {
	const localCASRoot = "/local_cas"
	result := &ScanResult{}
	if _, err := os.Stat(localCASRoot); err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, err
	}
	err := filepath.Walk(localCASRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			result.Failed++
			ctx.Warn("扫描本地CAS失败", zap.String("path", path), zap.Error(err))
			return nil
		}
		if info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".cas") {
			return nil
		}
		result.Scanned++
		strmPath := strings.TrimSuffix(strings.Replace(path, "/local_cas/", "/cas_strm/", 1), ".cas") + ".strm"
		if _, statErr := os.Stat(strmPath); statErr == nil {
			result.Skipped++
			return nil
		}
		if _, _, ensureErr := s.ensureForLocalPath(ctx, path, info.Name()); ensureErr != nil {
			result.Failed++
			ctx.Error("本地CAS补生成STRM失败", zap.String("cas_path", path), zap.Error(ensureErr))
			return nil
		}
		result.Created++
		return nil
	})
	if err != nil {
		return nil, err
	}
	ctx.Info("本地CAS扫描补STRM完成",
		zap.Int("scanned", result.Scanned),
		zap.Int("created", result.Created),
		zap.Int("skipped", result.Skipped),
		zap.Int("failed", result.Failed),
	)
	return result, nil
}

func (s *service) ensureForLocalPath(ctx appctx.Context, localCASPath string, fileName string) (string, int64, error) {
	if strings.TrimSpace(localCASPath) == "" {
		return "", 0, fmt.Errorf("localCASPath不能为空")
	}
	if strings.TrimSpace(fileName) == "" {
		fileName = filepath.Base(localCASPath)
	}
	vf := &models.VirtualFile{
		TopId:    0,
		CloudId:  filepath.ToSlash(strings.TrimSpace(localCASPath)),
		ParentId: 0,
		Name:     fileName,
	}
	return s.EnsureForLocalCAS(ctx, vf, localCASPath)
}
