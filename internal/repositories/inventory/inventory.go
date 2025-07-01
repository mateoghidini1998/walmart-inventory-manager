package inventory

import (
	"database/sql"
	"fmt"
	"walmart-inventory-manager/internal/entities"
)

type inventoryRepository struct {
	db *sql.DB
}

func NewInventoryRepository(db *sql.DB) *inventoryRepository {
	return &inventoryRepository{
		db: db,
	}
}

func (r *inventoryRepository) FindAll() ([]entities.Product, error) {
	query := `
		SELECT 
			p.id,
			p.seller_sku,
			p.upc,
			p.product_name,
			d.price,
			d.available_to_sell_qty,
			d.gtin
		FROM products p
		INNER JOIN wmt_product_details d ON p.id = d.product_id
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []entities.Product

	for rows.Next() {
		var p entities.Product
		var upc sql.NullString
		err := rows.Scan(
			&p.ID,
			&p.SKU,
			&upc,
			&p.ProductName,
			&p.Price,
			&p.AvailableToSellQTY,
			&p.GTIN,
		)
		if err != nil {
			return nil, err
		}
		if upc.Valid {
			p.UPC = upc.String
		} else {
			p.UPC = ""
		}
		products = append(products, p)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}

func (r *inventoryRepository) InsertProduct(product entities.Product) (int64, error) {
	query := `
		INSERT INTO products (product_name, product_image, supplier_id, supplier_item_number, product_cost, upc, marketplace_id, seller_sku, createdAt, updatedAt)
		VALUES (?, NULL, NULL, NULL, NULL, ?, 2, ?, NOW(), NOW())
	`

	result, err := r.db.Exec(query, product.ProductName, product.UPC, product.SKU)
	if err != nil {
		return 0, err
	}

	insertedID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return insertedID, nil
}

func (r *inventoryRepository) InsertWmtProductDetail(productId int64, product entities.Product) error {
	query := `
		INSERT INTO wmt_product_details (product_id, gtin, wpid, available_to_sell_qty, price, createdAt, updatedAt)
		VALUES (?, ?, ?, ?, ?, NOW(), NOW())
	`

	_, err := r.db.Exec(query,
		productId,
		product.GTIN,
		product.WPID,
		product.AvailableToSellQTY,
		product.Price,
	)

	return err
}

func (r *inventoryRepository) InsertProductImage(gtin, imageUrl string) error {
	var productID int64

	query := `SELECT product_id FROM wmt_product_details WHERE gtin = ? LIMIT 1`
	err := r.db.QueryRow(query, gtin).Scan(&productID)
	if err != nil {
		return fmt.Errorf("failed to find product_id for gtin %s: %w", gtin, err)
	}

	updateQuery := `UPDATE products SET product_image = ? WHERE id = ?`
	_, err = r.db.Exec(updateQuery, imageUrl, productID)
	if err != nil {
		return fmt.Errorf("failed to update product_image: %w", err)
	}

	return nil
}

func (r *inventoryRepository) GetFirstProductByMarketplaceID(marketplaceID int) (*entities.Product, error) {
	query := `
		SELECT 
			p.seller_sku,
			p.upc,
			p.product_name,
			d.price,
			d.available_to_sell_qty,
			d.gtin
		FROM products p
		INNER JOIN wmt_product_details d ON p.id = d.product_id
		WHERE p.marketplace_id = ?
		LIMIT 1
	`

	var product entities.Product
	var upc sql.NullString
	err := r.db.QueryRow(query, marketplaceID).Scan(
		&product.SKU,
		&upc,
		&product.ProductName,
		&product.Price,
		&product.AvailableToSellQTY,
		&product.GTIN,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if upc.Valid {
		product.UPC = upc.String
	}

	return &product, nil
}

func (r *inventoryRepository) UpdateListingStatus(productID int64, listingStatusID int) error {
	query := `UPDATE products SET listing_status_id = ? WHERE id = ?`
	_, err := r.db.Exec(query, listingStatusID, productID)
	return err
}

func (r *inventoryRepository) GetProductBySKU(sku string) (*entities.Product, error) {
	query := `
		SELECT 
			p.id,
			p.seller_sku,
			p.upc,
			p.product_name,
			d.price,
			d.available_to_sell_qty,
			d.gtin,
			p.warehouse_stock,
			p.listing_status_id
		FROM products p
		INNER JOIN wmt_product_details d ON p.id = d.product_id
		WHERE p.seller_sku = ?
		LIMIT 1
	`

	var product entities.Product
	var productID int64
	var warehouseStock sql.NullInt32
	var listingStatusID sql.NullInt32
	var upc sql.NullString

	err := r.db.QueryRow(query, sku).Scan(
		&productID,
		&product.SKU,
		&upc,
		&product.ProductName,
		&product.Price,
		&product.AvailableToSellQTY,
		&product.GTIN,
		&warehouseStock,
		&listingStatusID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	product.ID = productID
	if upc.Valid {
		product.UPC = upc.String
	}
	if warehouseStock.Valid {
		product.WarehouseStock = int(warehouseStock.Int32)
	} else {
		product.WarehouseStock = 0
	}
	if listingStatusID.Valid {
		product.ListingStatusID = int(listingStatusID.Int32)
	} else {
		product.ListingStatusID = 0
	}
	return &product, nil
}

func (r *inventoryRepository) GetAllProductsByMarketplaceID(marketplaceID int) ([]*entities.Product, error) {
	query := `
		SELECT p.id, p.seller_sku, p.upc, p.product_name, p.warehouse_stock
		FROM products p
		WHERE p.marketplace_id = ? LIMIT 100
	`
	rows, err := r.db.Query(query, marketplaceID)
	if err != nil {
		return nil, fmt.Errorf("error querying products: %v", err)
	}
	defer rows.Close()

	var products []*entities.Product
	for rows.Next() {
		product := &entities.Product{}
		var warehouseStock sql.NullInt32
		var upc sql.NullString
		var productName sql.NullString
		var sellerSku sql.NullString
		err := rows.Scan(
			&product.ID,
			&sellerSku,
			&upc,
			&productName,
			&warehouseStock,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning product row: %v", err)
		}

		// Handle NULL values
		if sellerSku.Valid {
			product.SKU = sellerSku.String
		} else {
			product.SKU = ""
		}
		if upc.Valid {
			product.UPC = upc.String
		} else {
			product.UPC = ""
		}
		if productName.Valid {
			product.ProductName = productName.String
		} else {
			product.ProductName = ""
		}
		if warehouseStock.Valid {
			product.WarehouseStock = int(warehouseStock.Int32)
		} else {
			product.WarehouseStock = 0
		}
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product rows: %v", err)
	}

	return products, nil
}

func (r *inventoryRepository) GetProductByWPID(wpid string) (*entities.Product, error) {
	query := `
		SELECT 
			p.id,
			p.seller_sku,
			p.upc,
			p.product_name,
			d.price,
			d.available_to_sell_qty,
			d.gtin,
			p.warehouse_stock,
			p.product_image
		FROM products p
		INNER JOIN wmt_product_details d ON p.id = d.product_id
		WHERE d.wpid = ?
		LIMIT 1
	`

	var product entities.Product
	var productID int64
	var warehouseStock sql.NullInt32
	var productImage sql.NullString
	var upc sql.NullString
	err := r.db.QueryRow(query, wpid).Scan(
		&productID,
		&product.SKU,
		&upc,
		&product.ProductName,
		&product.Price,
		&product.AvailableToSellQTY,
		&product.GTIN,
		&warehouseStock,
		&productImage,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	product.ID = productID
	if upc.Valid {
		product.UPC = upc.String
	}
	if warehouseStock.Valid {
		product.WarehouseStock = int(warehouseStock.Int32)
	} else {
		product.WarehouseStock = 0
	}
	if productImage.Valid {
		product.ProductImage = productImage.String
	} else {
		product.ProductImage = ""
	}
	return &product, nil
}

func (r *inventoryRepository) UpdateProduct(product entities.Product) error {
	query := `
		UPDATE products 
		SET 
			product_name = ?,
			upc = ?,
			seller_sku = ?,
			updatedAt = NOW()
		WHERE id = ?
	`

	_, err := r.db.Exec(query,
		product.ProductName,
		product.UPC,
		product.SKU,
		product.ID,
	)

	return err
}

func (r *inventoryRepository) UpdateWmtProductDetail(productID int64, product entities.Product) error {
	query := `
		UPDATE wmt_product_details 
		SET 
			gtin = ?,
			available_to_sell_qty = ?,
			price = ?,
			updatedAt = NOW()
		WHERE product_id = ?
	`

	_, err := r.db.Exec(query,
		product.GTIN,
		product.AvailableToSellQTY,
		product.Price,
		productID,
	)

	return err
}
