package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Suhaibinator/SRouter/pkg/middleware"
	"github.com/Suhaibinator/SRouter/pkg/router"
	"go.uber.org/zap"
)

// User represents a user in the system
type User struct {
	ID    string
	Name  string
	Email string
	Roles []string
}

// Protected resource that requires authentication and uses the user object
func protectedUserHandler(w http.ResponseWriter, r *http.Request) {
	// Get the user from the context
	user := middleware.GetUser[User](r)
	if user == nil {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"message":"Hello, %s! This is a protected resource", "user_id":"%s", "email":"%s", "roles":["%s"]}`,
		user.Name, user.ID, user.Email, strings.Join(user.Roles, `","`))
}

// Protected resource that requires authentication but doesn't use the user object
func protectedHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message":"This is a protected resource"}`))
}

// Public resource that doesn't require authentication
func publicHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message":"This is a public resource"}`))
}

func main() {
	// Create a logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Mock user database
	users := map[string]User{
		"user1": {
			ID:    "1",
			Name:  "User One",
			Email: "user1@example.com",
			Roles: []string{"user"},
		},
		"user2": {
			ID:    "2",
			Name:  "User Two",
			Email: "user2@example.com",
			Roles: []string{"admin", "user"},
		},
	}

	// Mock token to user mapping
	tokens := map[string]string{
		"token1": "user1",
		"token2": "user2",
	}

	// Create a custom authentication function that returns a user
	customUserAuth := func(r *http.Request) (*User, error) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			return nil, errors.New("no authorization header")
		}

		// Extract the token
		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Look up the username for this token
		username, exists := tokens[token]
		if !exists {
			return nil, errors.New("invalid token")
		}

		// Look up the user
		user, exists := users[username]
		if !exists {
			return nil, errors.New("user not found")
		}

		// Return a pointer to the user
		return &user, nil
	}

	// Create a bearer token authentication function that returns a user
	bearerTokenUserAuth := func(token string) (*User, error) {
		// Look up the username for this token
		username, exists := tokens[token]
		if !exists {
			return nil, errors.New("invalid token")
		}

		// Look up the user
		user, exists := users[username]
		if !exists {
			return nil, errors.New("user not found")
		}

		// Return a pointer to the user
		return &user, nil
	}

	// Create a router configuration
	routerConfig := router.RouterConfig{
		Logger:            logger,
		GlobalTimeout:     2 * time.Second,
		GlobalMaxBodySize: 1 << 20, // 1 MB
		SubRouters: []router.SubRouterConfig{
			{
				PathPrefix: "/public",
				Routes: []router.RouteConfigBase{
					{
						Path:      "/resource",
						Methods:   []string{"GET"},
						AuthLevel: router.NoAuth,
						Handler:   publicHandler,
					},
				},
			},
			{
				PathPrefix: "/boolean-auth",
				Routes: []router.RouteConfigBase{
					{
						Path:      "/resource",
						Methods:   []string{"GET"},
						AuthLevel: router.AuthRequired,
						Middlewares: []router.Middleware{
							middleware.AuthenticationBool(func(r *http.Request) bool {
								// Simple boolean authentication
								authHeader := r.Header.Get("Authorization")
								if authHeader == "" {
									return false
								}
								token := strings.TrimPrefix(authHeader, "Bearer ")
								_, exists := tokens[token]
								return exists
							}),
						},
						Handler: protectedHandler,
					},
				},
			},
			{
				PathPrefix: "/user-auth",
				Routes: []router.RouteConfigBase{
					{
						Path:      "/custom",
						Methods:   []string{"GET"},
						AuthLevel: router.AuthRequired,
						Middlewares: []router.Middleware{
							middleware.AuthenticationWithUser(customUserAuth),
						},
						Handler: protectedUserHandler,
					},
					{
						Path:      "/bearer",
						Methods:   []string{"GET"},
						AuthLevel: router.AuthRequired,
						Middlewares: []router.Middleware{
							middleware.NewBearerTokenWithUserMiddleware(bearerTokenUserAuth, logger),
						},
						Handler: protectedUserHandler,
					},
				},
			},
		},
	}

	// Create a router
	r := router.NewRouter(routerConfig)

	// Start the server
	fmt.Println("User Authentication Example Server listening on :8080")
	fmt.Println("Available endpoints:")
	fmt.Println("  - GET /public/resource (no auth required)")
	fmt.Println("  - GET /boolean-auth/resource (boolean auth required)")
	fmt.Println("  - GET /user-auth/custom (custom user auth)")
	fmt.Println("  - GET /user-auth/bearer (bearer token user auth)")
	fmt.Println("  - GET /user-auth/basic (basic user auth)")
	fmt.Println("\nExample curl commands:")
	fmt.Println("  curl http://localhost:8080/public/resource")
	fmt.Println("  curl -H \"Authorization: Bearer token1\" http://localhost:8080/boolean-auth/resource")
	fmt.Println("  curl -H \"Authorization: Bearer token1\" http://localhost:8080/user-auth/custom")
	fmt.Println("  curl -H \"Authorization: Bearer token2\" http://localhost:8080/user-auth/bearer")
	fmt.Println("  curl -u user1:password http://localhost:8080/user-auth/basic")
	log.Fatal(http.ListenAndServe(":8080", r))
}
