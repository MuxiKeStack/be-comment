package grpc

import (
	"context"
	commentv1 "github.com/MuxiKeStack/be-api/gen/proto/comment/v1"
	"github.com/MuxiKeStack/be-comment/domain"
	"github.com/MuxiKeStack/be-comment/service"
	"google.golang.org/grpc"
	"math"
)

type CommentServiceServer struct {
	svc service.CommentService
	commentv1.UnimplementedCommentServiceServer
}

func NewCommentServiceServer(svc service.CommentService) *CommentServiceServer {
	return &CommentServiceServer{svc: svc}
}

func (s *CommentServiceServer) Register(server grpc.ServiceRegistrar) {
	commentv1.RegisterCommentServiceServer(server, s)
}

func (s *CommentServiceServer) GetCommentList(ctx context.Context, request *commentv1.CommentListRequest) (*commentv1.CommentListResponse, error) {
	curCommentId := request.GetCurCommentId()
	// 第一次查询
	if curCommentId <= 0 {
		curCommentId = math.MaxInt64
	}
	domainComments, err := s.svc.
		GetCommentList(ctx,
			request.GetBiz(),
			request.GetBizId(),
			curCommentId,
			request.GetLimit())
	if err != nil {
		return nil, err
	}
	return &commentv1.CommentListResponse{
		Comments: s.toDTO(domainComments),
	}, nil
}

func (s *CommentServiceServer) DeleteComment(ctx context.Context, request *commentv1.DeleteCommentRequest) (*commentv1.DeleteCommentResponse, error) {
	err := s.svc.DeleteComment(ctx, request.GetCommentId(), request.GetUid())
	return &commentv1.DeleteCommentResponse{}, err
}

// TODO 缺少外键约束，无法避免的会有错误的bizId，目前没有解决，其他地方也有这样的问题，question create
func (s *CommentServiceServer) CreateComment(ctx context.Context, request *commentv1.CreateCommentRequest) (*commentv1.CreateCommentResponse, error) {
	err := s.svc.CreateComment(ctx, convertToDomain(request.GetComment()))
	return &commentv1.CreateCommentResponse{}, err
}

func (s *CommentServiceServer) GetMoreReplies(ctx context.Context, request *commentv1.GetMoreRepliesRequest) (*commentv1.GetMoreRepliesResponse, error) {
	cs, err := s.svc.GetMoreReplies(ctx, request.GetRid(), request.GetCurCommentId(), request.GetLimit())
	if err != nil {
		return nil, err
	}
	return &commentv1.GetMoreRepliesResponse{
		Replies: s.toDTO(cs),
	}, nil
}

func (s *CommentServiceServer) CountComment(ctx context.Context, request *commentv1.CountCommentRequest) (*commentv1.CountCommentResponse, error) {
	count, err := s.svc.Count(ctx, request.GetBiz(), request.GetBizId())
	return &commentv1.CountCommentResponse{
		Count: count,
	}, err
}

func convertToDomain(comment *commentv1.Comment) domain.Comment {
	domainComment := domain.Comment{
		Id: comment.GetId(),
		Commentator: domain.User{
			ID: comment.GetCommentatorId(),
		},
		Biz:        comment.GetBiz(),
		BizId:      comment.GetBizId(),
		Content:    comment.Content,
		ReplyToUid: comment.GetReplyToUid(),
	}
	if comment.GetParentComment() != nil {
		domainComment.ParentComment = &domain.Comment{
			Id: comment.GetParentComment().GetId(),
		}
	}
	if comment.GetRootComment() != nil {
		domainComment.RootComment = &domain.Comment{
			Id: comment.GetRootComment().GetId(),
		}
	}
	return domainComment
}

func (s *CommentServiceServer) toDTO(domainComments []domain.Comment) []*commentv1.Comment {
	rpcComments := make([]*commentv1.Comment, 0, len(domainComments))
	for _, domainComment := range domainComments {
		rpcComment := &commentv1.Comment{
			Id:            domainComment.Id,
			CommentatorId: domainComment.Commentator.ID,
			Biz:           domainComment.Biz,
			BizId:         domainComment.BizId,
			Content:       domainComment.Content,
			ReplyToUid:    domainComment.ReplyToUid,
			Ctime:         domainComment.CTime.UnixMilli(),
			Utime:         domainComment.UTime.UnixMilli(),
		}
		if domainComment.RootComment != nil {
			rpcComment.RootComment = &commentv1.Comment{
				Id: domainComment.RootComment.Id,
			}
		}
		if domainComment.ParentComment != nil {
			rpcComment.ParentComment = &commentv1.Comment{
				Id: domainComment.ParentComment.Id,
			}
		}
		rpcComments = append(rpcComments, rpcComment)
	}
	rpcCommentMap := make(map[int64]*commentv1.Comment, len(rpcComments))
	for _, rpcComment := range rpcComments {
		rpcCommentMap[rpcComment.Id] = rpcComment
	}
	for _, domainComment := range domainComments {
		rpcComment := rpcCommentMap[domainComment.Id]
		if domainComment.RootComment != nil {
			val, ok := rpcCommentMap[domainComment.RootComment.Id]
			if ok {
				rpcComment.RootComment = val
			}
		}
		if domainComment.ParentComment != nil {
			val, ok := rpcCommentMap[domainComment.ParentComment.Id]
			if ok {
				rpcComment.ParentComment = val
			}
		}
	}
	return rpcComments
}
