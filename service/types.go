package service

import "context"

type UIDGetter interface {
	GetUID(ctx context.Context, bizId int64) (int64, error)
}
