package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"tg-users-database/pkg/db"
	"tg-users-database/pkg/user"
)

const (
	timeoutToContext = 10 * time.Second
)

// UserHandler contains the dependencies for the HTTP handlers and the router.
type UserHandler struct {
	Database *db.Database
	Router   *gin.Engine
}

// NewHandler creates a new UserHandler with an initialized router.
func NewHandler(database *db.Database) *UserHandler {
	handler := &UserHandler{
		Database: database,
		Router:   gin.Default(),
	}
	handler.setupRouter()
	return handler
}

// setupRouter registers the routes.
func (h *UserHandler) setupRouter() {
	h.Router.Use(gin.Logger())
	h.Router.Use(gin.Recovery())

	userRoutes := h.Router.Group("/users")
	{
		userRoutes.POST("/", h.createUser)
		userRoutes.GET("/:username", h.getUser)
		userRoutes.PUT("/:username", h.updateUser)
		userRoutes.DELETE("/:username", h.deleteUser)
		userRoutes.GET("/:username/subscription", h.getSubscriptionStatus)
	}

	// Swagger endpoint
	h.Router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

// createUser handles the creation of a new user.
// @Summary Create a new user
// @Description Create a new user with the provided details
// @Accept json
// @Produce json
// @Param user body user.User true "User details"
// @Success 201 {object} user.User
// @Failure 400 string error message
// @Failure 500 string error message
// @Router /users [post]
func (h *UserHandler) createUser(c *gin.Context) {
	var newUser user.User
	if err := c.BindJSON(&newUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutToContext)
	defer cancel()

	if err := h.Database.CreateUser(ctx, &newUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, newUser)
}

// getUser handles retrieving a user by username.
// @Summary Get a user by username
// @Description Get user details by username
// @Produce json
// @Param username path string true "Username"
// @Success 200 {object} user.User
// @Failure 400 string error message
// @Failure 404 string error message
// @Failure 500 string error message
// @Router /users/{username} [get]
func (h *UserHandler) getUser(c *gin.Context) {
	username := c.Param("username")

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutToContext)
	defer cancel()

	user, err := h.Database.GetUser(ctx, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// updateUser handles updating a user's subscription status.
// @Summary Update a user's subscription status
// @Description Update the subscription status of a user by username
// @Accept json
// @Produce json
// @Param username path string true "Username"
// @Param user body user.User true "Updated user details"
// @Success 200 {object} user.User
// @Failure 400 string error message
// @Failure 500 string error message
// @Router /users/{username} [put]
func (h *UserHandler) updateUser(c *gin.Context) {
	username := c.Param("username")
	var updateUser user.User
	if err := c.BindJSON(&updateUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutToContext)
	defer cancel()

	if err := h.Database.UpdateUser(ctx, username, updateUser.SubscriptionStatus); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updateUser)
}

// deleteUser handles deleting a user by username.
// @Summary Delete a user by username
// @Description Delete a user by their username
// @Produce json
// @Param username path string true "Username"
// @Success 204 {object} nil
// @Failure 400 string error message
// @Failure 500 string error message
// @Router /users/{username} [delete]
func (h *UserHandler) deleteUser(c *gin.Context) {
	username := c.Param("username")

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutToContext)
	defer cancel()

	if err := h.Database.DeleteUser(ctx, username); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// getSubscriptionStatus handles retrieving the subscription status of a user by username.
// @Summary Get subscription status of a user by username
// @Description Get the subscription status of a user by their username
// @Produce json
// @Param username path string true "Username"
// @Success 200 {string} string "Subscription status"
// @Failure 400 string error message
// @Failure 404 string error message
// @Failure 500 string error message
// @Router /users/{username}/subscription [get]
func (h *UserHandler) getSubscriptionStatus(c *gin.Context) {
	username := c.Param("username")

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutToContext)
	defer cancel()

	status, err := h.Database.SubscriptionStatus(ctx, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}
