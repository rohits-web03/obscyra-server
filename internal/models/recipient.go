package models

import (
	"time"

	"github.com/google/uuid"
)

type Recipient struct {
    ID           uuid.UUID `json:"id" gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
    TransferID   uuid.UUID `json:"transferId" gorm:"type:uuid;not null;index"`
    ReceiverID   uuid.UUID `json:"receiverId" gorm:"type:uuid;not null;index"` // The UserID of receiver
    EncryptedKey string    `json:"encryptedKey" gorm:"type:text;not null"` 
    CreatedAt    time.Time `json:"createdAt" gorm:"autoCreateTime"`
}