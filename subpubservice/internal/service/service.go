package service

import (
	"context"
	//	"errors"
	"log/slog"

	"projectGo2/vkAlg/app/subpub"
	pb "projectGo2/vkAlg/subpubservice/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type PubSubService struct {
	pb.UnimplementedPubSubServer
	bus    *subpub.SubPub
	logger *slog.Logger
}

func NewPubSubService(bus *subpub.SubPub, logger *slog.Logger) *PubSubService {
	return &PubSubService{
		bus:    bus,
		logger: logger,
	}
}

func (s *PubSubService) Subscribe(req *pb.SubscribeRequest, stream pb.PubSub_SubscribeServer) error {
	ctx := stream.Context()
	key := req.GetKey()

	s.logger.InfoContext(ctx, "New subscriber", "key", key)

	sub, err := s.bus.Subscribe(key, func(msg interface{}) {
		data, ok := msg.(string)
		if !ok {
			s.logger.ErrorContext(ctx, "Invalid message type", "key", key)
			return
		}

		if err := stream.Send(&pb.Event{Data: data}); err != nil {
			s.logger.ErrorContext(ctx, "Failed to send event", "error", err)
		}
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	// Ждем завершения контекста
	<-ctx.Done()
	s.logger.InfoContext(ctx, "Subscription closed", "key", key)

	return nil
}

func (s *PubSubService) Publish(ctx context.Context, req *pb.PublishRequest) (*emptypb.Empty, error) {
	key := req.GetKey()
	data := req.GetData()

	if err := s.bus.Publish(key, data); err != nil {
		s.logger.ErrorContext(ctx, "Publish failed", "key", key, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to publish: %v", err)
	}

	s.logger.DebugContext(ctx, "Message published", "key", key, "data", data)
	return &emptypb.Empty{}, nil
}
