package model

import "time"

// AccessToken represents a personal access token
type AccessToken struct {
	AccessTokenID  int       `gorm:"primaryKey;autoIncrement;column:access_token_id" json:"accessTokenId"`
	Token          string    `gorm:"column:token;size:255;uniqueIndex" json:"token"`
	UserName       string    `gorm:"column:user_name;index" json:"userName"`
	Note           string    `gorm:"column:note" json:"note"`
	RegisteredDate time.Time `gorm:"column:registered_date" json:"registeredDate"`
}

func (AccessToken) TableName() string { return "access_token" }
