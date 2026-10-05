package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RoleRepository mendefinisikan operasi pembacaan pemetaan role dan permission dari basis data.
type RoleRepository interface {
	GetRolePermissions(ctx context.Context) (map[string][]string, error)
}

type rolePostgresRepository struct {
	pool *pgxpool.Pool
}

// NewRoleRepository membuat instansiasi RoleRepository berbasis PostgreSQL.
func NewRoleRepository(pool *pgxpool.Pool) RoleRepository {
	return &rolePostgresRepository{pool: pool}
}

// GetRolePermissions membaca seluruh relasi role_permissions dari database
// dan memetakannya menjadi map[role_id][]permission_id untuk dimuat ke memori.
func (r *rolePostgresRepository) GetRolePermissions(ctx context.Context) (map[string][]string, error) {
	query := `SELECT role_id, permission_id FROM role_permissions ORDER BY role_id, permission_id`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query role_permissions: %w", err)
	}
	defer rows.Close()

	mapping := make(map[string][]string)
	for rows.Next() {
		var roleID, permissionID string
		if err := rows.Scan(&roleID, &permissionID); err != nil {
			return nil, fmt.Errorf("scan role_permission: %w", err)
		}
		mapping[roleID] = append(mapping[roleID], permissionID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi role_permissions: %w", err)
	}

	return mapping, nil
}
