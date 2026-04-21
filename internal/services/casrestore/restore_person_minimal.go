package casrestore

import (
	"fmt"

	"github.com/tickstep/cloudpan189-api/cloudpan"
	"github.com/xxcheng123/cloudpan189-share/internal/services/casparser"
)

// personRestoreAdapter 已停用。
// 根据当前恢复设计约束：所有上传必须先秒传到家庭云盘，再转移到个人云盘。
// 保留这个类型仅为了明确禁止 direct person upload 作为主恢复路径。
type personRestoreAdapter struct{}

type personRestoreResult struct {
	RestoredFileID   string
	RestoredFileName string
}

func (a *personRestoreAdapter) TryRestore(
	_ *cloudpan.PanClient,
	_ string,
	_ string,
	_ *casparser.CasInfo,
) (*personRestoreResult, error) {
	return nil, fmt.Errorf("direct person upload 已禁用: 必须先恢复到家庭云盘再转移到个人云盘")
}
