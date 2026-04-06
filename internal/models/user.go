package models

type UserCreds struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type User struct {
	UserID       string
	Login        string
	PasswordHash string
}
