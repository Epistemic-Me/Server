package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"

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

type Server struct {
	bsvc         *svc.BeliefService
	dsvc         *svc.DialecticService
	kvStore      *db.KeyValueStore // Change this to a pointer
	selfModelSvc *svc.SelfModelService
}

func (s *Server) CreateBelief(
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

func (s *Server) ListBeliefs(
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

func (s *Server) CreateDialectic(ctx context.Context, req *connect.Request[pb.CreateDialecticRequest]) (*connect.Response[pb.CreateDialecticResponse], error) {
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

func (s *Server) ListDialectics(
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

func (s *Server) UpdateDialectic(
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
		DryRun: req.Msg.DryRun,
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

func (s *Server) GetBeliefSystemDetail(
	ctx context.Context,
	req *connect.Request[pb.GetBeliefSystemDetailRequest],
) (*connect.Response[pb.GetBeliefSystemDetailResponse], error) {
	log.Printf("GetBeliefSystemDetail called for user: %s", req.Msg.UserId)

	// Log the state of the KeyValueStore before retrieving the belief system
	log.Printf("KeyValueStore state before retrieval: %+v", s.kvStore)

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
	log.Printf("Number of beliefs in response: %d", len(response.BeliefSystem.Beliefs))
	log.Printf("Number of observation contexts in response: %d", len(response.BeliefSystem.ObservationContexts))

	// Log the first few beliefs if any exist
	if len(response.BeliefSystem.Beliefs) > 0 {
		log.Printf("First belief: %+v", response.BeliefSystem.Beliefs[0])
	}

	protoResponse := &pb.GetBeliefSystemDetailResponse{
		BeliefSystemDetail: response.ToProto(),
	}

	log.Printf("Number of beliefs in proto response: %d", len(protoResponse.BeliefSystemDetail.BeliefSystem.Beliefs))
	log.Printf("Number of observation contexts in proto response: %d", len(protoResponse.BeliefSystemDetail.BeliefSystem.ObservationContexts))

	// Log the first few beliefs in the proto response if any exist
	if len(protoResponse.BeliefSystemDetail.BeliefSystem.Beliefs) > 0 {
		log.Printf("First belief in proto response: %+v", protoResponse.BeliefSystemDetail.BeliefSystem.Beliefs[0])
	}

	return connect.NewResponse(protoResponse), nil
}

// Add this method to your server type
func (s *Server) UpdateKeyValueStore(ctx context.Context, req *connect.Request[pb.UpdateKeyValueStoreRequest]) (*connect.Response[pb.UpdateKeyValueStoreResponse], error) {
	// Implement the logic for updating the key-value store
	// For now, return a placeholder response
	return connect.NewResponse(&pb.UpdateKeyValueStoreResponse{}), nil
}

func (s *Server) CreateSelfModel(ctx context.Context, req *connect.Request[pb.CreateSelfModelRequest]) (*connect.Response[pb.CreateSelfModelResponse], error) {
	input := &svcmodels.CreateSelfModelInput{
		ID:           req.Msg.Id,
		Philosophies: req.Msg.Philosophies,
	}
	resp, err := s.selfModelSvc.CreateSelfModel(ctx, input)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pb.CreateSelfModelResponse{
		SelfModel: resp.SelfModel.ToProto(),
	}), nil
}

func (s *Server) GetSelfModel(ctx context.Context, req *connect.Request[pb.GetSelfModelRequest]) (*connect.Response[pb.GetSelfModelResponse], error) {
	input := &svcmodels.GetSelfModelInput{
		SelfModelID: req.Msg.SelfModelId,
	}
	resp, err := s.selfModelSvc.GetSelfModel(ctx, input)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pb.GetSelfModelResponse{
		SelfModel: resp.SelfModel.ToProto(),
	}), nil
}

func (s *Server) AddPhilosophy(ctx context.Context, req *connect.Request[pb.AddPhilosophyRequest]) (*connect.Response[pb.AddPhilosophyResponse], error) {
	input := &svcmodels.AddPhilosophyInput{
		SelfModelID:  req.Msg.SelfModelId,
		PhilosophyID: req.Msg.PhilosophyId,
	}
	resp, err := s.selfModelSvc.AddPhilosophy(ctx, input)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pb.AddPhilosophyResponse{
		UpdatedSelfModel: resp.UpdatedSelfModel.ToProto(),
	}), nil
}

func (s *Server) CreatePhilosophy(ctx context.Context, req *connect.Request[pb.CreatePhilosophyRequest]) (*connect.Response[pb.CreatePhilosophyResponse], error) {
	input := &svcmodels.CreatePhilosophyInput{
		Description:         req.Msg.Description,
		ExtrapolateContexts: req.Msg.ExtrapolateContexts,
	}
	resp, err := s.selfModelSvc.CreatePhilosophy(ctx, input)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pb.CreatePhilosophyResponse{
		Philosophy: resp.Philosophy.ToProto(),
	}), nil
}

func NewServer(kvStore *db.KeyValueStore) *Server {
	if kvStore == nil {
		log.Fatal("KeyValueStore is nil in NewServer")
	}

	aih := ai.NewAIHelper(os.Getenv("OPENAI_API_KEY"))
	bsvc := svc.NewBeliefService(kvStore, aih)
	dsvc := svc.NewDialecticService(kvStore, bsvc, aih)

	return &Server{
		bsvc:         bsvc,
		dsvc:         dsvc,
		kvStore:      kvStore,
		selfModelSvc: svc.NewSelfModelService(kvStore, dsvc, bsvc), // Pass bsvc here as well
	}
}

func RunServer(kvStore *db.KeyValueStore, port string) (*http.Server, *sync.WaitGroup, string) {
	svcServer := NewServer(kvStore)

	mux := http.NewServeMux()
	path, handler := pbconnect.NewEpistemicMeServiceHandler(svcServer)
	mux.Handle(path, handler)

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8081", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "Connect-Protocol-Version"},
		ExposedHeaders:   []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		Debug:            true,
	})

	var listener net.Listener
	var err error

	if port == "" {
		// For testing, use a dynamic port
		listener, err = net.Listen("tcp", ":0")
		if err != nil {
			log.Fatalf("Failed to listen: %v", err)
		}
		port = strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	} else {
		// For production, use the specified port
		listener, err = net.Listen("tcp", ":"+port)
		if err != nil {
			log.Fatalf("Failed to listen on port %s: %v", port, err)
		}
	}

	srv := &http.Server{
		Handler: h2c.NewHandler(corsHandler.Handler(mux), &http2.Server{}),
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Server is running on port %s for Connect", port)
		if err := srv.Serve(listener); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	return srv, &wg, port
}
