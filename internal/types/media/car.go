package media

import (
	"path"
	"path/filepath"
)

type WriterCar interface {
	RootPath() string
	NewSubCar(filePath string) WriterCar
	GetFileConflictPolicy() FileConflictPolicy
	GetBaseURL() string
	// GetFullPath 返回 car 指向的文件的完整路径
	GetFullPath() string
	// GetPath 返回 car 指向的文件的路径（相对于根路径）
	GetPath() string
	// GetName 获取本级文件名
	GetName() string
}

func NewWriterCar(rootPath string, conflictPolicy FileConflictPolicy, baseURL string) WriterCar {
	return &writerCar{
		rootPath:       rootPath,
		conflictPolicy: conflictPolicy,
		baseURL:        baseURL,
	}
}

type writerCar struct {
	rootPath       string
	conflictPolicy FileConflictPolicy
	baseURL        string
	fullPath       string
}

func (c *writerCar) RootPath() string {
	return c.rootPath
}

func (c *writerCar) NewSubCar(filePath string) WriterCar {
	return &writerCar{
		rootPath:       c.rootPath,
		conflictPolicy: c.conflictPolicy,
		baseURL:        c.baseURL,
		fullPath:       path.Join(c.fullPath, filePath),
	}
}

func (c *writerCar) GetFileConflictPolicy() FileConflictPolicy {
	return c.conflictPolicy
}

func (c *writerCar) GetBaseURL() string {
	return c.baseURL
}

func (c *writerCar) GetFullPath() string {
	return filepath.Join(c.rootPath, c.fullPath)
}

func (c *writerCar) GetPath() string {
	return path.Join(c.fullPath)
}

func (c *writerCar) GetName() string {
	return path.Base(c.fullPath)
}
