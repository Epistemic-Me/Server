package main

import (
	"context"
	"net"
	"net/http"
	"testing"
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

func createCustomHttpClient() *http.Client {
	customHttpClient := &http.Client{
		Timeout: 10 * time.Second, // Set a 10-second timeout for all requests
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

	clientWithHeaders := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.Header.Add("Origin", "http://localhost:8081") // emulating frontend CORS request
			return customHttpClient.Do(req)
		}),
		Timeout: customHttpClient.Timeout,
	}

	return clientWithHeaders
}

func TestIntegration(t *testing.T) {
	client := pbconnect.NewEpistemicMeServiceClient(
		createCustomHttpClient(),
		"http://localhost:8080",
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test CreateBelief
	createBeliefReq := &pb.CreateBeliefRequest{UserId: "test-user-id", BeliefContent: "I believe that the earth revolves around the sun"}
	createBeliefResp, err := client.CreateBelief(ctx, connect.NewRequest(createBeliefReq))
	if err != nil {
		t.Fatalf("CreateBelief failed: %v %v", err, createBeliefReq.String())
	}
	t.Logf("CreateBelief response: %+v\n", createBeliefResp.Msg)

	// Test ListBeliefs
	listBeliefsReq := &pb.ListBeliefsRequest{UserId: "test-user-id"}
	listBeliefsResp, err := client.ListBeliefs(ctx, connect.NewRequest(listBeliefsReq))
	if err != nil {
		t.Fatalf("ListBeliefs failed: %v", err)
	}
	t.Logf("ListBeliefs response: %+v\n", listBeliefsResp.Msg)

	// Test CreateDialectic
	createDialecticReq := &pb.CreateDialecticRequest{UserId: "test-user-id"}
	createDialecticResp, err := client.CreateDialectic(ctx, connect.NewRequest(createDialecticReq))
	if err != nil {
		t.Fatalf("CreateDialectic failed: %v", err)
	}
	t.Logf("CreateDialectic response: %+v\n", createDialecticResp.Msg)

	dialecticId := createDialecticResp.Msg.DialecticId

	// Test ListDialectics
	listDialecticsReq := &pb.ListDialecticsRequest{UserId: "test-user-id"}
	listDialecticsResp, err := client.ListDialectics(ctx, connect.NewRequest(listDialecticsReq))
	if err != nil {
		t.Fatalf("ListDialectics failed: %v", err)
	}
	t.Logf("ListDialectics response: %+v\n", listDialecticsResp.Msg)

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test UpdateDialectic
	updateDialecticReq := &pb.UpdateDialecticRequest{
		DialecticId: dialecticId,
		Answer: &models.UserAnswer{
			UserAnswer:         "answer",
			CreatedAtMillisUtc: 1000,
		},
		UserId: "test-user-id",
	}

	updateDialecticResp, err := client.UpdateDialectic(ctx, connect.NewRequest(updateDialecticReq))
	if err != nil {
		t.Fatalf("UpdateDialectic failed: %v", err)
	}
	t.Logf("UpdateDialectic response: %+v\n", updateDialecticResp.Msg)
}
