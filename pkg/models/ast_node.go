package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ASTNode represents an AST node entity in the knowledge graph
type ASTNode struct {
	NodeID     string            `json:"node_id" db:"node_id"`
	FileID     string            `json:"file_id" db:"file_id"`
	Type       string            `json:"type" db:"type"`
	ParentID   *string           `json:"parent_id" db:"parent_id"`
	StartLine  int               `json:"start_line" db:"start_line"`
	EndLine    int               `json:"end_line" db:"end_line"`
	StartByte  int               `json:"start_byte" db:"start_byte"`
	EndByte    int               `json:"end_byte" db:"end_byte"`
	Text       string            `json:"text" db:"text"`
	Attributes map[string]string `json:"attributes" db:"attributes"`
	CreatedAt  time.Time         `json:"created_at" db:"created_at"`
}

// ASTNodeRepository handles CRUD operations for AST nodes
type ASTNodeRepository struct {
	db *DB
}

// NewASTNodeRepository creates a new AST node repository
func NewASTNodeRepository(db *DB) *ASTNodeRepository {
	return &ASTNodeRepository{db: db}
}

// Create inserts a new AST node record
func (r *ASTNodeRepository) Create(ctx context.Context, node *ASTNode) error {
	query := `
		INSERT INTO ast_nodes (node_id, file_id, type, parent_id, start_line, end_line,
			start_byte, end_byte, text, attributes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	node.CreatedAt = time.Now()

	// Convert attributes to JSON
	var attributesJSON []byte
	var err error
	if node.Attributes != nil {
		attributesJSON, err = json.Marshal(node.Attributes)
		if err != nil {
			return fmt.Errorf("failed to marshal attributes: %w", err)
		}
	}

	_, err = r.db.ExecContext(ctx, query,
		node.NodeID, node.FileID, node.Type, node.ParentID,
		node.StartLine, node.EndLine, node.StartByte, node.EndByte,
		node.Text, attributesJSON, node.CreatedAt)
	return err
}

// GetByID retrieves an AST node by its ID
func (r *ASTNodeRepository) GetByID(ctx context.Context, nodeID string) (*ASTNode, error) {
	query := `
		SELECT node_id, file_id, type, parent_id, start_line, end_line,
			start_byte, end_byte, text, attributes, created_at
		FROM ast_nodes WHERE node_id = $1
	`
	var node ASTNode
	var attributesJSON []byte
	err := r.db.QueryRowContext(ctx, query, nodeID).Scan(
		&node.NodeID, &node.FileID, &node.Type, &node.ParentID,
		&node.StartLine, &node.EndLine, &node.StartByte, &node.EndByte,
		&node.Text, &attributesJSON, &node.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Unmarshal attributes
	if attributesJSON != nil {
		err = json.Unmarshal(attributesJSON, &node.Attributes)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
		}
	}

	return &node, nil
}

// GetByFileID retrieves all AST nodes for a file
func (r *ASTNodeRepository) GetByFileID(ctx context.Context, fileID string) ([]*ASTNode, error) {
	query := `
		SELECT node_id, file_id, type, parent_id, start_line, end_line,
			start_byte, end_byte, text, attributes, created_at
		FROM ast_nodes WHERE file_id = $1 ORDER BY start_line, start_byte
	`
	rows, err := r.db.QueryContext(ctx, query, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*ASTNode
	for rows.Next() {
		var node ASTNode
		var attributesJSON []byte
		err := rows.Scan(
			&node.NodeID, &node.FileID, &node.Type, &node.ParentID,
			&node.StartLine, &node.EndLine, &node.StartByte, &node.EndByte,
			&node.Text, &attributesJSON, &node.CreatedAt)
		if err != nil {
			return nil, err
		}

		// Unmarshal attributes
		if attributesJSON != nil {
			err = json.Unmarshal(attributesJSON, &node.Attributes)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
			}
		}

		nodes = append(nodes, &node)
	}
	return nodes, rows.Err()
}

// GetByParentID retrieves child nodes for a parent node
func (r *ASTNodeRepository) GetByParentID(ctx context.Context, parentID string) ([]*ASTNode, error) {
	query := `
		SELECT node_id, file_id, type, parent_id, start_line, end_line,
			start_byte, end_byte, text, attributes, created_at
		FROM ast_nodes WHERE parent_id = $1 ORDER BY start_line, start_byte
	`
	rows, err := r.db.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*ASTNode
	for rows.Next() {
		var node ASTNode
		var attributesJSON []byte
		err := rows.Scan(
			&node.NodeID, &node.FileID, &node.Type, &node.ParentID,
			&node.StartLine, &node.EndLine, &node.StartByte, &node.EndByte,
			&node.Text, &attributesJSON, &node.CreatedAt)
		if err != nil {
			return nil, err
		}

		// Unmarshal attributes
		if attributesJSON != nil {
			err = json.Unmarshal(attributesJSON, &node.Attributes)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
			}
		}

		nodes = append(nodes, &node)
	}
	return nodes, rows.Err()
}

// GetRootNodes retrieves nodes that have no parent (root nodes)
func (r *ASTNodeRepository) GetRootNodes(ctx context.Context, fileID string) ([]*ASTNode, error) {
	query := `
		SELECT node_id, file_id, type, parent_id, start_line, end_line,
			start_byte, end_byte, text, attributes, created_at
		FROM ast_nodes WHERE file_id = $1 AND parent_id IS NULL 
		ORDER BY start_line, start_byte
	`
	rows, err := r.db.QueryContext(ctx, query, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*ASTNode
	for rows.Next() {
		var node ASTNode
		var attributesJSON []byte
		err := rows.Scan(
			&node.NodeID, &node.FileID, &node.Type, &node.ParentID,
			&node.StartLine, &node.EndLine, &node.StartByte, &node.EndByte,
			&node.Text, &attributesJSON, &node.CreatedAt)
		if err != nil {
			return nil, err
		}

		// Unmarshal attributes
		if attributesJSON != nil {
			err = json.Unmarshal(attributesJSON, &node.Attributes)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
			}
		}

		nodes = append(nodes, &node)
	}
	return nodes, rows.Err()
}

// GetByType retrieves nodes filtered by type
func (r *ASTNodeRepository) GetByType(ctx context.Context, fileID, nodeType string) ([]*ASTNode, error) {
	query := `
		SELECT node_id, file_id, type, parent_id, start_line, end_line,
			start_byte, end_byte, text, attributes, created_at
		FROM ast_nodes WHERE file_id = $1 AND type = $2 
		ORDER BY start_line, start_byte
	`
	rows, err := r.db.QueryContext(ctx, query, fileID, nodeType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*ASTNode
	for rows.Next() {
		var node ASTNode
		var attributesJSON []byte
		err := rows.Scan(
			&node.NodeID, &node.FileID, &node.Type, &node.ParentID,
			&node.StartLine, &node.EndLine, &node.StartByte, &node.EndByte,
			&node.Text, &attributesJSON, &node.CreatedAt)
		if err != nil {
			return nil, err
		}

		// Unmarshal attributes
		if attributesJSON != nil {
			err = json.Unmarshal(attributesJSON, &node.Attributes)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
			}
		}

		nodes = append(nodes, &node)
	}
	return nodes, rows.Err()
}

// Update updates an existing AST node record
func (r *ASTNodeRepository) Update(ctx context.Context, node *ASTNode) error {
	query := `
		UPDATE ast_nodes 
		SET type = $3, parent_id = $4, start_line = $5, end_line = $6,
			start_byte = $7, end_byte = $8, text = $9, attributes = $10
		WHERE node_id = $1 AND file_id = $2
	`

	// Convert attributes to JSON
	var attributesJSON []byte
	var err error
	if node.Attributes != nil {
		attributesJSON, err = json.Marshal(node.Attributes)
		if err != nil {
			return fmt.Errorf("failed to marshal attributes: %w", err)
		}
	}

	result, err := r.db.ExecContext(ctx, query,
		node.NodeID, node.FileID, node.Type, node.ParentID,
		node.StartLine, node.EndLine, node.StartByte, node.EndByte,
		node.Text, attributesJSON)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("AST node not found: %s", node.NodeID)
	}
	return nil
}

// Delete removes an AST node record
func (r *ASTNodeRepository) Delete(ctx context.Context, nodeID string) error {
	query := `DELETE FROM ast_nodes WHERE node_id = $1`
	result, err := r.db.ExecContext(ctx, query, nodeID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("AST node not found: %s", nodeID)
	}
	return nil
}

// BatchCreate inserts multiple AST nodes preserving parent-child relationships
func (r *ASTNodeRepository) BatchCreate(ctx context.Context, nodes []*ASTNode) error {
	if len(nodes) == 0 {
		return nil
	}

	query := `
		INSERT INTO ast_nodes (node_id, file_id, type, parent_id, start_line, end_line,
			start_byte, end_byte, text, attributes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (node_id) 
		DO UPDATE SET 
			type = EXCLUDED.type,
			parent_id = EXCLUDED.parent_id,
			start_line = EXCLUDED.start_line,
			end_line = EXCLUDED.end_line,
			start_byte = EXCLUDED.start_byte,
			end_byte = EXCLUDED.end_byte,
			text = EXCLUDED.text,
			attributes = EXCLUDED.attributes
	`

	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare batch insert statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, node := range nodes {
		node.CreatedAt = now

		// Convert attributes to JSON
		var attributesJSON []byte
		if node.Attributes != nil {
			attributesJSON, err = json.Marshal(node.Attributes)
			if err != nil {
				return fmt.Errorf("failed to marshal attributes for node %s: %w", node.NodeID, err)
			}
		}

		_, err := stmt.ExecContext(ctx,
			node.NodeID, node.FileID, node.Type, node.ParentID,
			node.StartLine, node.EndLine, node.StartByte, node.EndByte,
			node.Text, attributesJSON, node.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert AST node %s: %w", node.NodeID, err)
		}
	}

	return nil
}

// BatchCreateTx inserts multiple AST nodes within a transaction
func (r *ASTNodeRepository) BatchCreateTx(ctx context.Context, tx *sql.Tx, nodes []*ASTNode) error {
	if len(nodes) == 0 {
		return nil
	}

	query := `
		INSERT INTO ast_nodes (node_id, file_id, type, parent_id, start_line, end_line,
			start_byte, end_byte, text, attributes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (node_id) 
		DO UPDATE SET 
			type = EXCLUDED.type,
			parent_id = EXCLUDED.parent_id,
			start_line = EXCLUDED.start_line,
			end_line = EXCLUDED.end_line,
			start_byte = EXCLUDED.start_byte,
			end_byte = EXCLUDED.end_byte,
			text = EXCLUDED.text,
			attributes = EXCLUDED.attributes
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare batch insert statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, node := range nodes {
		node.CreatedAt = now

		// Convert attributes to JSON
		var attributesJSON []byte
		if node.Attributes != nil {
			attributesJSON, err = json.Marshal(node.Attributes)
			if err != nil {
				return fmt.Errorf("failed to marshal attributes for node %s: %w", node.NodeID, err)
			}
		}

		_, err := stmt.ExecContext(ctx,
			node.NodeID, node.FileID, node.Type, node.ParentID,
			node.StartLine, node.EndLine, node.StartByte, node.EndByte,
			node.Text, attributesJSON, node.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert AST node %s: %w", node.NodeID, err)
		}
	}

	return nil
}

// DeleteByFileID removes all AST nodes for a file
func (r *ASTNodeRepository) DeleteByFileID(ctx context.Context, fileID string) error {
	query := `DELETE FROM ast_nodes WHERE file_id = $1`
	_, err := r.db.ExecContext(ctx, query, fileID)
	return err
}

// Count returns the total number of AST nodes for a file
func (r *ASTNodeRepository) Count(ctx context.Context, fileID string) (int64, error) {
	query := `SELECT COUNT(*) FROM ast_nodes WHERE file_id = $1`
	var count int64
	err := r.db.QueryRowContext(ctx, query, fileID).Scan(&count)
	return count, err
}

// CountByType returns the count of AST nodes by type for a file
func (r *ASTNodeRepository) CountByType(ctx context.Context, fileID string) (map[string]int64, error) {
	query := `
		SELECT type, COUNT(*) 
		FROM ast_nodes 
		WHERE file_id = $1 
		GROUP BY type
	`
	rows, err := r.db.QueryContext(ctx, query, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int64)
	for rows.Next() {
		var nodeType string
		var count int64
		err := rows.Scan(&nodeType, &count)
		if err != nil {
			return nil, err
		}
		counts[nodeType] = count
	}
	return counts, rows.Err()
}

// GetNodeHierarchy retrieves a node and all its descendants
func (r *ASTNodeRepository) GetNodeHierarchy(ctx context.Context, nodeID string) ([]*ASTNode, error) {
	query := `
		WITH RECURSIVE node_hierarchy AS (
			-- Base case: start with the specified node
			SELECT node_id, file_id, type, parent_id, start_line, end_line,
				start_byte, end_byte, text, attributes, created_at, 0 as level
			FROM ast_nodes 
			WHERE node_id = $1
			
			UNION ALL
			
			-- Recursive case: find children
			SELECT n.node_id, n.file_id, n.type, n.parent_id, n.start_line, n.end_line,
				n.start_byte, n.end_byte, n.text, n.attributes, n.created_at, h.level + 1
			FROM ast_nodes n
			INNER JOIN node_hierarchy h ON n.parent_id = h.node_id
		)
		SELECT node_id, file_id, type, parent_id, start_line, end_line,
			start_byte, end_byte, text, attributes, created_at
		FROM node_hierarchy
		ORDER BY level, start_line, start_byte
	`
	rows, err := r.db.QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*ASTNode
	for rows.Next() {
		var node ASTNode
		var attributesJSON []byte
		err := rows.Scan(
			&node.NodeID, &node.FileID, &node.Type, &node.ParentID,
			&node.StartLine, &node.EndLine, &node.StartByte, &node.EndByte,
			&node.Text, &attributesJSON, &node.CreatedAt)
		if err != nil {
			return nil, err
		}

		// Unmarshal attributes
		if attributesJSON != nil {
			err = json.Unmarshal(attributesJSON, &node.Attributes)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
			}
		}

		nodes = append(nodes, &node)
	}
	return nodes, rows.Err()
}
