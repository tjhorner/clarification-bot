package main

// User represents a Telegram user
type User struct {
	TelegramID int  `gorm:"unique;not null;primary_key"`
	OptedOut   bool `gorm:"default:false"`
}
