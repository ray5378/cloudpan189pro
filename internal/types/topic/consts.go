package topic

const (
	KeyFileScanFile    = "topic::file::scan::file"
	KeyFileClearFile   = "topic::file::clear::file"
	KeyFileBatchDelete = "topic::file::batch_delete::file"

	KeyAutoIngestRefreshSubscribe = "topic::autoingest::refresh::subscribe"

	// KeyMediaClear 清除媒体文件
	KeyMediaClear = "topic::media::clear"
	// KeyMediaRebuildStrmFile 重建媒体文件 strm 文件
	KeyMediaRebuildStrmFile = "topic::media::rebuild::strm::file"

	// KeyExternalCreateStorage 外部接口创建挂载
	KeyExternalCreateStorage = "topic::external::create_storage"
)
