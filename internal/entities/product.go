package entities

type Product struct {
	ID                 int64   `json:"id"`
	SKU                string  `json:"sku"`
	UPC                string  `json:"upc"`
	ProductName        string  `json:"productName"`
	Price              float64 `json:"price"`
	AvailableToSellQTY int     `json:"availableToSellQTY"`
	GTIN               string  `json:"gtin"`
	WarehouseStock     int     `json:"warehouseStock"`
	WPID               string  `json:"wpid"`
	ProductImage       string  `json:"productImage"`
	Availability       string  `json:"availability"`
	PublishedStatus    string  `json:"publishedStatus"`
	LifecycleStatus    string  `json:"lifecycleStatus"`
	ListingStatusID    int     `json:"listing_status_id"`
}
