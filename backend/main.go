package main

import (
	"context"
	"log"
	"net"
	"net/http"

	"connectrpc.com/connect"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	pb "epistemic-me-backend/pb"
	models "epistemic-me-backend/pb/models"
	"epistemic-me-backend/pb/pbconnect"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// server is used to implement the EpistemicMeService.
type grpcServer struct {
	pb.UnimplementedEpistemicMeServiceServer
}

// gRPC methods
func (s *grpcServer) CreateBelief(ctx context.Context, req *pb.CreateBeliefRequest) (*pb.CreateBeliefResponse, error) {
	// Mock response
	return &pb.CreateBeliefResponse{
		Belief: &models.Belief{
			Id:     "mock-belief-id",
			UserId: req.UserId,
		},
	}, nil
}

func (s *grpcServer) ListBeliefs(ctx context.Context, req *pb.ListBeliefsRequest) (*pb.ListBeliefsResponse, error) {
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

func (s *grpcServer) CreateDialectic(ctx context.Context, req *pb.CreateDialecticRequest) (*pb.CreateDialecticResponse, error) {
	// Mock response
	return &pb.CreateDialecticResponse{
		DialecticId: "mock-dialectic-id",
		Dialectic: &models.Dialectic{
			Id:     "mock-dialectic-id",
			UserId: req.UserId,
		},
	}, nil
}

func (s *grpcServer) ListDialectics(ctx context.Context, req *pb.ListDialecticsRequest) (*pb.ListDialecticsResponse, error) {
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

func (s *grpcServer) UpdateDialectic(ctx context.Context, req *pb.UpdateDialecticRequest) (*pb.UpdateDialecticResponse, error) {
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

// connectServer is used to implement the Connect RPC service.
type connectServer struct{}

func (s *connectServer) CreateBelief(ctx context.Context, req *connect.Request[pb.CreateBeliefRequest]) (*connect.Response[pb.CreateBeliefResponse], error) {
	log.Println("CreateBelief called with request:", req.Msg)
	// Mock response
	return connect.NewResponse(&pb.CreateBeliefResponse{
		Belief: &models.Belief{
			Id:     "mock-belief-id",
			UserId: req.Msg.UserId,
		},
	}), nil
}

func (s *connectServer) ListBeliefs(ctx context.Context, req *connect.Request[pb.ListBeliefsRequest]) (*connect.Response[pb.ListBeliefsResponse], error) {
	log.Println("ListBeliefs called with request:", req.Msg)
	// Mock response
	return connect.NewResponse(&pb.ListBeliefsResponse{
		Beliefs: []*models.Belief{
			{
				Id:     "mock-belief-id",
				UserId: req.Msg.UserId,
			},
		},
	}), nil
}

func (s *connectServer) CreateDialectic(ctx context.Context, req *connect.Request[pb.CreateDialecticRequest]) (*connect.Response[pb.CreateDialecticResponse], error) {
	log.Println("CreateDialectic called with request:", req.Msg)
	// Mock response
	return connect.NewResponse(&pb.CreateDialecticResponse{
		DialecticId: "mock-dialectic-id",
		Dialectic: &models.Dialectic{
			Id:     "mock-dialectic-id",
			UserId: req.Msg.UserId,
		},
	}), nil
}

func (s *connectServer) ListDialectics(ctx context.Context, req *connect.Request[pb.ListDialecticsRequest]) (*connect.Response[pb.ListDialecticsResponse], error) {
	log.Println("ListDialectics called with request:", req.Msg)
	// Mock response
	return connect.NewResponse(&pb.ListDialecticsResponse{
		Dialectics: []*models.Dialectic{
			{
				Id:     "mock-dialectic-id",
				UserId: req.Msg.UserId,
			},
		},
	}), nil
}

func (s *connectServer) UpdateDialectic(ctx context.Context, req *connect.Request[pb.UpdateDialecticRequest]) (*connect.Response[pb.UpdateDialecticResponse], error) {
	log.Println("UpdateDialectic called with request:", req.Msg)
	// Mock response
	return connect.NewResponse(&pb.UpdateDialecticResponse{
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
	}), nil
}

func main() {
	grpcSrv := grpc.NewServer()
	pb.RegisterEpistemicMeServiceServer(grpcSrv, &grpcServer{})

	// Register reflection service on gRPC server.
	reflection.Register(grpcSrv)

	// Create a TCP listener for gRPC
	grpcListener, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("failed to listen on port 9090: %v", err)
	}

	go func() {
		log.Println("Starting gRPC server on port 9090")
		if err := grpcSrv.Serve(grpcListener); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	// Set up the Connect server
	connectSvc := &connectServer{}
	mux := http.NewServeMux()
	path, handler := pbconnect.NewEpistemicMeServiceHandler(connectSvc)
	mux.Handle(path, handler)

	// Configure CORS
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8081"},
		AllowedMethods:   []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "Connect-Protocol-Version"},
		ExposedHeaders:   []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		Debug:            true,
	})

	log.Println("Server is running on port 8080 for Connect")
	http.ListenAndServe(
		":8080",
		h2c.NewHandler(corsHandler.Handler(mux), &http2.Server{}),
	)
}
