package repository

import(
	
	"github.com/jackc/pgx/v5/pgxpool"
	"context"
	"github.com/pmkrish02/payflow-ai/internal/model"
)

type AuthRepository struct {
    DB *pgxpool.Pool
}

func (r *AuthRepository) CreateUser(email, passwordHash string) error {
	_, err := r.DB.Exec(context.Background(), "INSERT INTO users (email, password_hash) VALUES ($1, $2)", email, passwordHash)
	if err != nil {
		return err
	}
	return nil
}
func (r *AuthRepository) GetUserByEmail(email string) (*model.User, error){
	user := &model.User{}
	err := r.DB.QueryRow(context.Background(), "SELECT id, email, password_hash, role, created_at, updated_at FROM users WHERE email = $1", email).Scan(
    &user.ID,
    &user.Email,
    &user.PasswordHash,
    &user.Role,
    &user.CreatedAt,
    &user.UpdatedAt,
)
	if err != nil {
		return nil,err
	}
	return user,nil


}
