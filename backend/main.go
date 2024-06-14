package main

import (
	"context"
	"log"
	"net"

	pb "epistemic-me-backend/pb"
	models "epistemic-me-backend/pb/models"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// server is used to implement the EpistemicMeService.
type server struct {
	pb.EpistemicMeServiceServer
}

func (s *server) CreateBelief(ctx context.Context, req *pb.CreateBeliefRequest) (*pb.CreateBeliefResponse, error) {
	// Mock response
	return &pb.CreateBeliefResponse{
		Belief: &models.Belief{
			Id:     "mock-belief-id",
			UserId: req.UserId,
		},
	}, nil
}

func (s *server) ListBeliefs(ctx context.Context, req *pb.ListBeliefsRequest) (*pb.ListBeliefsResponse, error) {
	// Mock response
	return &pb.ListBeliefsResponse{
		Beliefs: []*models.Belief{
			{
				Id:     "mock-belief-id",
				UserId: req.UserId,
			},
		},
	}, nil
}

func (s *server) CreateDialectic(ctx context.Context, req *pb.CreateDialecticRequest) (*pb.CreateDialecticResponse, error) {
	// Mock response
	return &pb.CreateDialecticResponse{
		DialecticId: "mock-dialectic-id",
		Dialectic: &models.Dialectic{
			Id:     "mock-dialectic-id",
			UserId: req.UserId,
		},
	}, nil
}

func (s *server) ListDialectics(ctx context.Context, req *pb.ListDialecticsRequest) (*pb.ListDialecticsResponse, error) {
	// Mock response
	return &pb.ListDialecticsResponse{
		Dialectics: []*models.Dialectic{
			{
				Id:     "mock-dialectic-id",
				UserId: req.UserId,
			},
		},
	}, nil
}

func (s *server) UpdateDialectic(ctx context.Context, req *pb.UpdateDialecticRequest) (*pb.UpdateDialecticResponse, error) {
	// Mock response
	return &pb.UpdateDialecticResponse{
		Dialectic: &models.Dialectic{
			Id: "mock-dialectic-id",
			UserInteractions: []*models.DialecticalInteraction{
				{
					Question: &models.Question{
						Question: "Mock question",
					},
					UserAnswer: &models.UserAnswer{
						UserAnswer: "Mock answer",
					},
				},
			},
		},
	}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()

	// Register reflection service on gRPC server.
	reflection.Register(s)

	pb.RegisterEpistemicMeServiceServer(s, &server{})
	log.Println("Server is running on port :8080")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
