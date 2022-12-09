package models

type User struct {
	Username     string `gorm:"primaryKey"`
	PasswordHash string
}
