package casrestore

import (
	"fmt"

	"github.com/tickstep/cloudpan189-api/cloudpan"
	"github.com/xxcheng123/cloudpan189-share/internal/services/casparser"
)

type familyRestoreAdapter struct{}

func (a *familyRestoreAdapter) TryRestore(
	_ *cloudpan.PanClient,
	_ string,
	_ string,
	_ *casparser.CasInfo,
) (*personRestoreResult, error) {
	return nil, fmt.Errorf("family fallback 尚未实现")
}
