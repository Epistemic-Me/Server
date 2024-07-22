package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	pb "epistemic-me-backend/pb"
	models "epistemic-me-backend/pb/models"
	"epistemic-me-backend/pb/pbconnect"

	"connectrpc.com/connect"
)

// Define roundTripperFunc type that takes an http.Request and returns an http.Response and error.
// This type will implement the http.RoundTripper interface.
type roundTripperFunc func(*http.Request) (*http.Response, error)

// Implement the RoundTrip method for roundTripperFunc type.
// This method is required to satisfy the http.RoundTripper interface.
// It simply calls the function itself with the provided http.Request.
func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func main() {
	customHttpClient := &http.Client{
		Timeout: 10 * time.Second, // Set a 10-second timeout for all requests
		// Add a transport that allows us to modify request headers
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	// Wrap the customHttpClient's Do function to add headers
	clientWithHeaders := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.Header.Add("Origin", "http://localhost:8081") // Replace example.com with the appropriate value
			return customHttpClient.Do(req)
		}),
		Timeout: customHttpClient.Timeout,
	}

	// Use this clientWithHeaders when creating your service client
	client := pbconnect.NewEpistemicMeServiceClient(
		clientWithHeaders,
		"http://localhost:8080",
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test CreateBelief
	createBeliefReq := &pb.CreateBeliefRequest{UserId: "test-user-id", BeliefContent: "I believe that the earth revolves around the sun"}
	createBeliefResp, err := client.CreateBelief(ctx, connect.NewRequest(createBeliefReq))
	if err != nil {
		log.Fatalf("CreateBelief failed: %v", err, createBeliefReq.String())
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
