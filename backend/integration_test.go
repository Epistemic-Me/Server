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
	"github.com/stretchr/testify/assert"
)

// roundTripperFunc type defines a custom HTTP RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip implements the RoundTripper interface for roundTripperFunc.
func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// createCustomHttpClient creates an HTTP client with custom settings.
func createCustomHttpClient() *http.Client {
	return &http.Client{
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
}

// createServiceClient creates a new gRPC service client.
func createServiceClient(customHttpClient *http.Client) pbconnect.EpistemicMeServiceClient {
	clientWithHeaders := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.Header.Add("Origin", "http://localhost:8081")
			return customHttpClient.Do(req)
		}),
		Timeout: customHttpClient.Timeout,
	}
	return pbconnect.NewEpistemicMeServiceClient(clientWithHeaders, "http://localhost:8080")
}

// TestIntegration runs an integration test for the gRPC methods.
func TestIntegration(t *testing.T) {
	customHttpClient := createCustomHttpClient()
	client := createServiceClient(customHttpClient)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test CreateBelief
	createBeliefReq := &pb.CreateBeliefRequest{UserId: "test-user-id", BeliefContent: "I believe that the earth revolves around the sun"}
	createBeliefResp, err := client.CreateBelief(ctx, connect.NewRequest(createBeliefReq))
	if err != nil {
		t.Fatalf("CreateBelief failed: %v %v", err, createBeliefReq.String())
	}

	assert.NotNil(t, createBeliefResp.Msg)
	t.Logf("CreateBelief response: %+v\n", createBeliefResp.Msg)

	// Test ListBeliefs
	listBeliefsReq := &pb.ListBeliefsRequest{UserId: "test-user-id"}
	listBeliefsResp, err := client.ListBeliefs(ctx, connect.NewRequest(listBeliefsReq))
	if err != nil {
		t.Fatalf("ListBeliefs failed: %v", err)
	}

	assert.NotNil(t, listBeliefsResp.Msg)
	t.Logf("ListBeliefs response: %+v\n", listBeliefsResp.Msg)

	// Test CreateDialectic
	createDialecticReq := &pb.CreateDialecticRequest{UserId: "test-user-id"}
	createDialecticResp, err := client.CreateDialectic(ctx, connect.NewRequest(createDialecticReq))
	if err != nil {
		t.Fatalf("CreateDialectic failed: %v", err)
	}

	dialecticId := createDialecticResp.Msg.DialecticId
	assert.NotEmpty(t, dialecticId)
	t.Logf("CreateDialectic response: %+v\n", createDialecticResp.Msg)

	// Test ListDialectics
	listDialecticsReq := &pb.ListDialecticsRequest{UserId: "test-user-id"}
	listDialecticsResp, err := client.ListDialectics(ctx, connect.NewRequest(listDialecticsReq))
	if err != nil {
		t.Fatalf("ListDialectics failed: %v", err)
	}

	assert.NotNil(t, listDialecticsResp.Msg)
	t.Logf("ListDialectics response: %+v\n", listDialecticsResp.Msg)

	// Test UpdateDialectic
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	updateDialecticReq := &pb.UpdateDialecticRequest{
		DialecticId: dialecticId,
		Answer: &models.UserAnswer{
			UserAnswer:         "answer",
			CreatedAtMillisUtc: 1000,
		},
		UserId: "test-user-id",
		DryRun: true,
	}

	updateDialecticResp, err := client.UpdateDialectic(ctx, connect.NewRequest(updateDialecticReq))
	if err != nil {
		t.Fatalf("UpdateDialectic failed: %v", err)
	}

	// Test UpdateDialectic
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	assert.NotNil(t, updateDialecticResp.Msg)
	t.Logf("UpdateDialectic response for Draft: %+v\n", updateDialecticResp.Msg)

	updateDialecticReq = &pb.UpdateDialecticRequest{
		DialecticId: dialecticId,
		Answer: &models.UserAnswer{
			UserAnswer:         "answer",
			CreatedAtMillisUtc: 1000,
		},
		UserId: "test-user-id",
	}

	updateDialecticResp, err = client.UpdateDialectic(ctx, connect.NewRequest(updateDialecticReq))
	if err != nil {
		t.Fatalf("UpdateDialectic failed: %v", err)
	}

	assert.NotNil(t, updateDialecticResp.Msg)
	t.Logf("UpdateDialectic response: %+v\n", updateDialecticResp.Msg)
}
