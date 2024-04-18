// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"github.com/MuxiKeStack/be-comment/grpc"
	"github.com/MuxiKeStack/be-comment/ioc"
	"github.com/MuxiKeStack/be-comment/pkg/grpcx"
	"github.com/MuxiKeStack/be-comment/repository"
	"github.com/MuxiKeStack/be-comment/repository/dao"
	"github.com/MuxiKeStack/be-comment/service"
)

// Injectors from wire.go:

func InitGRPCServer() grpcx.Server {
	logger := ioc.InitLogger()
	db := ioc.InitDB(logger)
	commentDAO := dao.NewCommentDAO(db)
	commentRepository := repository.NewCachedCommentRepo(commentDAO, logger)
	commentService := service.NewCommentService(commentRepository)
	commentServiceServer := grpc.NewCommentServiceServer(commentService)
	client := ioc.InitEtcdClient()
	server := ioc.InitGRPCxKratosServer(commentServiceServer, client, logger)
	return server
}
