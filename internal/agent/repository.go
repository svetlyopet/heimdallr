package agent

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	FindAll(ctx context.Context, serverID string, limit int, offset int) ([]Agent, int64, error)
	FindAllGlobal(ctx context.Context, filter ListFilters, limit int, offset int) ([]AgentWithCount, int64, error)
	FindById(ctx context.Context, agentID string, serverID string) (Agent, error)
	FindByIdGlobal(ctx context.Context, agentID string) (AgentWithCount, error)
	FindByName(ctx context.Context, name string) (Agent, error)
	FindServersByAgent(ctx context.Context, agentID string, limit int, offset int) ([]LinkedServer, int64, error)
	CreateUnassigned(ctx context.Context, agent Agent) (Agent, error)
	CreateOnServer(ctx context.Context, serverID uuid.UUID, agent Agent) (Agent, error)
	AttachToServer(ctx context.Context, serverID uuid.UUID, agentIDs []uuid.UUID) error
	DetachFromServer(ctx context.Context, serverID string, agentID string) error
	DeleteGlobal(ctx context.Context, agentID string) error
}

type repository struct {
	db *gorm.DB
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
		Joins("INNER JOIN server_agents ON server_agents.agent_id = agents.id").
		Where("server_agents.server_id = ? AND agents.deleted_at IS NULL", serverID)

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

func (r repository) FindAllGlobal(ctx context.Context, filter ListFilters, limit int, offset int) ([]AgentWithCount, int64, error) {
	var agents []AgentWithCount
	var total int64

	query := r.db.WithContext(ctx).
		Table("agents").
		Where("agents.deleted_at IS NULL")

	if filter.UnassignedOnly {
		query = query.Where(`
			NOT EXISTS (
				SELECT 1 FROM server_agents
				WHERE server_agents.agent_id = agents.id
			)
		`)
	}

	if strings.TrimSpace(filter.ServerID) != "" {
		query = query.Where(`
			EXISTS (
				SELECT 1 FROM server_agents
				WHERE server_agents.agent_id = agents.id
				  AND server_agents.server_id = ?
			)
		`, strings.TrimSpace(filter.ServerID))
	}

	if strings.TrimSpace(filter.AgentID) != "" {
		query = query.Where("agents.id = ?", strings.TrimSpace(filter.AgentID))
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	findQuery := query.
		Select(agentWithCountSelectColumns()).
		Order("agents.name ASC")

	if limit > 0 {
		findQuery = findQuery.Limit(limit)
	}

	if offset > 0 {
		findQuery = findQuery.Offset(offset)
	}

	if err := findQuery.Scan(&agents).Error; err != nil {
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

	var agent Agent

	if err := r.db.WithContext(ctx).
		Table("agents").
		Select(agentSelectColumns()).
		Joins("INNER JOIN server_agents ON server_agents.agent_id = agents.id").
		Where("agents.id = ? AND server_agents.server_id = ? AND agents.deleted_at IS NULL", agentID, serverID).
		Take(&agent).Error; err != nil {
		return Agent{}, err
	}

	return agent, nil
}

func (r repository) FindByIdGlobal(ctx context.Context, agentID string) (AgentWithCount, error) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return AgentWithCount{}, gorm.ErrRecordNotFound
	}

	var agent AgentWithCount

	if err := r.db.WithContext(ctx).
		Table("agents").
		Select(agentWithCountSelectColumns()).
		Where("agents.id = ? AND agents.deleted_at IS NULL", agentID).
		Take(&agent).Error; err != nil {
		return AgentWithCount{}, err
	}

	return agent, nil
}

func (r repository) FindByName(ctx context.Context, name string) (Agent, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Agent{}, gorm.ErrRecordNotFound
	}

	var agent Agent

	if err := r.db.WithContext(ctx).
		Where("name = ? AND deleted_at IS NULL", name).
		Take(&agent).Error; err != nil {
		return Agent{}, err
	}

	return agent, nil
}

func (r repository) FindServersByAgent(ctx context.Context, agentID string, limit int, offset int) ([]LinkedServer, int64, error) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return nil, 0, gorm.ErrRecordNotFound
	}

	var servers []LinkedServer
	var total int64

	query := r.db.WithContext(ctx).
		Table("servers").
		Joins("INNER JOIN server_agents ON server_agents.server_id = servers.id").
		Where("server_agents.agent_id = ? AND servers.deleted_at IS NULL", agentID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	findQuery := query.
		Select("servers.id, servers.hostname").
		Order("servers.hostname ASC")

	if limit > 0 {
		findQuery = findQuery.Limit(limit)
	}

	if offset > 0 {
		findQuery = findQuery.Offset(offset)
	}

	if err := findQuery.Scan(&servers).Error; err != nil {
		return nil, 0, err
	}

	return servers, total, nil
}

func (r repository) CreateUnassigned(ctx context.Context, agent Agent) (Agent, error) {
	if err := r.db.WithContext(ctx).Create(&agent).Error; err != nil {
		return Agent{}, err
	}

	created, err := r.FindByIdGlobal(ctx, agent.ID.String())
	if err != nil {
		return Agent{}, err
	}

	return created.Agent, nil
}

func (r repository) CreateOnServer(ctx context.Context, serverID uuid.UUID, agent Agent) (Agent, error) {
	if serverID == uuid.Nil {
		return Agent{}, gorm.ErrRecordNotFound
	}

	var created Agent

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&agent).Error; err != nil {
			return err
		}

		link := ServerAgent{
			ServerID:  serverID,
			AgentID:   agent.ID,
			CreatedAt: time.Now().UTC(),
		}

		if err := tx.Create(&link).Error; err != nil {
			return err
		}

		if err := tx.Table("agents").
			Select(agentSelectColumns()).
			Where("agents.id = ? AND agents.deleted_at IS NULL", agent.ID).
			Take(&created).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return Agent{}, err
	}

	return created, nil
}

func (r repository) AttachToServer(ctx context.Context, serverID uuid.UUID, agentIDs []uuid.UUID) error {
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

			var count int64
			if err := tx.Model(&ServerAgent{}).
				Where("server_id = ? AND agent_id = ?", serverID, agentID).
				Count(&count).Error; err != nil {
				return err
			}

			if count > 0 {
				return ErrAgentAlreadyLinked
			}

			link := ServerAgent{
				ServerID:  serverID,
				AgentID:   agentID,
				CreatedAt: time.Now().UTC(),
			}

			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r repository) DetachFromServer(ctx context.Context, serverID string, agentID string) error {
	result := r.db.WithContext(ctx).
		Where("server_id = ? AND agent_id = ?", serverID, agentID).
		Delete(&ServerAgent{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r repository) DeleteGlobal(ctx context.Context, agentID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("agent_id = ?", agentID).Delete(&ServerAgent{}).Error; err != nil {
			return err
		}

		result := tx.Where("id = ?", agentID).Delete(&Agent{})
		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return nil
	})
}

func agentSelectColumns() string {
	return `
		agents.id,
		agents.name,
		agents.type,
		agents.version,
		agents.metadata,
		agents.created_at,
		agents.updated_at
	`
}

func agentWithCountSelectColumns() string {
	return `
		agents.id,
		agents.name,
		agents.type,
		agents.version,
		agents.metadata,
		agents.created_at,
		agents.updated_at,
		(
			SELECT COUNT(*)
			FROM server_agents
			WHERE server_agents.agent_id = agents.id
		) AS server_count
	`
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "unique constraint")
}
