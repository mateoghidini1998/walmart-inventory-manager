package inventory

import "walmart-inventory-manager/internal/entities"

type InventoryService interface {
	FindAll() ([]entities.Product, error)
}
