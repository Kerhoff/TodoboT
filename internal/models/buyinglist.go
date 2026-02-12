package models

import "time"

// BuyingList represents a shared shopping list
type BuyingList struct {
	ID        int64     `json:"id" db:"id"`
	FamilyID  int64     `json:"family_id" db:"family_id"`
	ChatID    int64     `json:"chat_id" db:"chat_id"`
	Name      string    `json:"name" db:"name"`
	CreatedByID int64   `json:"created_by_id" db:"created_by_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	Items     []BuyingItem `json:"items,omitempty"`
	CreatedBy *User     `json:"created_by,omitempty"`
}

// BuyingItem represents an item in a shopping list
type BuyingItem struct {
	ID           int64     `json:"id" db:"id"`
	BuyingListID int64     `json:"buying_list_id" db:"buying_list_id"`
	Name         string    `json:"name" db:"name"`
	Quantity     string    `json:"quantity" db:"quantity"`
	Bought       bool      `json:"bought" db:"bought"`
	BoughtByID   *int64    `json:"bought_by_id" db:"bought_by_id"`
	AddedByID    int64     `json:"added_by_id" db:"added_by_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	BoughtBy     *User     `json:"bought_by,omitempty"`
	AddedBy      *User     `json:"added_by,omitempty"`
}
