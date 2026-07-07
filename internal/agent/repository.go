package agent

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	FindAll(ctx context.Context, serverID string, limit int, offset int) ([]Agent, int64, error)
	FindAllGlobal(ctx context.Context, unassignedOnly bool, limit int, offset int) ([]Agent, int64, error)
	FindById(ctx context.Context, agentID string, serverID string) (Agent, error)
	FindByIdGlobal(ctx context.Context, agentID string) (Agent, error)
	CreateUnassigned(ctx context.Context, agent Agent) (Agent, error)
	CreateOnServer(ctx context.Context, agent Agent) (Agent, error)
	AttachToServer(ctx context.Context, serverID uuid.UUID, hostname string, agentIDs []uuid.UUID) error
	Delete(ctx context.Context, serverID string, agentID string) error
}

type repository struct {
	db *gorm.DB
}

type serverRelation struct {
	ServerID uuid.UUID
	Server   string
}

func (r repository) FindAll(ctx context.Context, serverID string, limit int, offset int) ([]Agent, int64, error) {
	serverID = strings.TrimSpace(serverID)
	if serverID == "" {
		return nil, 0, gorm.ErrRecordNotFound
	}

	var agents []Agent
	var total int64

	query := r.db.WithContext(ctx).
		Table("agents").
		Joins("JOIN servers ON servers.id = agents.server_id").
		Where("agents.server_id = ? AND agents.deleted_at IS NULL", serverID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	findQuery := query.
		Select(agentSelectColumns()).
		Order("agents.name ASC")

	if limit > 0 {
		findQuery = findQuery.Limit(limit)
	}

	if offset > 0 {
		findQuery = findQuery.Offset(offset)
	}

	if err := findQuery.Find(&agents).Error; err != nil {
		return nil, 0, err
	}

	return agents, total, nil
}

func (r repository) FindAllGlobal(ctx context.Context, unassignedOnly bool, limit int, offset int) ([]Agent, int64, error) {
	var agents []Agent
	var total int64

	query := r.db.WithContext(ctx).
		Table("agents").
		Joins("LEFT JOIN servers ON servers.id = agents.server_id").
		Where("agents.deleted_at IS NULL")

	if unassignedOnly {
		query = query.Where("agents.server_id IS NULL")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	findQuery := query.
		Select(agentSelectColumns()).
		Order("agents.name ASC")

	if limit > 0 {
		findQuery = findQuery.Limit(limit)
	}

	if offset > 0 {
		findQuery = findQuery.Offset(offset)
	}

	if err := findQuery.Find(&agents).Error; err != nil {
		return nil, 0, err
	}

	return agents, total, nil
}

func (r repository) FindById(ctx context.Context, agentID string, serverID string) (Agent, error) {
	agentID = strings.TrimSpace(agentID)
	serverID = strings.TrimSpace(serverID)

	if agentID == "" || serverID == "" {
		return Agent{}, gorm.ErrRecordNotFound
	}

	return findAgentById(ctx, r.db, agentID, serverID)
}

func (r repository) FindByIdGlobal(ctx context.Context, agentID string) (Agent, error) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return Agent{}, gorm.ErrRecordNotFound
	}

	var agent Agent

	if err := r.db.WithContext(ctx).
		Table("agents").
		Select(agentSelectColumns()).
		Joins("LEFT JOIN servers ON servers.id = agents.server_id").
		Where("agents.id = ? AND agents.deleted_at IS NULL", agentID).
		Take(&agent).Error; err != nil {
		return Agent{}, err
	}

	return agent, nil
}

func (r repository) CreateUnassigned(ctx context.Context, agent Agent) (Agent, error) {
	agent.ServerID = nil
	agent.Server = ""

	if err := r.db.WithContext(ctx).Create(&agent).Error; err != nil {
		return Agent{}, err
	}

	return r.FindByIdGlobal(ctx, agent.ID.String())
}

func (r repository) CreateOnServer(ctx context.Context, agent Agent) (Agent, error) {
	if agent.ServerID == nil || *agent.ServerID == uuid.Nil {
		return Agent{}, gorm.ErrRecordNotFound
	}

	var returned Agent

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		relation, err := findServerRelation(ctx, tx, *agent.ServerID)
		if err != nil {
			return err
		}

		agent.Server = relation.Server
		serverID := relation.ServerID
		agent.ServerID = &serverID

		if err := tx.Create(&agent).Error; err != nil {
			return err
		}

		created, err := findAgentById(ctx, tx, agent.ID.String(), agent.ServerID.String())
		if err != nil {
			return err
		}

		returned = created
		return nil
	})

	if err != nil {
		return Agent{}, err
	}

	return returned, nil
}

func (r repository) AttachToServer(ctx context.Context, serverID uuid.UUID, hostname string, agentIDs []uuid.UUID) error {
	if serverID == uuid.Nil || len(agentIDs) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, agentID := range agentIDs {
			if agentID == uuid.Nil {
				return gorm.ErrRecordNotFound
			}

			var existing Agent
			if err := tx.Where("id = ? AND deleted_at IS NULL", agentID).Take(&existing).Error; err != nil {
				return err
			}

			if existing.ServerID != nil {
				return ErrAgentAlreadyAssigned
			}

			result := tx.Model(&Agent{}).
				Where("id = ? AND server_id IS NULL AND deleted_at IS NULL", agentID).
				Updates(map[string]any{
					"server_id": serverID,
					"server":    hostname,
				})

			if result.Error != nil {
				return result.Error
			}

			if result.RowsAffected == 0 {
				return ErrAgentAlreadyAssigned
			}
		}

		return nil
	})
}

func (r repository) Delete(ctx context.Context, serverID string, agentID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND server_id = ?", agentID, serverID).
		Delete(&Agent{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func findServerRelation(ctx context.Context, db *gorm.DB, serverID uuid.UUID) (serverRelation, error) {
	if serverID == uuid.Nil {
		return serverRelation{}, gorm.ErrRecordNotFound
	}

	var relation serverRelation

	if err := db.WithContext(ctx).
		Table("servers").
		Select("servers.id AS server_id, servers.hostname AS server").
		Where("servers.id = ?", serverID).
		Take(&relation).Error; err != nil {
		return serverRelation{}, err
	}

	if relation.ServerID == uuid.Nil || relation.Server == "" {
		return serverRelation{}, gorm.ErrRecordNotFound
	}

	return relation, nil
}

func findAgentById(ctx context.Context, db *gorm.DB, agentID string, serverID string) (Agent, error) {
	var agent Agent

	if err := db.WithContext(ctx).
		Table("agents").
		Select(agentSelectColumns()).
		Joins("JOIN servers ON servers.id = agents.server_id").
		Where("agents.id = ? AND agents.server_id = ? AND agents.deleted_at IS NULL", agentID, serverID).
		Take(&agent).Error; err != nil {
		return Agent{}, err
	}

	return agent, nil
}

func agentSelectColumns() string {
	return `
		agents.id,
		COALESCE(servers.hostname, agents.server, '') AS server,
		agents.server_id,
		agents.name,
		agents.type,
		agents.version,
		agents.metadata,
		agents.created_at,
		agents.updated_at
	`
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}
