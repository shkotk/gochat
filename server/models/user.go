package models

type User struct {
	Username     string `gorm:"primaryKey;default:null"`
	PasswordHash string `gorm:"not null;default:null"`
}
