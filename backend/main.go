package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	ai "epistemic-me-backend/ai"
	db "epistemic-me-backend/db"
	pb "epistemic-me-backend/pb"
	models "epistemic-me-backend/pb/models"
	"epistemic-me-backend/pb/pbconnect"
	svc "epistemic-me-backend/svc"
	svcmodels "epistemic-me-backend/svc/models"
)

type server struct {
	bsvc *svc.BeliefService
	dsvc *svc.DialecticService
}

func (s *server) CreateBelief(
	ctx context.Context,
	req *connect.Request[pb.CreateBeliefRequest],
) (*connect.Response[pb.CreateBeliefResponse], error) {
	log.Printf("CreateBelief called with request: %+v", req.Msg)

	input := &svcmodels.CreateBeliefInput{
		UserID:        req.Msg.UserId,
		BeliefContent: req.Msg.BeliefContent,
	}
	log.Printf("CreateBelief input: %+v", input)

	response, err := s.bsvc.CreateBelief(input)
	if err != nil {
		log.Printf("CreateBelief ERROR: %v", err)
		return nil, err
	}

	log.Printf("CreateBelief response: %+v", response)

	// Ensure belief system is created
	if response.BeliefSystem.Beliefs == nil {
		response.BeliefSystem.Beliefs = []*svcmodels.Belief{&response.Belief}
	}

	protoResponse := &pb.CreateBeliefResponse{
		Belief:       response.Belief.ToProto(),
		BeliefSystem: response.BeliefSystem.ToProto(),
	}
	log.Printf("CreateBelief proto response: %+v", protoResponse)

	return connect.NewResponse(protoResponse), nil
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
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if response == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unexpected nil response"))
	}

	var beliefPbs []*models.Belief
	for _, belief := range response.Beliefs {
		beliefPbs = append(beliefPbs, belief.ToProto())
	}

	protoResponse := &pb.ListBeliefsResponse{
		Beliefs:      beliefPbs,
		BeliefSystem: response.BeliefSystem.ToProto(),
	}

	return connect.NewResponse(protoResponse), nil
}

func (s *server) CreateDialectic(ctx context.Context, req *connect.Request[pb.CreateDialecticRequest]) (*connect.Response[pb.CreateDialecticResponse], error) {
	log.Printf("CreateDialectic called with request: %+v", req.Msg)

	input := &svcmodels.CreateDialecticInput{
		UserID:        req.Msg.UserId,
		DialecticType: svcmodels.DialecticType(req.Msg.DialecticType),
	}
	log.Printf("CreateDialectic input: %+v", input)

	response, err := s.dsvc.CreateDialectic(input)
	if err != nil {
		log.Printf("CreateDialectic ERROR: %v", err)
		return nil, err
	}

	log.Printf("CreateDialectic response: %+v", response)

	protoResponse := &pb.CreateDialecticResponse{
		DialecticId: response.DialecticID,
		Dialectic:   response.Dialectic.ToProto(),
	}
	log.Printf("CreateDialectic proto response: %+v", protoResponse)

	return connect.NewResponse(protoResponse), nil
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
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if response == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unexpected nil response"))
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
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if response == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unexpected nil response"))
	}

	protoResponse := &pb.UpdateDialecticResponse{
		Dialectic: response.Dialectic.ToProto(),
	}

	return connect.NewResponse(protoResponse), nil
}

func (s *server) GetBeliefSystemDetail(
	ctx context.Context,
	req *connect.Request[pb.GetBeliefSystemDetailRequest],
) (*connect.Response[pb.GetBeliefSystemDetailResponse], error) {
	log.Printf("GetBeliefSystemDetail called for user: %s", req.Msg.UserId)

	response, err := s.bsvc.GetBeliefSystemDetail(&svcmodels.GetBeliefSystemDetailInput{
		UserID:                       req.Msg.UserId,
		CurrentObservationContextIds: req.Msg.CurrentObservationContextIds,
	})

	if err != nil {
		log.Printf("Error in GetBeliefSystemDetail: %v", err)
		if err.Error() == "no belief systems found for user" || err.Error() == "error retrieving beliefs: user not found" {
			log.Printf("No belief system found for user: %s", req.Msg.UserId)
			// Return an empty belief system instead of an error
			return connect.NewResponse(&pb.GetBeliefSystemDetailResponse{
				BeliefSystemDetail: &models.BeliefSystemDetail{
					ExampleName: "Empty Belief System",
					BeliefSystem: &models.BeliefSystem{
						Beliefs:             []*models.Belief{},
						ObservationContexts: []*models.ObservationContext{},
					},
				},
			}), nil
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if response == nil {
		log.Printf("GetBeliefSystemDetail response is nil for user: %s", req.Msg.UserId)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unexpected nil response"))
	}

	log.Printf("Retrieved belief system for user %s: %+v", req.Msg.UserId, response)

	return connect.NewResponse(&pb.GetBeliefSystemDetailResponse{
		BeliefSystemDetail: response.ToProto(),
	}), nil
}

// Add this method to your server type
func (s *server) UpdateKeyValueStore(ctx context.Context, req *connect.Request[pb.UpdateKeyValueStoreRequest]) (*connect.Response[pb.UpdateKeyValueStoreResponse], error) {
	// Implement the logic for updating the key-value store
	// For now, return a placeholder response
	return connect.NewResponse(&pb.UpdateKeyValueStoreResponse{}), nil
}

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatalf("OPENAI_API_KEY environment variable not set")
	}

	aih := ai.NewAIHelper(apiKey)

	// Create a new KeyValueStore
	kvStore, err := db.NewKeyValueStore("./epistemic_me.json") // Use a JSON file for persistence
	if err != nil {
		log.Printf("Warning: Failed to create KeyValueStore: %v", err)
		log.Println("Continuing with in-memory storage. Data will not be persisted.")
		// Create an in-memory store as a fallback
		kvStore, err = db.NewKeyValueStore("")
		if err != nil {
			log.Fatalf("Failed to create in-memory KeyValueStore: %v", err)
		}
	}
	log.Println("Successfully created KeyValueStore")

	bsvc := svc.NewBeliefService(kvStore, aih)
	dsvc := svc.NewDialecticService(kvStore, bsvc, aih)

	svcServer := &server{
		bsvc: bsvc,
		dsvc: dsvc,
	}

	mux := http.NewServeMux()
	path, handler := pbconnect.NewEpistemicMeServiceHandler(svcServer)
	mux.Handle(path, handler)

	// Configure CORS
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8081", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
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
