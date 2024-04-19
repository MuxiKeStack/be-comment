package cache

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

var ErrKeyNotExists = redis.Nil

type CommentCache interface {
	GetBizCommentCount(ctx context.Context, biz int32, bizId int64) (int64, error)
	SetBizCommentCount(ctx context.Context, biz int32, bizId int64, count int64) error
	IncrBizCommentCountIfPresent(ctx context.Context, biz int32, bizId int64) error
	DecrBizCommentCountIfPresent(ctx context.Context, biz int32, bizId int64) error
}

type RedisCommentCache struct {
	cmd redis.Cmdable
}

func NewRedisCommentCache(cmd redis.Cmdable) CommentCache {
	return &RedisCommentCache{cmd: cmd}
}

func (cache *RedisCommentCache) GetBizCommentCount(ctx context.Context, biz int32, bizId int64) (int64, error) {
	key := cache.bizCommentCountKey(biz, bizId)
	return cache.cmd.Get(ctx, key).Int64()
}

func (cache *RedisCommentCache) SetBizCommentCount(ctx context.Context, biz int32, bizId int64, count int64) error {
	key := cache.bizCommentCountKey(biz, bizId)
	return cache.cmd.Set(ctx, key, count, time.Minute*10).Err()
}

func (cache *RedisCommentCache) IncrBizCommentCountIfPresent(ctx context.Context, biz int32, bizId int64) error {
	key := cache.bizCommentCountKey(biz, bizId)
	return cache.cmd.Incr(ctx, key).Err()
}

func (cache *RedisCommentCache) DecrBizCommentCountIfPresent(ctx context.Context, biz int32, bizId int64) error {
	key := cache.bizCommentCountKey(biz, bizId)
	return cache.cmd.Decr(ctx, key).Err()
}

func (cache *RedisCommentCache) bizCommentCountKey(biz int32, bizId int64) string {
	return fmt.Sprintf("kstack:comment:biz_comment_count:<%d,%d>", biz, bizId)
}
