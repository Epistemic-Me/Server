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
	"github.com/google/uuid"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc/metadata" // Changed from internal/metadata

	ai "epistemic-me-core/ai"
	db "epistemic-me-core/db"
	pb "epistemic-me-core/pb"
	models "epistemic-me-core/pb/models"
	"epistemic-me-core/pb/pbconnect"
	svc "epistemic-me-core/svc"
	svcmodels "epistemic-me-core/svc/models"
)

type Server struct {
	bsvc         *svc.BeliefService
	dsvc         *svc.DialecticService
	kvStore      *db.KeyValueStore
	selfModelSvc *svc.SelfModelService
	developerSvc *svc.DeveloperService
	userSvc      *svc.UserService
}

// Move validateAPIKey to be a regular function instead of a method
func validateAPIKey[T any](
	ctx context.Context,
	req *connect.Request[T],
) (context.Context, error) {
	var apiKey string

	// First try metadata
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if apiKeys := md.Get("x-api-key"); len(apiKeys) > 0 {
			apiKey = apiKeys[0]
		}
	}

	// Then try headers
	if apiKey == "" {
		apiKey = req.Header().Get("x-api-key")
	}

	if apiKey == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing API key"))
	}

	// Verify the API key format (should be a UUID)
	if _, err := uuid.Parse(apiKey); err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid API key format"))
	}

	// TODO: Add actual API key validation against the database
	// For now, we'll consider any well-formed UUID as valid
	// This should be replaced with actual validation logic

	return ctx, nil
}

// Update the CreateBelief method to handle evidence
func (s *Server) CreateBelief(
	ctx context.Context,
	req *connect.Request[pb.CreateBeliefRequest],
) (*connect.Response[pb.CreateBeliefResponse], error) {
	ctx, err := validateAPIKey(ctx, req)
	if err != nil {
		return nil, err
	}

	log.Printf("CreateBelief called with request: %+v", req.Msg)

	input := &svcmodels.CreateBeliefInput{
		SelfModelID:   req.Msg.SelfModelId,
		BeliefContent: req.Msg.BeliefContent,
	}

	// Handle evidence if provided
	switch evidence := req.Msg.Evidence.(type) {
	case *pb.CreateBeliefRequest_HypothesisEvidence:
		input.BeliefEvidence = &svcmodels.BeliefEvidence{
			Type:      svcmodels.EvidenceTypeHypothesis,
			Content:   evidence.HypothesisEvidence.Evidence,
			IsCounter: evidence.HypothesisEvidence.IsCounterfactual,
		}
	case *pb.CreateBeliefRequest_ActionOutcome:
		input.BeliefEvidence = &svcmodels.BeliefEvidence{
			Type:    svcmodels.EvidenceTypeAction,
			Action:  evidence.ActionOutcome.Action,
			Outcome: evidence.ActionOutcome.Outcome,
		}
	}

	response, err := s.bsvc.CreateBelief(input)
	if err != nil {
		log.Printf("CreateBelief ERROR: %v", err)
		return nil, err
	}

	return connect.NewResponse(&pb.CreateBeliefResponse{
		Belief:       response.Belief.ToProto(),
		BeliefSystem: response.BeliefSystem.ToProto(),
	}), nil
}

