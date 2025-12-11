package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                  uuid.UUID `json:"id" gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Username            string    `json:"username" gorm:"uniqueIndex;not null"`
	Email               string    `json:"email" gorm:"uniqueIndex;not null"`
	Password            string    `json:"-" gorm:"not null"`
	PublicKey           string    `json:"publicKey" gorm:"type:text"`           // Visible to everyone
	EncryptedPrivateKey string    `json:"encryptedPrivateKey" gorm:"type:text"` // JSON blob: { key, iv }
	CreatedAt           time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt           time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}
