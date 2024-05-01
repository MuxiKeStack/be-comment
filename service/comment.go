package service

import (
	"context"
	"errors"
	answerv1 "github.com/MuxiKeStack/be-api/gen/proto/answer/v1"
	commentv1 "github.com/MuxiKeStack/be-api/gen/proto/comment/v1"
	evaluationv1 "github.com/MuxiKeStack/be-api/gen/proto/evaluation/v1"
	feedv1 "github.com/MuxiKeStack/be-api/gen/proto/feed/v1"
	"github.com/MuxiKeStack/be-comment/domain"
	"github.com/MuxiKeStack/be-comment/events"
	"github.com/MuxiKeStack/be-comment/pkg/logger"
	"github.com/MuxiKeStack/be-comment/repository"
	"strconv"
	"time"
)

var (
	ErrCommentNotFound = repository.ErrCommentNotFound
	ErrInvalidBiz      = errors.New("创建的评论所属biz无效")
)

type CommentService interface {
	CreateComment(ctx context.Context, comment domain.Comment) error
	GetCommentList(ctx context.Context, biz commentv1.Biz, bizId int64, curCommentId int64, limit int64) ([]domain.Comment, error)
	DeleteComment(ctx context.Context, commentId int64, uid int64) error
	GetMoreReplies(ctx context.Context, rid int64, curCommentId int64, limit int64) ([]domain.Comment, error)
	Count(ctx context.Context, biz commentv1.Biz, bizId int64) (int64, error)
	GetComment(ctx context.Context, commentId int64) (domain.Comment, error)
}

type commentService struct {
	repo       repository.CommentRepository
	uidGetters map[commentv1.Biz]UIDGetter
	producer   events.Producer
	l          logger.Logger
}

func NewCommentService(repo repository.CommentRepository, producer events.Producer, evaluationClient evaluationv1.EvaluationServiceClient,
	answerClient answerv1.AnswerServiceClient, l logger.Logger) CommentService {
	return &commentService{
		repo: repo,
		uidGetters: map[commentv1.Biz]UIDGetter{
			commentv1.Biz_Evaluation: &EvaluationUIDGetter{evaluationClient: evaluationClient},
			commentv1.Biz_Answer:     &AnswerUIDGetter{answerClient: answerClient},
		},
		producer: producer,
		l:        l,
	}
}

func (s *commentService) GetCommentList(ctx context.Context, biz commentv1.Biz, bizId int64, curCommentId int64, limit int64) ([]domain.Comment, error) {
	list, err := s.repo.FindByBiz(ctx, biz, bizId, curCommentId, limit)
	if err != nil {
		return nil, err
	}
	return list, err
}

func (s *commentService) DeleteComment(ctx context.Context, commentId int64, uid int64) error {
	return s.repo.DeleteComment(ctx, commentId, uid)
}

func (s *commentService) Count(ctx context.Context, biz commentv1.Biz, bizId int64) (int64, error) {
	return s.repo.GetCountByBiz(ctx, biz, bizId)
}

func (s *commentService) GetMoreReplies(ctx context.Context, rid int64, curCommentId int64, limit int64) ([]domain.Comment, error) {
	return s.repo.GetMoreReplies(ctx, rid, curCommentId, limit)
}

func (s *commentService) CreateComment(ctx context.Context, comment domain.Comment) error {
	// 自己是根评论，reply to biz的owner
	getter, ok := s.uidGetters[comment.Biz]
	if !ok {
		return ErrInvalidBiz
	}
	publisherId, err := getter.GetUID(ctx, comment.BizId)
	if err != nil {
		return err
	}
	// 要去聚合一下 replyToUid
	if comment.ParentComment.Id != 0 {
		// 有父评论，找到父评论的发布者
		pc, er := s.repo.FindById(ctx, comment.ParentComment.Id)
		if er != nil {
			return er
		}
		comment.ReplyToUid = pc.Commentator.ID
	} else {
		comment.ReplyToUid = publisherId
	}
	err = s.repo.CreateComment(ctx, comment)
	if err != nil {
		return err
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()
		er := s.producer.ProduceFeedEvent(ctx, events.FeedEvent{
			Type: feedv1.EventType_Comment,
			Metadata: map[string]string{
				// 评论者
				"commentator": strconv.FormatInt(comment.Commentator.ID, 10),
				// 被评论者
				"recipient": strconv.FormatInt(comment.ReplyToUid, 10),
				// 资源发布者，可能与被评论者相同
				"bizPublisher": strconv.FormatInt(publisherId, 10),
				"biz":          comment.Biz.String(),
				"bizId":        strconv.FormatInt(comment.BizId, 10),
				"commentId":    strconv.FormatInt(comment.Id, 10),
			},
		})
		if er != nil {
			s.l.Error("发送评论事件失败",
				logger.Error(er),
				logger.Int64("commentator", comment.Commentator.ID),
				logger.Int64("recipient", comment.ReplyToUid),
				logger.Int64("bizPublisher", publisherId),
				logger.String("biz", comment.Biz.String()),
				logger.Int64("bizId", comment.BizId),
				logger.Int64("commentId", comment.Id))
		}
	}()

	return nil
}

func (s *commentService) GetComment(ctx context.Context, commentId int64) (domain.Comment, error) {
	return s.repo.FindById(ctx, commentId)
}

func NewCommentSvc(repo repository.CommentRepository) CommentService {
	return &commentService{
		repo: repo,
	}
}
