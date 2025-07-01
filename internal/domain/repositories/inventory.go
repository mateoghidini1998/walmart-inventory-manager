package repositories

import (
	"walmart-inventory-manager/internal/domain/entities"
)

type InventoryRepository interface {
	FindAll() ([]entities.Product, error)
	InsertProduct(product entities.Product) (int64, error)
	InsertWmtProductDetail(productId int64, product entities.Product) error
	InsertProductImage(gtin, imageUrl string) error
	GetFirstProductByMarketplaceID(marketplaceID int) (*entities.Product, error)
}
