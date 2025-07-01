package entities

type Product struct {
	SKU                string  `json:"sku"`
	UPC                string  `json:"upc"`
	ProductName        string  `json:"productName"`
	Price              float64 `json:"price"`
	AvailableToSellQTY int     `json:"availableToSellQTY"`
	GTIN               string  `json:"gtin"`
}

type OrderStats struct {
	SKU         string `json:"sku"`
	ProductName string `json:"productName"`
	OrderCount  int    `json:"orderCount"`
	UnitsSold   int    `json:"unitsSold"`
}
