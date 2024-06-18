package main

import (
	"context"
	"log"
	"net/http"

	"connectrpc.com/connect"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	pb "epistemic-me-backend/pb"
	models "epistemic-me-backend/pb/models"
	"epistemic-me-backend/pb/pbconnect" // Import generated Connect Go code
)

// server is used to implement the EpistemicMeService.
type server struct{}

func (s *server) CreateBelief(
	ctx context.Context,
	req *connect.Request[pb.CreateBeliefRequest],
) (*connect.Response[pb.CreateBeliefResponse], error) {
	log.Println("CreateBelief called with request:", req.Msg)
	// Mock response
	return connect.NewResponse(&pb.CreateBeliefResponse{
		Belief: &models.Belief{
			Id:     "mock-belief-id",
			UserId: req.Msg.UserId,
		},
	}), nil
}

func (s *server) ListBeliefs(
	ctx context.Context,
	req *connect.Request[pb.ListBeliefsRequest],
) (*connect.Response[pb.ListBeliefsResponse], error) {
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

func (s *server) CreateDialectic(
	ctx context.Context,
	req *connect.Request[pb.CreateDialecticRequest],
) (*connect.Response[pb.CreateDialecticResponse], error) {
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

func (s *server) ListDialectics(
	ctx context.Context,
	req *connect.Request[pb.ListDialecticsRequest],
) (*connect.Response[pb.ListDialecticsResponse], error) {
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

func (s *server) UpdateDialectic(
	ctx context.Context,
	req *connect.Request[pb.UpdateDialecticRequest],
) (*connect.Response[pb.UpdateDialecticResponse], error) {
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
	svc := &server{}
	mux := http.NewServeMux()
	path, handler := pbconnect.NewEpistemicMeServiceHandler(svc)
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
		// Use h2c so we can serve HTTP/2 without TLS.
		h2c.NewHandler(corsHandler.Handler(mux), &http2.Server{}),
	)
}
