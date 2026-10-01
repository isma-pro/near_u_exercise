package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ismaelucky94/near_u_exercise/internal/models"
	"github.com/ismaelucky94/near_u_exercise/internal/services"
)

// UserHandler handles HTTP requests for users.
type UserHandler struct {
	service *services.UserService
}

// NewUserHandler returns a new UserHandler.
func NewUserHandler() *UserHandler {
	return &UserHandler{service: services.NewUserService()}
}

// List returns all users.
func (h *UserHandler) List(c *gin.Context) {
	c.JSON(http.StatusOK, h.service.List())
}

// Get returns a single user by ID.
func (h *UserHandler) Get(c *gin.Context) {
	user, err := h.service.Get(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}

// Create creates a new user.
func (h *UserHandler) Create(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.service.Create(&user)
	c.JSON(http.StatusCreated, user)
}

// Update updates an existing user.
func (h *UserHandler) Update(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated, err := h.service.Update(c.Param("id"), &user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}

// Delete removes a user.
func (h *UserHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
