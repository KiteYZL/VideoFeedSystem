package video

import "time"

const (
	GlobalTimelineKey         = "feed:global_timeline"
	HotSnapshotPrefix         = "feed:hot:snapshot:"
	HotWindowPrefix           = "feed:hot:window:"
	VideoCachePrefix          = "video:detail:"
	ViewDedupPrefix           = "video:view:"
	CacheInvalidateChannel    = "video:cache:invalidate"
	ProcessedRedisEventPrefix = "video:event:processed:"
	StateVersionPrefix        = "video:state:version:"
)

func hotWindowKey(t time.Time) string { return HotWindowPrefix + t.UTC().Format("200601021504") }
