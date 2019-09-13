package main

// User represents a Telegram user
type User struct {
	TelegramID int  `gorm:"unique;not null;primary_key;auto_increment:false"`
	OptedOut   bool `gorm:"default:false"`
}
