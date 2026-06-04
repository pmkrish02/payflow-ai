package service

import (
	"golang.org/x/crypto/bcrypt"
	"github.com/pmkrish02/payflow-ai/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type AuthService struct {
    AuthRepo *repository.AuthRepository
}

func (s *AuthService) Register(email, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	err = s.AuthRepo.CreateUser(email, string(hash))
	if err != nil {
		return err
	}
	return nil

}

func (s *AuthService) Login(email, password string) (string, error) {
	userdata,err := s.AuthRepo.GetUserByEmail(email)
	if err != nil {
		return "",err
	}
	
	err = bcrypt.CompareHashAndPassword([]byte(userdata.PasswordHash), []byte(password))
	if err != nil {
		return "",err
	}
	
	mySigningKey := []byte("your-security-key")
	
	claims := &jwt.RegisteredClaims{
	ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	Issuer:    "test",
	Subject: userdata.ID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString(mySigningKey)
	return ss, err


}