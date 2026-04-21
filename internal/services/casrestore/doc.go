// Package casrestore 负责基于 .cas 元数据恢复真实媒体文件。
//
// 约束：
//   - 这里只承载恢复链，不承载播放入口、回收调度、STRM 写入。
//   - cloud189 API 细节后续通过子文件逐步补齐，避免把恢复链做成大泥球。
package casrestore
