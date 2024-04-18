package repository

import (
	"context"
	"database/sql"
	commentv1 "github.com/MuxiKeStack/be-api/gen/proto/comment/v1"
	"github.com/MuxiKeStack/be-comment/domain"
	"github.com/MuxiKeStack/be-comment/pkg/logger"
	"github.com/MuxiKeStack/be-comment/repository/dao"
	"golang.org/x/sync/errgroup"
	"time"
)

type CommentRepository interface {
	FindByBiz(ctx context.Context, biz commentv1.Biz, bizId int64, curCommentId int64, limit int64) ([]domain.Comment, error)
	DeleteComment(ctx context.Context, commentId int64, uid int64) error
	GetCountByBiz(ctx context.Context, biz commentv1.Biz, bizId int64) (int64, error)
	GetMoreReplies(ctx context.Context, rid int64, curCommentId int64, limit int64) ([]domain.Comment, error)
	CreateComment(ctx context.Context, comment domain.Comment) error
	FindById(ctx context.Context, commentId int64) (domain.Comment, error)
}

type CachedCommentRepo struct {
	dao dao.CommentDAO
	l   logger.Logger
}

func (repo *CachedCommentRepo) FindById(ctx context.Context, commentId int64) (domain.Comment, error) {
	comment, err := repo.dao.FindById(ctx, commentId)
	return repo.toDomain(comment), err
}

func NewCachedCommentRepo(dao dao.CommentDAO, l logger.Logger) CommentRepository {
	return &CachedCommentRepo{dao: dao, l: l}
}

func (repo *CachedCommentRepo) FindByBiz(ctx context.Context, biz commentv1.Biz, bizId int64, curCommentId int64, limit int64) ([]domain.Comment, error) {
	daoComments, err := repo.dao.FindByBiz(ctx, int32(biz), bizId, curCommentId, limit)
	if err != nil {
		return nil, err
	}
	res := make([]domain.Comment, 0, len(daoComments))
	// 只找三条
	var eg errgroup.Group
	downgraded := ctx.Value("downgraded") == "true"
	for _, d := range daoComments {
		cm := repo.toDomain(d) // todo
		res = append(res, cm)
		if downgraded {
			continue
		}
		eg.Go(func() error {
			// 只展示三条
			cm.Children = make([]domain.Comment, 0, 3)
			rs, err := repo.dao.FindRepliesByPid(ctx, d.Id, 0, 3)
			if err != nil {
				// 我们认为这是一个可以容忍的错误
				repo.l.Error("查询子评论失败", logger.Error(err))
				return nil
			}
			for _, r := range rs {
				cm.Children = append(cm.Children, repo.toDomain(r))
			}
			return nil
		})
	}
	return res, eg.Wait()
}

func (repo *CachedCommentRepo) DeleteComment(ctx context.Context, commentId int64, uid int64) error {
	return repo.dao.Delete(ctx, commentId, uid)
}

func (repo *CachedCommentRepo) GetCountByBiz(ctx context.Context, biz commentv1.Biz, bizId int64) (int64, error) {
	return repo.dao.GetCountByBiz(ctx, int32(biz), bizId)
}

func (repo *CachedCommentRepo) GetMoreReplies(ctx context.Context, rid int64, curCommentId int64, limit int64) ([]domain.Comment, error) {
	cs, err := repo.dao.FindRepliesByRid(ctx, rid, curCommentId, limit)
	if err != nil {
		return nil, err
	}
	res := make([]domain.Comment, 0, len(cs))
	for _, cm := range cs {
		res = append(res, repo.toDomain(cm))
	}
	return res, nil
}

func (repo *CachedCommentRepo) CreateComment(ctx context.Context, comment domain.Comment) error {
	return repo.dao.Insert(ctx, repo.toEntity(comment))
}

func (repo *CachedCommentRepo) toDomain(daoComment dao.Comment) domain.Comment {
	val := domain.Comment{
		Id: daoComment.Id,
		Commentator: domain.User{
			ID: daoComment.Uid,
		},
		Biz:        commentv1.Biz(daoComment.Biz),
		BizID:      daoComment.BizID,
		Content:    daoComment.Content,
		ReplyToUid: daoComment.ReplyToUid,
		CTime:      time.UnixMilli(daoComment.Ctime),
		UTime:      time.UnixMilli(daoComment.Utime),
	}
	if daoComment.PID.Valid {
		val.ParentComment = &domain.Comment{
			Id: daoComment.PID.Int64,
		}
	}
	if daoComment.RootID.Valid {
		val.RootComment = &domain.Comment{
			Id: daoComment.RootID.Int64,
		}
	}
	return val
}

func (repo *CachedCommentRepo) toEntity(domainComment domain.Comment) dao.Comment {
	daoComment := dao.Comment{
		Id:            domainComment.Id,
		Uid:           domainComment.Commentator.ID,
		Biz:           int32(domainComment.Biz),
		BizID:         domainComment.BizID,
		ReplyToUid:    domainComment.ReplyToUid,
		ParentComment: nil,
		Content:       domainComment.Content,
	}
	if domainComment.RootComment != nil {
		daoComment.RootID = sql.NullInt64{
			Valid: domainComment.RootComment.Id != 0,
			Int64: domainComment.RootComment.Id,
		}
	}
	if domainComment.ParentComment != nil {
		daoComment.PID = sql.NullInt64{
			Valid: domainComment.RootComment.Id != 0,
			Int64: domainComment.ParentComment.Id,
		}
	}
	return daoComment
}
