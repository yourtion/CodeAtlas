package models

import (
	"testing"
)

// Unit tests for repository functions that don't require database

func TestNewRepositoryRepository(t *testing.T) {
	db := &DB{} // Mock DB
	repo := NewRepositoryRepository(db)
	
	if repo == nil {
		t.Fatal("Expected non-nil repository")
	}
	
	if repo.db != db {
		t.Error("Expected repository to store DB reference")
	}
}

func TestNewFileRepository(t *testing.T) {
	db := &DB{}
	repo := NewFileRepository(db)
	
	if repo == nil {
		t.Fatal("Expected non-nil repository")
	}
	
	if repo.db != db {
		t.Error("Expected repository to store DB reference")
	}
}

func TestNewSymbolRepository(t *testing.T) {
	db := &DB{}
	repo := NewSymbolRepository(db)
	
	if repo == nil {
		t.Fatal("Expected non-nil repository")
	}
	
	if repo.db != db {
		t.Error("Expected repository to store DB reference")
	}
}

func TestNewEdgeRepository(t *testing.T) {
	db := &DB{}
	repo := NewEdgeRepository(db)
	
	if repo == nil {
		t.Fatal("Expected non-nil repository")
	}
	
	if repo.db != db {
		t.Error("Expected repository to store DB reference")
	}
}

func TestNewASTNodeRepository(t *testing.T) {
	db := &DB{}
	repo := NewASTNodeRepository(db)
	
	if repo == nil {
		t.Fatal("Expected non-nil repository")
	}
	
	if repo.db != db {
		t.Error("Expected repository to store DB reference")
	}
}

func TestNewVectorRepository(t *testing.T) {
	db := &DB{}
	repo := NewVectorRepository(db)
	
	if repo == nil {
		t.Fatal("Expected non-nil repository")
	}
	
	if repo.db != db {
		t.Error("Expected repository to store DB reference")
	}
}

func TestNewSchemaManager(t *testing.T) {
	db := &DB{}
	manager := NewSchemaManager(db)
	
	if manager == nil {
		t.Fatal("Expected non-nil schema manager")
	}
	
	if manager.db != db {
		t.Error("Expected schema manager to store DB reference")
	}
}

func TestNewTransactionManager(t *testing.T) {
	db := &DB{}
	manager := NewTransactionManager(db)
	
	if manager == nil {
		t.Fatal("Expected non-nil transaction manager")
	}
	
	if manager.db != db {
		t.Error("Expected transaction manager to store DB reference")
	}
}
