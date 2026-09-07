package domain

import (
	"time"

	"github.com/google/uuid"
)

type Announcement struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	DisplayMode string    `json:"display_mode"`
	Dismissible bool      `json:"dismissible"`
	Scrolling   bool      `json:"scrolling"`
	IsPublished bool      `json:"is_published"`
	Revision    uuid.UUID `json:"revision"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
