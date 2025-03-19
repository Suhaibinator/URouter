package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Suhaibinator/SRouter/pkg/codec"
	"github.com/Suhaibinator/SRouter/pkg/router"
	"go.uber.org/zap"
)

// User represents a user in our system
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// GetUserRequest is the request body for getting a user
type GetUserRequest struct {
	ID string `json:"id"`
}

// GetUserResponse is the response body for getting a user
type GetUserResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// In-memory user store
var users = map[string]User{
	"1": {ID: "1", Name: "John Doe", Email: "john@example.com"},
	"2": {ID: "2", Name: "Jane Smith", Email: "jane@example.com"},
	"3": {ID: "3", Name: "Bob Johnson", Email: "bob@example.com"},
}

// GetUserHandler handles getting a user
func GetUserHandler(r *http.Request, req GetUserRequest) (GetUserResponse, error) {
	// Get the user ID from the request
	id := req.ID
	if id == "" {
		// If no ID in the request, try to get it from the path parameter
		id = router.GetParam(r, "id")
	}

	if id == "" {
		return GetUserResponse{}, router.NewHTTPError(http.StatusBadRequest, "User ID is required")
	}

	// Get the user
	user, ok := users[id]
	if !ok {
		return GetUserResponse{}, router.NewHTTPError(http.StatusNotFound, "User not found")
	}

	// Return the response
	return GetUserResponse(user), nil
}

func main() {
	// Create a logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer func() {
		if syncErr := logger.Sync(); syncErr != nil {
			log.Printf("Failed to sync logger: %v", syncErr)
		}
	}()

	// Create a router configuration
	routerConfig := router.RouterConfig{
		Logger:            logger,
		GlobalTimeout:     2 * time.Second,
		GlobalMaxBodySize: 1 << 20, // 1 MB
	}

	// Define the auth function that takes a context and token and returns a string and a boolean
	authFunction := func(ctx context.Context, token string) (string, bool) {
		// This is a simple example, so we'll just validate that the token is not empty
		if token != "" {
			return token, true
		}
		return "", false
	}

	// Define the function to get the user ID from a string
	userIdFromUserFunction := func(user string) string {
		// In this example, we're using the string itself as the ID
		return user
	}

	// Create a router with string as both the user ID and user type
	r := router.NewRouter[string, string](routerConfig, authFunction, userIdFromUserFunction)

	// Register routes demonstrating different source types

	// 1. Standard body-based route (default)
	router.RegisterGenericRoute[GetUserRequest, GetUserResponse, string](r, router.RouteConfig[GetUserRequest, GetUserResponse]{
		Path:    "/users/body/:id",
		Methods: []string{"GET"},
		Codec:   codec.NewJSONCodec[GetUserRequest, GetUserResponse](),
		Handler: GetUserHandler,
		// SourceType defaults to Body
	})

	// 2. Base64 query parameter route
	router.RegisterGenericRoute[GetUserRequest, GetUserResponse, string](r, router.RouteConfig[GetUserRequest, GetUserResponse]{
		Path:       "/users/query/:id",
		Methods:    []string{"GET"},
		Codec:      codec.NewJSONCodec[GetUserRequest, GetUserResponse](),
		Handler:    GetUserHandler,
		SourceType: router.Base64QueryParameter,
		SourceKey:  "data", // Will look for ?data=base64encodedstring
	})

	// 3. Base64 path parameter route
	router.RegisterGenericRoute[GetUserRequest, GetUserResponse, string](r, router.RouteConfig[GetUserRequest, GetUserResponse]{
		Path:       "/users/path/:data",
		Methods:    []string{"GET"},
		Codec:      codec.NewJSONCodec[GetUserRequest, GetUserResponse](),
		Handler:    GetUserHandler,
		SourceType: router.Base64PathParameter,
		SourceKey:  "data", // Will use the :data path parameter
	})

	// Start the server
	fmt.Println("Source Types Example Server listening on :8080")
	fmt.Println("Available endpoints:")
	fmt.Println("  - GET /users/body/:id (standard body-based route)")
	fmt.Println("  - GET /users/query/:id?data=base64encodedstring (base64 query parameter route)")
	fmt.Println("  - GET /users/path/:data (base64 path parameter route)")
	fmt.Println("\nExample curl commands:")

	// Create a sample request
	sampleReq := GetUserRequest{ID: "1"}
	jsonBytes, _ := json.Marshal(sampleReq)
	base64Str := base64.StdEncoding.EncodeToString(jsonBytes)

	fmt.Println("  curl -X GET http://localhost:8080/users/body/1")
	fmt.Printf("  curl -X GET \"http://localhost:8080/users/query/1?data=%s\"\n", base64Str)
	fmt.Printf("  curl -X GET http://localhost:8080/users/path/%s\n", base64Str)

	log.Fatal(http.ListenAndServe(":8080", r))
}
