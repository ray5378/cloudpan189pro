package virtualfile

import (
	"github.com/samber/lo"
	"github.com/xxcheng123/cloudpan189-share/internal/framework/context"
	"go.uber.org/zap"
)

type GroupCountByTopIdRequest struct {
	TopId     int64
	TopIdList []int64
}

type GroupCountByTopId struct {
	TopId int64 `gorm:"column:top_id" json:"topId"`
	Count int64 `gorm:"column:count" json:"count"`
}

const groupCountByTopIDBatchSize = 100

func (s *service) GroupCountByTopId(ctx context.Context, req *GroupCountByTopIdRequest) ([]*GroupCountByTopId, error) {
	if len(req.TopIdList) == 0 {
		query := s.getDB(ctx).Select("top_id, COUNT(*) as count").Where("top_id != id").Group("top_id")
		if req.TopId != 0 {
			query = query.Where("top_id = ?", req.TopId)
		}

		var result []*GroupCountByTopId
		if err := query.Find(&result).Error; err != nil {
			ctx.Error("按TopId分组统计失败", zap.Error(err))
			return nil, err
		}
		return result, nil
	}

	resultMap := make(map[int64]int64, len(req.TopIdList))
	for _, batch := range lo.Chunk(req.TopIdList, groupCountByTopIDBatchSize) {
		if len(batch) == 0 {
			continue
		}

		query := s.getDB(ctx).
			Select("top_id, COUNT(*) as count").
			Where("top_id != id").
			Where("top_id IN ?", batch).
			Group("top_id")
		if req.TopId != 0 {
			query = query.Where("top_id = ?", req.TopId)
		}

		var partial []*GroupCountByTopId
		if err := query.Find(&partial).Error; err != nil {
			ctx.Error("按TopId分组统计失败", zap.Error(err))
			return nil, err
		}

		for _, item := range partial {
			if item == nil {
				continue
			}
			resultMap[item.TopId] = item.Count
		}
	}

	result := make([]*GroupCountByTopId, 0, len(resultMap))
	for topID, count := range resultMap {
		result = append(result, &GroupCountByTopId{TopId: topID, Count: count})
	}
	return result, nil
}
