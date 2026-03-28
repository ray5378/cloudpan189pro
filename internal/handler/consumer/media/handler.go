package media

import (
	"github.com/xxcheng123/cloudpan189-share/internal/framework/taskcontext"

	mediafileSvi "github.com/xxcheng123/cloudpan189-share/internal/services/mediafile"
	mountpointSvi "github.com/xxcheng123/cloudpan189-share/internal/services/mountpoint"
	verifySvi "github.com/xxcheng123/cloudpan189-share/internal/services/verify"
	virtualfileSvi "github.com/xxcheng123/cloudpan189-share/internal/services/virtualfile"
)

type Handler interface {
	Clear() taskcontext.HandlerFunc
	RebuildStrmFile() taskcontext.HandlerFunc
}

type handler struct {
	mediaFileService   mediafileSvi.Service
	mountpointService  mountpointSvi.Service
	virtualfileService virtualfileSvi.Service
	verifyService      verifySvi.Service
}

func NewHandler(
	mediaFileService mediafileSvi.Service,
	mountpointService mountpointSvi.Service,
	virtualfileService virtualfileSvi.Service,
	verifyService verifySvi.Service,
) Handler {
	return &handler{
		mediaFileService:   mediaFileService,
		mountpointService:  mountpointService,
		virtualfileService: virtualfileService,
		verifyService:      verifyService,
	}
}
