package autoingest

type SourceType string

const (
	SourceTypeSubscribe SourceType = "subscribe"
)

func (s SourceType) String() string {
	return string(s)
}
