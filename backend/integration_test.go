package main

import (
	"context"
	"log"
	"net/http"
	"time"

	pb "epistemic-me-backend/pb"
	models "epistemic-me-backend/pb/models"
	"epistemic-me-backend/pb/pbconnect"

	"connectrpc.com/connect"
)

func main() {
	client := pbconnect.NewEpistemicMeServiceClient(
		http.DefaultClient,
		"http://localhost:8080",
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test CreateBelief
	createBeliefReq := &pb.CreateBeliefRequest{UserId: "test-user-id"}
	createBeliefResp, err := client.CreateBelief(ctx, connect.NewRequest(createBeliefReq))
	if err != nil {
		log.Fatalf("CreateBelief failed: %v", err)
	}
	log.Printf("CreateBelief response: %+v\n", createBeliefResp.Msg)

	// Test ListBeliefs
	listBeliefsReq := &pb.ListBeliefsRequest{UserId: "test-user-id"}
	listBeliefsResp, err := client.ListBeliefs(ctx, connect.NewRequest(listBeliefsReq))
	if err != nil {
		log.Fatalf("ListBeliefs failed: %v", err)
	}
	log.Printf("ListBeliefs response: %+v\n", listBeliefsResp.Msg)

	// Test CreateDialectic
	createDialecticReq := &pb.CreateDialecticRequest{UserId: "test-user-id"}
	createDialecticResp, err := client.CreateDialectic(ctx, connect.NewRequest(createDialecticReq))
	if err != nil {
		log.Fatalf("CreateDialectic failed: %v", err)
	}
	log.Printf("CreateDialectic response: %+v\n", createDialecticResp.Msg)

	// Test ListDialectics
	listDialecticsReq := &pb.ListDialecticsRequest{UserId: "test-user-id"}
	listDialecticsResp, err := client.ListDialectics(ctx, connect.NewRequest(listDialecticsReq))
	if err != nil {
		log.Fatalf("ListDialectics failed: %v", err)
	}
	log.Printf("ListDialectics response: %+v\n", listDialecticsResp.Msg)

	// Test UpdateDialectic
	updateDialecticReq := &pb.UpdateDialecticRequest{
		DialecticId: "mock-dialectic-id",
		Answer: &models.UserAnswer{
			UserAnswer:         "answer",
			CreatedAtMillisUtc: 1000,
		},
	}
	updateDialecticResp, err := client.UpdateDialectic(ctx, connect.NewRequest(updateDialecticReq))
	if err != nil {
		log.Fatalf("UpdateDialectic failed: %v", err)
	}
	log.Printf("UpdateDialectic response: %+v\n", updateDialecticResp.Msg)
}
