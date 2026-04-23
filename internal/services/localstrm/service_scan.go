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
	Pruned  int `json:"pruned"`
}

func (s *service) ScanAndEnsureAll(ctx appctx.Context) (*ScanResult, error) {
	const localCASRoot = "/local_cas"
	const localSTRMRoot = "/cas_strm"

	result := &ScanResult{}
	if _, err := os.Stat(localCASRoot); err != nil {
		if os.IsNotExist(err) {
			if pruneErr := s.pruneOrphanSTRM(ctx, localCASRoot, localSTRMRoot, result); pruneErr != nil {
				return nil, pruneErr
			}
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

	if err := s.pruneOrphanSTRM(ctx, localCASRoot, localSTRMRoot, result); err != nil {
		return nil, err
	}

	ctx.Info("本地CAS扫描补STRM完成",
		zap.Int("scanned", result.Scanned),
		zap.Int("created", result.Created),
		zap.Int("skipped", result.Skipped),
		zap.Int("failed", result.Failed),
		zap.Int("pruned", result.Pruned),
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

func (s *service) pruneOrphanSTRM(ctx appctx.Context, localCASRoot, localSTRMRoot string, result *ScanResult) error {
	if _, err := os.Stat(localSTRMRoot); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return filepath.Walk(localSTRMRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			result.Failed++
			ctx.Warn("扫描本地STRM失败", zap.String("path", path), zap.Error(err))
			return nil
		}
		if info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".strm") {
			return nil
		}

		relSTRMPath, relErr := filepath.Rel(localSTRMRoot, path)
		if relErr != nil {
			result.Failed++
			ctx.Warn("计算STRM相对路径失败", zap.String("strm_path", path), zap.Error(relErr))
			return nil
		}
		if strings.HasPrefix(relSTRMPath, "..") {
			result.Failed++
			ctx.Warn("STRM路径不在/cas_strm下", zap.String("strm_path", path))
			return nil
		}

		casFullPath := filepath.Join(localCASRoot, strings.TrimSuffix(relSTRMPath, ".strm")+".cas")
		if _, statErr := os.Stat(casFullPath); statErr == nil {
			return nil
		} else if !os.IsNotExist(statErr) {
			result.Failed++
			ctx.Warn("检查对应CAS文件失败", zap.String("cas_path", casFullPath), zap.Error(statErr))
			return nil
		}

		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			result.Failed++
			ctx.Warn("删除孤儿STRM失败", zap.String("strm_path", path), zap.Error(err))
			return nil
		}

		casRelPath := filepath.ToSlash(strings.TrimSuffix(relSTRMPath, ".strm") + ".cas")
		if delErr := s.casRecordService.DeleteByCasFilePath(ctx, casRelPath); delErr != nil {
			result.Failed++
			ctx.Warn("删除孤儿STRM对应CAS记录失败", zap.String("cas_file_path", casRelPath), zap.Error(delErr))
			return nil
		}

		s.clearEmptySTRMAncestors(ctx, localSTRMRoot, filepath.Dir(path))

		result.Pruned++
		ctx.Info("已清理孤儿STRM",
			zap.String("strm_path", path),
			zap.String("cas_file_path", casRelPath),
		)
		return nil
	})
}

func (s *service) clearEmptySTRMAncestors(ctx appctx.Context, rootPath, startDir string) {
	rootClean := filepath.Clean(rootPath)
	current := filepath.Clean(startDir)

	for {
		if current == rootClean || current == "." || current == string(filepath.Separator) {
			return
		}
		if !strings.HasPrefix(current, rootClean+string(filepath.Separator)) {
			return
		}

		entries, err := os.ReadDir(current)
		if err != nil {
			return
		}
		if len(entries) > 0 {
			return
		}
		if err := os.Remove(current); err != nil && !os.IsNotExist(err) {
			ctx.Warn("清理空STRM目录失败", zap.String("dir", current), zap.Error(err))
			return
		}
		ctx.Debug("已清理空STRM目录", zap.String("dir", current))
		current = filepath.Dir(current)
	}
}
