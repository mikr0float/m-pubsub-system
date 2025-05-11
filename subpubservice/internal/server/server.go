package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mikr0float/m-pubsub-system/subpub"
	"github.com/mikr0float/m-pubsub-system/subpubservice/config"
	"github.com/mikr0float/m-pubsub-system/subpubservice/internal/service"
	pb "github.com/mikr0float/m-pubsub-system/subpubservice/proto"

	"google.golang.org/grpc"
)

type Server struct {
	grpcServer *grpc.Server
	config     *config.Config
	logger     *slog.Logger
	bus        *subpub.SubPub
}

func New(cfg *config.Config, logger *slog.Logger) *Server {
	return &Server{
		config: cfg,
		logger: logger,
		bus:    subpub.NewSubPub(),
	}
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.GRPCPort))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.grpcServer = grpc.NewServer(
		grpc.UnaryInterceptor(s.loggingUnaryInterceptor),
		grpc.StreamInterceptor(s.loggingStreamInterceptor),
	)

	pb.RegisterPubSubServer(s.grpcServer, service.NewPubSubService(s.bus, s.logger))

	s.logger.Info("Starting gRPC server", "port", s.config.GRPCPort)

	go func() {
		if err := s.grpcServer.Serve(lis); err != nil {
			s.logger.Error("gRPC server failed", "error", err)
		}
	}()

	return nil
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		s.logger.Info("Server stopped gracefully")
		return nil
	case <-ctx.Done():
		s.grpcServer.Stop()
		s.logger.Warn("Server forced to stop")
		return ctx.Err()
	}
}

func (s *Server) loggingUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	start := time.Now()
	resp, err = handler(ctx, req)
	s.logger.Info("Unary call",
		"method", info.FullMethod,
		"duration", time.Since(start),
		"error", err,
	)
	return
}

func (s *Server) loggingStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	start := time.Now()
	err := handler(srv, ss)
	s.logger.Info("Stream call",
		"method", info.FullMethod,
		"duration", time.Since(start),
		"error", err,
	)
	return err
}

func (s *Server) WaitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("Shutting down server...")
	if err := s.Stop(); err != nil {
		s.logger.Error("Failed to stop server", "error", err)
	}
}
