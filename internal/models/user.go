package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleManager  Role = "manager"
	RoleCustomer Role = "customer"
)

type User struct {
	gorm.Model
	Email        string    `json:"email" gorm:"uniqueIndex"`
	Password     string    `json:"-"`
	Role         Role      `json:"role"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	LastLogin    time.Time `json:"last_login"`
	IsActive     bool      `json:"is_active" gorm:"default:true"`
	RefreshToken string    `json:"-"`
}

func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

type Permission struct {
	ID          uint   `gorm:"primarykey"`
	Name        string `gorm:"uniqueIndex"`
	Description string
}

type RolePermission struct {
	RoleID       string `gorm:"primarykey"`
	PermissionID uint   `gorm:"primarykey"`
}
