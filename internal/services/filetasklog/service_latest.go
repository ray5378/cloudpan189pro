package filetasklog

import (
	"github.com/samber/lo"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
)

const latestByFileIDsBatchSize = 100

// LatestByFileIDs 按 file_id 批量查询每个文件最新一条任务日志。
func (s *service) LatestByFileIDs(ctx context.Context, fileIDs []int64) (map[int64]*models.FileTaskLog, error) {
	result := make(map[int64]*models.FileTaskLog)
	if len(fileIDs) == 0 {
		return result, nil
	}

	for _, batch := range lo.Chunk(fileIDs, latestByFileIDsBatchSize) {
		if len(batch) == 0 {
			continue
		}

		list := make([]*models.FileTaskLog, 0, len(batch))
		err := s.getDB(ctx).
			Table("file_task_logs AS fl").
			Select("fl.*").
			Joins("JOIN (SELECT file_id, MAX(id) AS max_id FROM file_task_logs WHERE file_id IN ? GROUP BY file_id) latest ON latest.max_id = fl.id", batch).
			Find(&list).Error
		if err != nil {
			return nil, err
		}

		for _, item := range list {
			if item == nil || item.FileId == 0 {
				continue
			}
			result[item.FileId] = item
		}
	}

	return result, nil
}
