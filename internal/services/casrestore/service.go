package casrestore

import (
	stdctx "context"
	"sync"

	"github.com/xxcheng123/cloudpan189-share/internal/bootstrap"
	appctx "github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/services/casparser"
	"github.com/xxcheng123/cloudpan189-share/internal/services/casrecord"
)

type RestoreRequest struct {
	StorageID      int64
	CasFileID      string
	CasFileName    string
	TargetFolderID string
}

type RestoreResult struct {
	RestoredFileID   string
	RestoredFileName string
	TargetFolderID   string
	CasInfo          *casparser.CasInfo
}

type Service interface {
	EnsureRestored(ctx appctx.Context, req RestoreRequest) (*RestoreResult, error)
}

type service struct {
	svc        bootstrap.ServiceContext
	recordSvc  casrecord.Service
	inflightMu sync.Mutex
	inflight   map[string]*restoreCall
}

type restoreCall struct {
	wg     sync.WaitGroup
	result *RestoreResult
	err    error
}

func NewService(svc bootstrap.ServiceContext) Service {
	return &service{
		svc:       svc,
		recordSvc: casrecord.NewService(svc),
		inflight:  make(map[string]*restoreCall),
	}
}

func inflightKey(req RestoreRequest) string {
	return req.CasFileID + "::" + req.TargetFolderID
}

func (s *service) withInflight(_ stdctx.Context, key string, fn func() (*RestoreResult, error)) (*RestoreResult, error) {
	s.inflightMu.Lock()
	if call, ok := s.inflight[key]; ok {
		s.inflightMu.Unlock()
		call.wg.Wait()
		return call.result, call.err
	}
	call := &restoreCall{}
	call.wg.Add(1)
	s.inflight[key] = call
	s.inflightMu.Unlock()

	defer func() {
		s.inflightMu.Lock()
		delete(s.inflight, key)
		s.inflightMu.Unlock()
		call.wg.Done()
	}()

	call.result, call.err = fn()
	return call.result, call.err
}