func (s *Server) ListBeliefs(
	ctx context.Context,
	req *connect.Request[pb.ListBeliefsRequest],
) (*connect.Response[pb.ListBeliefsResponse], error) {
	ctx, err := validateAPIKey(ctx, req)
	if err != nil {
		return nil, err
	}

	log.Println("ListBeliefs called with request:", req.Msg)

	response, err := s.bsvc.ListBeliefs(&svcmodels.ListBeliefsInput{
		SelfModelID: req.Msg.SelfModelId,
		BeliefIDs:   req.Msg.BeliefIds,
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
	ctx, err := validateAPIKey(ctx, req)
	if err != nil {
		return nil, err
	}

	log.Printf("CreateDialectic called with request: %+v", req.Msg)

	input := &svcmodels.CreateDialecticInput{
		SelfModelID:   req.Msg.SelfModelId,
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
		Dialectic: response.Dialectic.ToProto(),
	}
	log.Printf("CreateDialectic proto response: %+v", protoResponse)

	return connect.NewResponse(protoResponse), nil
}

func (s *Server) ListDialectics(
	ctx context.Context,
	req *connect.Request[pb.ListDialecticsRequest],
) (*connect.Response[pb.ListDialecticsResponse], error) {
	ctx, err := validateAPIKey(ctx, req)
	if err != nil {
		return nil, err
	}

	log.Println("ListDialectics called with request:", req.Msg)

	response, err := s.dsvc.ListDialectics(&svcmodels.ListDialecticsInput{
		SelfModelID: req.Msg.SelfModelId,
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
	ctx, err := validateAPIKey(ctx, req)
	if err != nil {
		return nil, err
	}

	log.Println("UpdateDialectic called with request:", req.Msg)

	input := &svcmodels.UpdateDialecticInput{
		ID:           req.Msg.Id,
		SelfModelID:  req.Msg.SelfModelId,
		DryRun:       req.Msg.DryRun,
		QuestionBlob: req.Msg.QuestionBlob,
		AnswerBlob:   req.Msg.AnswerBlob,
	}

	// Set Answer if provided
	if req.Msg.Answer != nil {
		input.Answer = svcmodels.UserAnswer{
			UserAnswer:         req.Msg.Answer.UserAnswer,
			CreatedAtMillisUTC: req.Msg.Answer.CreatedAtMillisUtc,
		}
	}

	// Set CustomQuestion if provided
	if req.Msg.CustomQuestion != "" {
		customQ := req.Msg.CustomQuestion
		input.CustomQuestion = &customQ
	}

	response, err := s.dsvc.UpdateDialectic(input)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if response == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unexpected nil response"))
	}

	return connect.NewResponse(&pb.UpdateDialecticResponse{
		Dialectic: response.Dialectic.ToProto(),
	}), nil
}

// Update GetBeliefSystem to support conceptualization
func (s *Server) GetBeliefSystem(
	ctx context.Context,
	req *connect.Request[pb.GetBeliefSystemRequest],
) (*connect.Response[pb.GetBeliefSystemResponse], error) {
	ctx, err := validateAPIKey(ctx, req)
	if err != nil {
		return nil, err
	}

	log.Printf("GetBeliefSystem called with request: %+v", req.Msg)

	beliefSystem, err := s.bsvc.GetBeliefSystem(req.Msg.SelfModelId)
	if err != nil {
		log.Printf("GetBeliefSystem ERROR: %v", err)
		return nil, err
	}

	// Generate conceptualization if requested
	if req.Msg.Conceptualize {
		err = s.bsvc.ConceptualizeBeliefSystem(beliefSystem)
		if err != nil {
			log.Printf("ConceptualizeBeliefSystem ERROR: %v", err)
			return nil, err
		}
	}

	// Include metrics if requested
	if req.Msg.IncludeMetrics {
		err = s.bsvc.ComputeMetrics(beliefSystem)
		if err != nil {
			log.Printf("ComputeMetrics ERROR: %v", err)
			return nil, err
		}
	}

	return connect.NewResponse(&pb.GetBeliefSystemResponse{
		BeliefSystem: beliefSystem.ToProto(),
	}), nil
}

// Add this method to your server type
func (s *Server) UpdateKeyValueStore(ctx context.Context, req *connect.Request[pb.UpdateKeyValueStoreRequest]) (*connect.Response[pb.UpdateKeyValueStoreResponse], error) {
	// Implement the logic for updating the key-value store
	// For now, return a placeholder response
	return connect.NewResponse(&pb.UpdateKeyValueStoreResponse{}), nil
}

func (s *Server) CreateSelfModel(ctx context.Context, req *connect.Request[pb.CreateSelfModelRequest]) (*connect.Response[pb.CreateSelfModelResponse], error) {
	ctx, err := validateAPIKey(ctx, req)
	if err != nil {
		return nil, err
	}

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
	ctx, err := validateAPIKey(ctx, req)
	if err != nil {
		return nil, err
	}

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
	ctx, err := validateAPIKey(ctx, req)
	if err != nil {
		return nil, err
	}

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
	ctx, err := validateAPIKey(ctx, req)
	if err != nil {
		return nil, err
	}

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

func (s *Server) CreateDeveloper(ctx context.Context, req *connect.Request[pb.CreateDeveloperRequest]) (*connect.Response[pb.CreateDeveloperResponse], error) {
	// This method doesn't require API key validation
	input := &svcmodels.CreateDeveloperInput{
		Name:  req.Msg.Name,
		Email: req.Msg.Email,
	}

	response, err := s.developerSvc.CreateDeveloper(input)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// The response should already include the API key from the service
	protoResponse := &pb.CreateDeveloperResponse{
		Developer: response.Developer.ToProto(),
	}

	return connect.NewResponse(protoResponse), nil
}

func (s *Server) CreateUser(ctx context.Context, req *connect.Request[pb.CreateUserRequest]) (*connect.Response[pb.CreateUserResponse], error) {
	ctx, err := validateAPIKey(ctx, req)
	if err != nil {
		return nil, err
	}

	log.Printf("CreateUser called with request: %+v", req.Msg)

	input := &svcmodels.CreateUserInput{
		DeveloperID: req.Msg.DeveloperId,
		Name:        req.Msg.Name,
		Email:       req.Msg.Email,
	}

	response, err := s.userSvc.CreateUser(input)
	if err != nil {
		log.Printf("CreateUser ERROR: %v", err)
		return nil, err
	}

	protoResponse := &pb.CreateUserResponse{
		User: response.User.ToProto(),
	}

	return connect.NewResponse(protoResponse), nil
}

func (s *Server) GetDeveloper(ctx context.Context, req *connect.Request[pb.GetDeveloperRequest]) (*connect.Response[pb.GetDeveloperResponse], error) {
	ctx, err := validateAPIKey(ctx, req)
	if err != nil {
		return nil, err
	}

	log.Printf("GetDeveloper called with request: %+v", req.Msg)

	input := &svcmodels.GetDeveloperInput{
		ID: req.Msg.Id,
	}

	response, err := s.developerSvc.GetDeveloper(input)
	if err != nil {
		log.Printf("GetDeveloper ERROR: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoResponse := &pb.GetDeveloperResponse{
		Developer: response.ToProto(),
	}

	return connect.NewResponse(protoResponse), nil
}

func (s *Server) PreprocessQA(ctx context.Context, req *connect.Request[pb.PreprocessQARequest]) (*connect.Response[pb.PreprocessQAResponse], error) {
	ctx, err := validateAPIKey(ctx, req)
	if err != nil {
		return nil, err
	}

	// Use dialectic service method
	result, err := s.dsvc.PreprocessQuestionAnswers(&svcmodels.PreprocessQAInput{
		QuestionBlobs: req.Msg.QuestionBlobs,
		AnswerBlobs:   req.Msg.AnswerBlobs,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Convert internal model to protobuf response
	protoPairs := make([]*pb.QuestionAnswerPair, len(result.QAPairs))
	for i, pair := range result.QAPairs {
		protoPairs[i] = &pb.QuestionAnswerPair{
			Question: pair.Question,
			Answer:   pair.Answer,
		}
	}

	return connect.NewResponse(&pb.PreprocessQAResponse{
		QaPairs: protoPairs,
	}), nil
}

func NewServer(kvStore *db.KeyValueStore) *Server {
	if kvStore == nil {
		log.Fatal("KeyValueStore is nil in NewServer")
	}

	aih := ai.NewAIHelper(os.Getenv("OPENAI_API_KEY"))
	bsvc := svc.NewBeliefService(kvStore, aih)
	de := svc.NewDialecticEpistemology(bsvc, aih)
	dsvc := svc.NewDialecticService(kvStore, aih, de)

	return &Server{
		bsvc:         bsvc,
		dsvc:         dsvc,
		kvStore:      kvStore,
		selfModelSvc: svc.NewSelfModelService(kvStore, dsvc, bsvc),
		developerSvc: svc.NewDeveloperService(kvStore, aih),
		userSvc:      svc.NewUserService(kvStore, aih),
	}
}

func RunServer(kvStore *db.KeyValueStore, port string) (*http.Server, *sync.WaitGroup, string) {
	svcServer := NewServer(kvStore)

	mux := http.NewServeMux()
	path, handler := pbconnect.NewEpistemicMeServiceHandler(svcServer)
	mux.Handle(path, handler)

	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:8081", "http://localhost:3001", "http://localhost:3000", "http://localhost"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{
			"Accept",
			"Content-Type",
			"Content-Length",
			"Accept-Encoding",
			"X-CSRF-Token",
			"Authorization",
			"Connect-Protocol-Version",
			"Connect-Timeout-Ms",
			"x-api-key",
			"Origin",
		},
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
