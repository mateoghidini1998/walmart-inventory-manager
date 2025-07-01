package walmart

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"walmart-inventory-manager/internal/entities"
	"walmart-inventory-manager/internal/repositories/inventory"
)

type OrderStats struct {
	SKU         string `json:"sku"`
	ProductName string `json:"productName"`
	OrderCount  int    `json:"orderCount"`
	UnitsSold   int    `json:"unitsSold"`
}

func OrdersCronjob(client *Client, repo inventory.InventoryRepository) {
	go func() {
		for {
			location, err := time.LoadLocation("America/Argentina/Buenos_Aires")
			if err != nil {
				log.Println("[OrdersCronjob] Error loading timezone:", err)
				location = time.UTC
			}

			now := time.Now().In(location)
			nextRun := time.Date(now.Year(), now.Month(), now.Day(), 23, 40, 0, 0, location)
			if now.After(nextRun) {
				nextRun = nextRun.Add(24 * time.Hour)
			}

			sleepDuration := time.Until(nextRun)
			log.Printf("[OrdersCronjob] Now: %s | Sleeping until: %s (duration: %s)\n", now.Format(time.RFC1123), nextRun.Format(time.RFC1123), sleepDuration)

			time.Sleep(sleepDuration)

			start := time.Now()
			log.Println("[OrdersCronjob] Starting Walmart orders fetch...")

			stats, err := FetchWalmartOrderStats(client)
			if err != nil {
				log.Printf("[OrdersCronjob] Error fetching Walmart orders: %v\n", err)
				continue
			}

			log.Printf("[OrdersCronjob] Successfully fetched %d SKUs\n", len(stats))

			jsonBytes, err := json.MarshalIndent(stats, "", "  ")
			if err != nil {
				log.Printf("[OrdersCronjob] Error marshaling stats to JSON: %v\n", err)
				continue
			}

			log.Println("[OrdersCronjob] JSON output:")
			fmt.Println(string(jsonBytes))

			log.Printf("[OrdersCronjob] Finished inserting Walmart Orders. Duration: %s\n", time.Since(start))
		}
	}()
}

