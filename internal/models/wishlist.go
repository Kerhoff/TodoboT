package models

import "time"

// WishList represents a personal wish list for a family member
type WishList struct {
	ID        int64      `json:"id" db:"id"`
	FamilyID  int64      `json:"family_id" db:"family_id"`
	UserID    int64      `json:"user_id" db:"user_id"`
	Name      string     `json:"name" db:"name"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	Items     []WishItem `json:"items,omitempty"`
	User      *User      `json:"user,omitempty"`
}

// WishItem represents an item in a wish list
type WishItem struct {
	ID         int64     `json:"id" db:"id"`
	WishListID int64     `json:"wish_list_id" db:"wish_list_id"`
	Name       string    `json:"name" db:"name"`
	URL        string    `json:"url" db:"url"`
	Price      string    `json:"price" db:"price"`
	Notes      string    `json:"notes" db:"notes"`
	Reserved   bool      `json:"reserved" db:"reserved"`
	ReservedByID *int64  `json:"reserved_by_id" db:"reserved_by_id"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	ReservedBy *User     `json:"reserved_by,omitempty"`
}
