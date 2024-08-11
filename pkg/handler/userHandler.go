package handler

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/joho/godotenv"

	_ "tg-users-database/docs"
	"tg-users-database/pkg/db"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

const (
	timeoutToContext = 60 * time.Second
)

// UserHandler contains the dependencies for the HTTPS handlers and the router.
type UserHandler struct {
	Database *db.Database
	Router   *gin.Engine
	botToken string
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse represents a success response.
type SuccessResponse struct {
	Message string `json:"message"`
}

// NewHandler creates a new UserHandler with an initialized router.
func NewHandler(database *db.Database) *UserHandler {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatal("BOT_TOKEN is not set")
	}

	handler := &UserHandler{
		Database: database,
		Router:   gin.Default(),
		botToken: botToken,
	}
	handler.setupRouter()
	return handler
}

func (h *UserHandler) BotAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/swagger") {
			c.Next()
			return
		}

		token := c.GetHeader("Authorization")
		if token != "Bearer "+h.botToken {
			logRequestDetails(c, "incorrect bot token")
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// logRequestDetails logs the details of the request.
func logRequestDetails(c *gin.Context, message string) {
	log.Printf("%s: Method=%s, URL=%s, Headers=%v, Params=%v",
		message,
		c.Request.Method,
		c.Request.URL.String(),
		c.Request.Header,
		c.Request.URL.Query(),
	)
}

// setupRouter registers the routes.
func (h *UserHandler) setupRouter() {
	h.Router.Use(gin.Logger())
	h.Router.Use(gin.Recovery())
	h.Router.Use(h.BotAuthMiddleware())

	// CORS configuration
	h.Router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://example.com"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	userRoutes := h.Router.Group("/users")
	{
		userRoutes.POST("/", h.createUser)
		userRoutes.GET("/:username", h.user)
		userRoutes.PUT("/:username", h.updateUserSubscription)
		userRoutes.DELETE("/:username", h.deleteUser)
		userRoutes.GET("/:username/subscription", h.subscriptionStatus)
		userRoutes.GET("/:username/exists", h.isUserExists)
		userRoutes.PUT("/:username/traffic", h.updateUserTraffic)
	}

	// Swagger endpoint without BotAuthMiddleware
	h.Router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

// checkUserExists checks if a user exists and handles errors.
func (h *UserHandler) checkUserExists(c *gin.Context, username string) (bool, error) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutToContext)
	defer cancel()

	exists, err := h.Database.IsUserExists(ctx, username)
	if err != nil {
		return false, err
	}
	if !exists {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "User not found"})
		return false, nil
	}
	return true, nil
}

// createUser handles the creation of a new db.User.
// @Summary Create a new User
// @Description Create a new User with the provided details
// @Tags users
// @Accept json
// @Produce json
// @Param User body db.User true "User details"
// @Success 201 {object} db.User
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /users [post]
func (h *UserHandler) createUser(c *gin.Context) {
	var newUser db.User
	if err := c.BindJSON(&newUser); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutToContext)
	defer cancel()

	if err := h.Database.CreateUser(ctx, &newUser); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, newUser)
}

// user handles retrieving a User by username.
// @Summary Get a User by username
// @Description Get User details by username
// @Tags users
// @Accept json
// @Produce json
// @Param username path string true "Username"
// @Success 200 {object} db.User
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /users/{username} [get]
func (h *UserHandler) user(c *gin.Context) {
	username := c.Param("username")

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutToContext)
	defer cancel()

	user, err := h.Database.User(ctx, username)
	if user == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// updateUserSubscription handles updating a User's subscription status.
// @Summary Update a User's subscription status
// @Description Update the subscription status of a User by username
// @Tags users
// @Accept json
// @Produce json
// @Param username path string true "Username"
// @Param User body db.User true "Updated User details"
// @Success 200 {object} db.User
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /users/{username} [put]
func (h *UserHandler) updateUserSubscription(c *gin.Context) {
	username := c.Param("username")
	var updateUser db.User
	if err := c.BindJSON(&updateUser); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	exists, err := h.checkUserExists(c, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	if !exists {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutToContext)
	defer cancel()

	err = h.Database.UpdateUserSubscription(ctx, username, updateUser.Subscription)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, updateUser)
}

// deleteUser handles deleting a User by username.
// @Summary Delete a User by username
// @Description Delete a User by their username
// @Tags users
// @Produce json
// @Param username path string true "Username"
// @Success 204 {object} nil
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /users/{username} [delete]
func (h *UserHandler) deleteUser(c *gin.Context) {
	username := c.Param("username")

	exists, err := h.checkUserExists(c, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	if !exists {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutToContext)
	defer cancel()

	if err := h.Database.DeleteUser(ctx, username); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// subscriptionStatus handles retrieving the subscription status of a User by username.
// @Summary Get subscription status of a User by username
// @Description Get the subscription status of a User by their username
// @Tags users
// @Produce json
// @Param username path string true "Username"
// @Success 200 {string} string "Subscription status"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /users/{username}/subscription [get]
func (h *UserHandler) subscriptionStatus(c *gin.Context) {
	username := c.Param("username")

	exists, err := h.checkUserExists(c, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	if !exists {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutToContext)
	defer cancel()

	status, err := h.Database.SubscriptionStatus(ctx, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// isUserExists handles checking if a User exists by username.
// @Summary Check if a User exists by username
// @Description Check if a User exists by their username
// @Tags users
// @Produce json
// @Param username path string true "Username"
// @Success 200 {bool} boolean "User exists or not"
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /users/{username}/exists [get]
func (h *UserHandler) isUserExists(c *gin.Context) {
	username := c.Param("username")

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutToContext)
	defer cancel()

	exist, err := h.Database.IsUserExists(ctx, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, exist)
}

// updateUserTraffic handles updating the amount of traffic used by a User
// @Summary Update the amount of traffic used by a User
// @Description Update the traffic used by a User identified by username
// @Tags users
// @Accept json
// @Produce json
// @Param username path string true "Username"
// @Param traffic body float64 true "Traffic used in MB"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /users/{username}/traffic [put]
func (h *UserHandler) updateUserTraffic(c *gin.Context) {
	username := c.Param("username")
	var traffic float64
	if err := c.BindJSON(&traffic); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	exists, err := h.checkUserExists(c, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	if !exists {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutToContext)
	defer cancel()

	err = h.Database.UpdateUserTraffic(ctx, username, traffic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "Traffic updated successfully"})
}
