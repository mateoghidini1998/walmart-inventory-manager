package inventory

import "walmart-inventory-manager/internal/entities"

type InventoryRepository interface {
	FindAll() ([]entities.Product, error)
	InsertProduct(product entities.Product) (int64, error)
	InsertWmtProductDetail(productID int64, product entities.Product) error
	InsertProductImage(gtin, imageUrl string) error
	GetFirstProductByMarketplaceID(marketplaceID int) (*entities.Product, error)
	GetAllProductsByMarketplaceID(marketplaceID int) ([]*entities.Product, error)
	UpdateListingStatus(productID int64, listingStatusID int) error
	GetProductBySKU(sku string) (*entities.Product, error)
	GetProductByWPID(wpid string) (*entities.Product, error)
	UpdateProduct(product entities.Product) error
	UpdateWmtProductDetail(productID int64, product entities.Product) error
}
