package media

type Type = string

const (
	TypeStrm Type = "strm"
)

type FileConflictPolicy = string

const (
	FileConflictPolicySkip    FileConflictPolicy = "skip"
	FileConflictPolicyReplace FileConflictPolicy = "replace"
)
