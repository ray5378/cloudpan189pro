package autoingest

type OnConflict string

const (
	OnConflictRename  OnConflict = "rename"
	OnConflictAbandon OnConflict = "abandon"
)
