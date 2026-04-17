package models

type Good struct {
	Id     int64  `gorm:"primaryKey"`
	Name   string `gorm:"column:name"`
	Amount int64  `gorm:"column:amount"`
}

type UserState struct {
	State string
	Phase int64
}

type UserRight struct {
	ID         int64  `gorm:"column:id"`
	Permission string `gorm:"column:permission"`
}
