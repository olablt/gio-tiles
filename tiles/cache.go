package tiles

type CacheType int

const (
    CacheImage CacheType = iota
    CacheImageOp
)

type Cache interface {
    Get(key string) (interface{}, bool)
    Set(key string, value interface{})
    Clear()
    GetType() CacheType
}
