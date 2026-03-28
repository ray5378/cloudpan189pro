package cloudbridge

import "github.com/xxcheng123/cloudpan189-interface/client"

type (
	AuthToken             = client.AuthToken
	GetFamilyListResponse = client.GetFamilyListResponse
	FileNode              struct {
		ParentId string `json:"parentId"`
		ID       string `json:"id"`
		Name     string `json:"name"`
		IsFolder int64  `json:"isFolder"`
	}
	FamilyFileListResponse struct {
		Data        []*FileNode `json:"data"`
		Total       int64       `json:"total"`
		CurrentPage int         `json:"currentPage"`
		PageSize    int         `json:"pageSize"`
	}
	PersonFileListResponse struct {
		Data        []*FileNode `json:"data"`
		Total       int64       `json:"total"`
		CurrentPage int         `json:"currentPage"`
		PageSize    int         `json:"pageSize"`
	}
)

func NewAuthToken(accessToken string, expires int64) AuthToken {
	return client.NewAuthToken(accessToken, expires)
}
