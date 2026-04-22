package converter

import (
	"strings"
	"time"

	"github.com/xxcheng123/cloudpan189-interface/client"
	"github.com/xxcheng123/cloudpan189-share/internal/consts"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/datatypes"
	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/utils"
	"github.com/xxcheng123/cloudpan189-share/internal/repository/models"
)

type VirtualFileConverter interface {
	TransformVirtualFile(tid, pid int64) *models.VirtualFile
}

type ShareFileInfo struct {
	*client.ShareFileInfo
	upUserId string
}

func NewShareFileInfo(fileInfo *client.ShareFileInfo, upUserId string) *ShareFileInfo {
	return &ShareFileInfo{
		ShareFileInfo: fileInfo,
		upUserId:      upUserId,
	}
}

func (f *ShareFileInfo) TransformVirtualFile(tid, pid int64) *models.VirtualFile {
	createDate, _ := time.Parse(time.DateTime, f.CreateDate)
	modifyDate, _ := time.Parse(time.DateTime, f.LastOpTime)

	addition := datatypes.JSONMap{
		consts.FileAdditionKeyShareId:  f.ShareId,
		consts.FileAdditionKeyIsFolder: f.Folder,
		consts.FileAdditionKeyUpUserId: f.upUserId,
	}
	if f.AccessURL != "" {
		addition[consts.FileAdditionKeyAccessURL] = f.AccessURL
	}

	return &models.VirtualFile{
		ParentId:   pid,
		Name:       utils.SanitizeFileName(f.Name),
		TopId:      tid,
		IsTop:      false,
		CloudId:    string(f.Id),
		Size:       f.Size,
		IsDir:      f.Folder == 1,
		Hash:       strings.ToLower(f.Md5),
		CreateDate: createDate,
		ModifyDate: modifyDate,
		OsType:     models.OsTypeSubscribeShareFolder,
		Addition:   addition,
		Rev:        f.Rev,
	}
}

type FileInfo struct {
	*client.FileInfo
	addition datatypes.JSONMap
	osType   models.OsType
}

func NewFileInfo(input client.FileInfo, osType models.OsType, addition datatypes.JSONMap) *FileInfo {
	f := &FileInfo{
		FileInfo: &input,
		addition: addition,
		osType:   osType,
	}

	return f
}

func (f *FileInfo) TransformVirtualFile(tid, pid int64) *models.VirtualFile {
	if f.FileInfo == nil {
		return nil
	}

	file := f.FileInfo

	createDate, _ := time.Parse(time.DateTime, file.CreateDate)
	modifyDate, _ := time.Parse(time.DateTime, file.LastOpTime)

	return &models.VirtualFile{
		ParentId:   pid,
		TopId:      tid,
		Name:       utils.SanitizeFileName(file.Name),
		IsTop:      false,
		Size:       file.Size,
		IsDir:      false,
		Hash:       strings.ToLower(file.Md5),
		CreateDate: createDate,
		ModifyDate: modifyDate,
		OsType:     f.osType,
		Addition:   f.addition,
		Rev:        file.Rev,
		CloudId:    string(file.Id),
	}
}

type FolderInfo struct {
	*client.FolderInfo
	addition datatypes.JSONMap
	osType   models.OsType
}

func NewFolderInfo(input client.FolderInfo, osType models.OsType, addition datatypes.JSONMap) *FolderInfo {
	f := &FolderInfo{
		FolderInfo: &input,
		addition:   addition,
		osType:     osType,
	}

	return f
}

func (f *FolderInfo) TransformVirtualFile(tid, pid int64) *models.VirtualFile {
	if f.FolderInfo == nil {
		return nil
	}

	folder := f.FolderInfo

	createDate, _ := time.Parse(time.DateTime, folder.CreateDate)
	modifyDate, _ := time.Parse(time.DateTime, folder.LastOpTime)

	return &models.VirtualFile{
		ParentId:   pid,
		TopId:      tid,
		Name:       utils.SanitizeFileName(folder.Name),
		IsTop:      false,
		Size:       0,
		IsDir:      true,
		Hash:       "",
		CreateDate: createDate,
		ModifyDate: modifyDate,
		OsType:     f.osType,
		Addition:   f.addition,
		Rev:        folder.Rev,
		CloudId:    string(folder.Id),
	}
}
