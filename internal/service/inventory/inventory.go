package inventory

import (
	"walmart-inventory-manager/internal/entities"
	"walmart-inventory-manager/internal/repositories/inventory"
)

type InventoryDefault struct {
	rp inventory.InventoryRepository
}

func NewInventoryDefault(rp inventory.InventoryRepository) *InventoryDefault {
	return &InventoryDefault{rp: rp}
}

func (s *InventoryDefault) FindAll() ([]entities.Product, error) {
	return s.rp.FindAll()
}
