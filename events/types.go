package events

import feedv1 "github.com/MuxiKeStack/be-api/gen/proto/feed/v1"

const topicFeedEvent = "feed_event"

type FeedEvent struct {
	Type     feedv1.EventType
	Metadata map[string]string
}
