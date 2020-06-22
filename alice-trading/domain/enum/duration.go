package enum

// キャッシュの有効期限
type Duration int

const (
	DefaultExpiration Duration = iota
	NoExpiration
)
