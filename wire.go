//go:build wireinject

package main

import (
	"github.com/MuxiKeStack/be-comment/grpc"
	"github.com/MuxiKeStack/be-comment/ioc"
	"github.com/MuxiKeStack/be-comment/pkg/grpcx"
	"github.com/MuxiKeStack/be-comment/repository"
	"github.com/MuxiKeStack/be-comment/repository/dao"
	"github.com/MuxiKeStack/be-comment/service"
	"github.com/google/wire"
)

func InitGRPCServer() grpcx.Server {
	wire.Build(
		ioc.InitGRPCxKratosServer,
		grpc.NewCommentServiceServer,
		service.NewCommentService,
		repository.NewCachedCommentRepo,
		dao.NewCommentDAO,
		ioc.InitEtcdClient,
		ioc.InitDB,
		ioc.InitLogger,
	)
	return grpcx.Server(nil)
}
