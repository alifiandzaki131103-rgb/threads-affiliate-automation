package model

import (
	"time"

	"github.com/google/uuid"
)

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// LoginRequest represents a user login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// UserResponse is a safe user representation for API responses
type UserResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Plan  string    `json:"plan"`
}

// LoginResponse represents the response after successful authentication
type LoginResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         UserResponse `json:"user"`
}

// AddLinkRequest represents a request to add a new affiliate link
type AddLinkRequest struct {
	URL         string   `json:"url" validate:"required,url"`
	ProductName *string  `json:"product_name,omitempty"`
	Category    *string  `json:"category,omitempty"`
	Price       *float64 `json:"price,omitempty"`
}

// BulkAddLinksRequest represents a request to add multiple affiliate links at once
type BulkAddLinksRequest struct {
	URLs []string `json:"urls" validate:"required,min=1,dive,url"`
}

// CreatePostRequest represents a request to create a new post
type CreatePostRequest struct {
	LinkID  uuid.UUID `json:"link_id" validate:"required"`
	Persona string    `json:"persona" validate:"required"`
	Format  string    `json:"format" validate:"required,oneof=single thread hot_take question story"`
}

// LinkWithProduct combines affiliate link data with product info
type LinkWithProduct struct {
	ID          uuid.UUID `json:"id"`
	ProductID   uuid.UUID `json:"product_id"`
	OriginalURL string    `json:"original_url"`
	ShortSlug   string    `json:"short_slug"`
	Platform    string    `json:"platform"`
	Status      string    `json:"status"`
	ClickCount  int       `json:"click_count"`
	ProductName string    `json:"product_name"`
	Price       float64   `json:"price"`
	Category    string    `json:"category"`
	CreatedAt   time.Time `json:"created_at"`
}
