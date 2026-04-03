package models

type UserCreds struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type User struct {
	ID           string
	Login        string
	PasswordHash string
}
