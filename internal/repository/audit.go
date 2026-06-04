package repository

import(
	"github.com/jackc/pgx/v5/pgxpool"
	"context"
)

type AuditRepository struct{
	DB *pgxpool.Pool
}

func (r *AuditRepository) CreateAuditLog(ctx context.Context, userID, action, resourceType string, resourceID *string, ipAddress *string) error{
	_, err := r.DB.Exec(ctx, "INSERT INTO audit_log (user_id, action, resource_type, resource_id, ip_address) VALUES ($1, $2, $3, $4, $5)", userID, action, resourceType, resourceID, ipAddress)
	if err != nil {
		return err
	}
	return nil

}

