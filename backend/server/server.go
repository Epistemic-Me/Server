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
	kvStore      *db.KeyValueStore
	selfModelSvc *svc.SelfModelService
	developerSvc *svc.DeveloperService
	userSvc      *svc.UserService
}

func (s *Server) CreateBelief(
	ctx context.Context,
	req *connect.Request[pb.CreateBeliefRequest],
) (*connect.Response[pb.CreateBeliefResponse], error) {
	log.Printf("CreateBelief called with request: %+v", req.Msg)

	input := &svcmodels.CreateBeliefInput{
		SelfModelID:   req.Msg.SelfModelId,
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
	log.Println("UpdateDialectic called with request:", req.Msg)

	response, err := s.dsvc.UpdateDialectic(&svcmodels.UpdateDialecticInput{
		SelfModelID: req.Msg.SelfModelId,
		ID:          req.Msg.Id,
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

func (s *Server) GetBeliefSystem(
	ctx context.Context,
	req *connect.Request[pb.GetBeliefSystemRequest],
) (*connect.Response[pb.GetBeliefSystemResponse], error) {
	log.Printf("GetBeliefSystem called with request: %+v", req.Msg)

	beliefSystem, err := s.bsvc.GetBeliefSystem(req.Msg.SelfModelId)
	if err != nil {
		log.Printf("GetBeliefSystem ERROR: %v", err)
		return nil, err
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

func (s *Server) CreateDeveloper(ctx context.Context, req *connect.Request[pb.CreateDeveloperRequest]) (*connect.Response[pb.CreateDeveloperResponse], error) {
	log.Printf("CreateDeveloper called with request: %+v", req.Msg)

	input := &svcmodels.CreateDeveloperInput{
		Name:  req.Msg.Name,
		Email: req.Msg.Email,
	}

	response, err := s.developerSvc.CreateDeveloper(input)
	if err != nil {
		log.Printf("CreateDeveloper ERROR: %v", err)
		return nil, err
	}

	protoResponse := &pb.CreateDeveloperResponse{
		Developer: response.Developer.ToProto(),
	}

	return connect.NewResponse(protoResponse), nil
}

func (s *Server) CreateUser(ctx context.Context, req *connect.Request[pb.CreateUserRequest]) (*connect.Response[pb.CreateUserResponse], error) {
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