func StartCronJob(client *Client, repo inventory.InventoryRepository) {
	go func() {
		for {
			now := time.Now()
			location, err := time.LoadLocation("America/Argentina/Buenos_Aires")
			if err != nil {
				log.Println("Error loading timezone:", err)
				location = time.UTC
			}

			nextRun := time.Date(now.Year(), now.Month(), now.Day(), 15, 25, 0, 0, location)

			if now.After(nextRun) {
				nextRun = nextRun.Add(24 * time.Hour)
			}

			durationUntilNextRun := time.Until(nextRun)
			log.Printf("Next Walmart Items Fetch Job at: %s\n", nextRun)

			time.Sleep(durationUntilNextRun)

			log.Println("Running Walmart Items Fetch Job...")

			productsMap, err := fetchWalmartItemsWithRetry(client, 3)
			if err != nil {
				log.Printf("Error fetching Walmart items after retries: %v\n", err)
				continue
			}

			inventoryMap, err := fetchWalmartInventoryWithRetry(client, 3)
			if err != nil {
				log.Printf("Error fetching Walmart inventory after retries: %v\n", err)
				continue
			}

			// Create inventory stats JSON
			inventoryStats := make(map[string]int)
			for sku, qty := range inventoryMap {
				inventoryStats[sku] = qty
			}

			// Marshal inventory stats
			inventoryJSON, err := json.MarshalIndent(inventoryStats, "", "  ")
			if err != nil {
				log.Printf("Error marshaling inventory stats to JSON: %v\n", err)
				continue
			}

			// Save to file in root directory
			err = os.WriteFile("inventory_stats.json", inventoryJSON, 0644)
			if err != nil {
				log.Printf("Error writing inventory stats to file: %v\n", err)
				continue
			}

			log.Println("Inventory stats saved to inventory_stats.json")

			dbProducts, err := repo.FindAll()
			if err != nil {
				log.Printf("Error fetching products from DB: %v\n", err)
				continue
			}

			dbProductSKUs := make(map[string]entities.Product)
			for _, p := range dbProducts {
				dbProductSKUs[p.SKU] = p
			}

			successCount := 0
			errorCount := 0
			updateCount := 0
			insertCount := 0

			for sku, productData := range productsMap {
				// Get available quantity from inventory data
				availableQty := 0
				if availToSellQty, exists := inventoryMap[sku]; exists {
					availableQty = availToSellQty
				}

				lifecycleStatus := getStringValue(productData, "lifecycleStatus")
				availability := getStringValue(productData, "availability")
				publishedStatus := getStringValue(productData, "publishedStatus")

				var listingStatusID int
				if lifecycleStatus == "ACTIVE" && availability == "In_stock" && publishedStatus == "PUBLISHED" {
					listingStatusID = 1
				} else if lifecycleStatus == "ACTIVE" && availability == "Out_of_stock" && publishedStatus == "PUBLISHED" {
					listingStatusID = 2
				} else if lifecycleStatus == "ARCHIVED" || (publishedStatus == "UNPUBLISHED" && lifecycleStatus == "ACTIVE") || (publishedStatus == "SYSTEM_PROBLEM" && lifecycleStatus == "ACTIVE") {
					listingStatusID = 3
				}

				// Create product structure with combined data
				product := entities.Product{
					SKU:                sku,
					UPC:                getStringValue(productData, "upc"),
					ProductName:        getStringValue(productData, "productName"),
					Price:              getFloatValue(productData, "price"),
					AvailableToSellQTY: availableQty,
					GTIN:               getStringValue(productData, "gtin"),
					WPID:               getStringValue(productData, "wpid"),
					Availability:       availability,
					PublishedStatus:    getStringValue(productData, "publishedStatus"),
					LifecycleStatus:    lifecycleStatus,
					ListingStatusID:    listingStatusID,
				}

				// Check if product exists by SKU (seller_sku in products table)
				existingProduct, err := repo.GetProductBySKU(sku)
				if err != nil {
					log.Printf("Error checking existing product for SKU %s: %v\n", sku, err)
					errorCount++
					continue
				}

				if existingProduct != nil {
					// Product exists, update it
					product.ID = existingProduct.ID

					// Update product details (product_name, upc, seller_sku)
					err = repo.UpdateProduct(product)
					if err != nil {
						log.Printf("Error updating product for SKU %s: %v\n", sku, err)
						errorCount++
						continue
					}

					// Update Walmart product details (gtin, available_to_sell_qty, price)
					err = repo.UpdateWmtProductDetail(product.ID, product)
					if err != nil {
						log.Printf("Error updating wmt_product_detail for SKU %s: %v\n", sku, err)
						errorCount++
						continue
					}

					// Update listing status if it has changed
					if existingProduct.ListingStatusID != product.ListingStatusID {
						err = repo.UpdateListingStatus(product.ID, product.ListingStatusID)
						if err != nil {
							log.Printf("Error updating listing status for SKU %s: %v\n", sku, err)
						}
					}

					// Search and update product image if it doesn't exist
					if existingProduct.ProductImage == "" {
						imageURL, err := client.ItemSearch(product.ProductName, product.UPC, product.GTIN)
						if err != nil {
							log.Printf("No image found for %s: %v\n", product.ProductName, err)
						} else {
							err = repo.InsertProductImage(product.GTIN, imageURL)
							if err != nil {
								log.Printf("Error updating product image for WPID %s: %v\n", product.WPID, err)
							} else {
								log.Printf("Updated product image for WPID %s\n", product.WPID)
							}
						}
					}

					updateCount++
					successCount++
					log.Printf("Updated product SKU %s - Available Qty: %d, Price: %.2f\n", sku, availableQty, product.Price)
				} else {
					// Product doesn't exist, insert new one
					productID, err := repo.InsertProduct(product)
					if err != nil {
						log.Printf("Error inserting product for SKU %s: %v\n", sku, err)
						errorCount++
						continue
					}

					err = repo.InsertWmtProductDetail(productID, product)
					if err != nil {
						log.Printf("Error inserting wmt_product_detail for SKU %s: %v\n", sku, err)
						errorCount++
						continue
					}

					// Set listing status for new product
					err = repo.UpdateListingStatus(productID, product.ListingStatusID)
					if err != nil {
						log.Printf("Error setting listing status for new SKU %s: %v\n", sku, err)
					}

					// Search and insert product image
					imageURL, err := client.ItemSearch(product.ProductName, product.UPC, product.GTIN)
					if err != nil {
						log.Printf("No image found for %s: %v\n", product.ProductName, err)
					} else {
						err = repo.InsertProductImage(product.GTIN, imageURL)
						if err != nil {
							log.Printf("Error inserting product image for SKU %s: %v\n", sku, err)
						} else {
							log.Printf("Inserted product image for SKU %s\n", sku)
						}
					}
					insertCount++
					successCount++
					log.Printf("Inserted new product SKU %s - Available Qty: %d, Price: %.2f\n", sku, availableQty, product.Price)
				}
				// Remove SKU from the map of DB products; remaining ones are not in the API response
				delete(dbProductSKUs, sku)
			}

			// For all the products retrieved from the DB that are not in the response of the walmart API we should set the listing_status_id of 5.
			for _, p := range dbProductSKUs {
				log.Printf("Product %s not in Walmart response, setting listing_status_id to 5", p.SKU)
				err := repo.UpdateListingStatus(p.ID, 5)
				if err != nil {
					log.Printf("Error updating listing status to 5 for SKU %s: %v\n", p.SKU, err)
					errorCount++
				}
			}

			log.Printf("Finished processing Walmart Inventory. Success: %d, Updates: %d, Inserts: %d, Errors: %d\n",
				successCount, updateCount, insertCount, errorCount)
		}
	}()
}

// fetchWalmartItemsWithRetry attempts to fetch Walmart items with retry logic
func fetchWalmartItemsWithRetry(client *Client, maxRetries int) (map[string]map[string]interface{}, error) {
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		productsMap, err := client.FetchWalmartItems()
		if err == nil {
			return productsMap, nil
		}
		lastErr = err
		log.Printf("Attempt %d/%d failed to fetch Walmart items: %v\n", attempt, maxRetries, err)
		if attempt < maxRetries {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			log.Printf("Retrying in %v...\n", backoff)
			time.Sleep(backoff)
		}
	}
	return nil, fmt.Errorf("failed after %d attempts: %v", maxRetries, lastErr)
}

// fetchWalmartInventoryWithRetry attempts to fetch Walmart inventory with retry logic
func fetchWalmartInventoryWithRetry(client *Client, maxRetries int) (map[string]int, error) {
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		inventoryMap, err := client.FetchWalmartInventory()
		if err == nil {
			return inventoryMap, nil
		}
		lastErr = err
		log.Printf("Attempt %d/%d failed to fetch Walmart inventory: %v\n", attempt, maxRetries, err)
		if attempt < maxRetries {
			// Exponential backoff: 2^attempt seconds
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			log.Printf("Retrying in %v...\n", backoff)
			time.Sleep(backoff)
		}
	}
	return nil, fmt.Errorf("failed after %d attempts: %v", maxRetries, lastErr)
}

func getStringValue(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getFloatValue(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

func getIntValue(m map[string]interface{}, key string) int {
	if v, ok := m[key].(int); ok {
		return v
	}
	return 0
}
