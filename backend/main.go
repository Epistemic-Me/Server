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
	svc "epistemic-me-backend/svc"
	svcmodels "epistemic-me-backend/svc/models"
)

// server is used to implement the EpistemicMeService.
type server struct {
	bsvc *svc.BeliefService
	dsvc *svc.DialecticService
}

func (s *server) CreateBelief(
	ctx context.Context,
	req *connect.Request[pb.CreateBeliefRequest],
) (*connect.Response[pb.CreateBeliefResponse], error) {
	log.Println("CreateBelief called with request:", req.Msg)

	response, err := s.bsvc.CreateBelief(&svcmodels.CreateBeliefInput{
		UserID:        req.Msg.UserId,
		BeliefContent: req.Msg.BeliefContent,
	})

	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&pb.CreateBeliefResponse{
		Belief:       response.Belief.ToProto(),
		BeliefSystem: response.BeliefSystem.ToProto(),
	}), nil
}

func (s *server) ListBeliefs(
	ctx context.Context,
	req *connect.Request[pb.ListBeliefsRequest],
) (*connect.Response[pb.ListBeliefsResponse], error) {
	log.Println("ListBeliefs called with request:", req.Msg)

	response, err := s.bsvc.ListBeliefs(&svcmodels.ListBeliefsInput{
		UserID:    req.Msg.UserId,
		BeliefIDs: req.Msg.BeliefIds,
	})

	if err != nil {
		return nil, err
	}

	var beliefPbs []*models.Belief
	for _, belief := range response.Beliefs {
		beliefPbs = append(beliefPbs, belief.ToProto())
	}

	return connect.NewResponse(&pb.ListBeliefsResponse{
		Beliefs:      beliefPbs,
		BeliefSystem: response.BeliefSystem.ToProto(),
	}), nil
}

func (s *server) CreateDialectic(
	ctx context.Context,
	req *connect.Request[pb.CreateDialecticRequest],
) (*connect.Response[pb.CreateDialecticResponse], error) {
	log.Println("CreateDialectic called with request:", req.Msg)

	response, err := s.dsvc.CreateDialectic(&svcmodels.CreateDialecticInput{
		UserID:        req.Msg.UserId,
		DialecticType: svcmodels.DialecticType(req.Msg.DialecticType),
	})

	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&pb.CreateDialecticResponse{
		DialecticId: response.DialecticID,
		Dialectic:   response.Dialectic.ToProto(),
	}), nil
}

func (s *server) ListDialectics(
	ctx context.Context,
	req *connect.Request[pb.ListDialecticsRequest],
) (*connect.Response[pb.ListDialecticsResponse], error) {
	log.Println("ListDialectics called with request:", req.Msg)

	response, err := s.dsvc.ListDialectics(&svcmodels.ListDialecticsInput{
		UserID: req.Msg.UserId,
	})

	if err != nil {
		return nil, err
	}

	var dialecticPbs []*models.Dialectic
	for _, dialectic := range response.Dialectics {
		dialecticPbs = append(dialecticPbs, dialectic.ToProto())
	}

	return connect.NewResponse(&pb.ListDialecticsResponse{
		Dialectics: dialecticPbs,
	}), nil
}

func (s *server) UpdateDialectic(
	ctx context.Context,
	req *connect.Request[pb.UpdateDialecticRequest],
) (*connect.Response[pb.UpdateDialecticResponse], error) {
	log.Println("UpdateDialectic called with request:", req.Msg)

	response, err := s.dsvc.UpdateDialectic(&svcmodels.UpdateDialecticInput{
		UserID:      req.Msg.UserId,
		DialecticID: req.Msg.DialecticId,
		Answer: svcmodels.UserAnswer{
			UserAnswer:         req.Msg.Answer.UserAnswer,
			CreatedAtMillisUTC: req.Msg.Answer.CreatedAtMillisUtc,
		},
	})

	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&pb.UpdateDialecticResponse{
		Dialectic: response.Dialectic.ToProto(),
	}), nil
}

func main() {
	svc := &server{
		bsvc: svc.NewBeliefService(),    // Initialize the BeliefService
		dsvc: svc.NewDialecticService(), // Initialize the DialecticService
	}
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
