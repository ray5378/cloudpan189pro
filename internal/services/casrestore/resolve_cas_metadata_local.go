package casrestore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xxcheng123/cloudpan189-share/internal/services/casparser"
)

func resolveLocalCAS(localPath string) (*casparser.CasInfo, string, error) {
	localPath = strings.TrimSpace(localPath)
	if localPath == "" {
		return nil, "", fmt.Errorf("localCasPath不能为空")
	}
	content, err := os.ReadFile(localPath)
	if err != nil {
		return nil, "", err
	}
	info, err := casparser.ParseCasContent(content)
	if err != nil {
		return nil, "", err
	}
	return info, filepath.Base(localPath), nil
}
