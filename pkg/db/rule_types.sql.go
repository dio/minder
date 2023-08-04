// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.19.1
// source: rule_types.sql

package db

import (
	"context"
	"encoding/json"
)

const createRuleType = `-- name: CreateRuleType :one
INSERT INTO rule_type (
    name,
    provider,
    group_id,
    definition) VALUES ($1, $2, $3, $4::jsonb) RETURNING id, name, provider, group_id, definition, created_at, updated_at
`

type CreateRuleTypeParams struct {
	Name       string          `json:"name"`
	Provider   string          `json:"provider"`
	GroupID    int32           `json:"group_id"`
	Definition json.RawMessage `json:"definition"`
}

func (q *Queries) CreateRuleType(ctx context.Context, arg CreateRuleTypeParams) (RuleType, error) {
	row := q.db.QueryRowContext(ctx, createRuleType,
		arg.Name,
		arg.Provider,
		arg.GroupID,
		arg.Definition,
	)
	var i RuleType
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Provider,
		&i.GroupID,
		&i.Definition,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const deleteRuleType = `-- name: DeleteRuleType :exec
DELETE FROM rule_type WHERE id = $1
`

func (q *Queries) DeleteRuleType(ctx context.Context, id int32) error {
	_, err := q.db.ExecContext(ctx, deleteRuleType, id)
	return err
}

const getRuleTypeByID = `-- name: GetRuleTypeByID :one
SELECT id, name, provider, group_id, definition, created_at, updated_at FROM rule_type WHERE id = $1
`

func (q *Queries) GetRuleTypeByID(ctx context.Context, id int32) (RuleType, error) {
	row := q.db.QueryRowContext(ctx, getRuleTypeByID, id)
	var i RuleType
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Provider,
		&i.GroupID,
		&i.Definition,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getRuleTypeByName = `-- name: GetRuleTypeByName :one
SELECT id, name, provider, group_id, definition, created_at, updated_at FROM rule_type WHERE provider = $1 AND group_id = $2 AND name = $3
`

type GetRuleTypeByNameParams struct {
	Provider string `json:"provider"`
	GroupID  int32  `json:"group_id"`
	Name     string `json:"name"`
}

func (q *Queries) GetRuleTypeByName(ctx context.Context, arg GetRuleTypeByNameParams) (RuleType, error) {
	row := q.db.QueryRowContext(ctx, getRuleTypeByName, arg.Provider, arg.GroupID, arg.Name)
	var i RuleType
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Provider,
		&i.GroupID,
		&i.Definition,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const listRuleTypesByProviderAndGroup = `-- name: ListRuleTypesByProviderAndGroup :many
SELECT id, name, provider, group_id, definition, created_at, updated_at FROM rule_type WHERE provider = $1 AND group_id = $2
`

type ListRuleTypesByProviderAndGroupParams struct {
	Provider string `json:"provider"`
	GroupID  int32  `json:"group_id"`
}

func (q *Queries) ListRuleTypesByProviderAndGroup(ctx context.Context, arg ListRuleTypesByProviderAndGroupParams) ([]RuleType, error) {
	rows, err := q.db.QueryContext(ctx, listRuleTypesByProviderAndGroup, arg.Provider, arg.GroupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []RuleType{}
	for rows.Next() {
		var i RuleType
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Provider,
			&i.GroupID,
			&i.Definition,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateRuleType = `-- name: UpdateRuleType :exec
UPDATE rule_type SET definition = $2::jsonb WHERE id = $1
`

type UpdateRuleTypeParams struct {
	ID         int32           `json:"id"`
	Definition json.RawMessage `json:"definition"`
}

func (q *Queries) UpdateRuleType(ctx context.Context, arg UpdateRuleTypeParams) error {
	_, err := q.db.ExecContext(ctx, updateRuleType, arg.ID, arg.Definition)
	return err
}