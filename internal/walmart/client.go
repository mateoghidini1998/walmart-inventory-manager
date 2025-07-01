package walmart

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	partnerID     string
	correlationID string
	serviceName   string
	clientID      string
	clientSecret  string
	accessToken   string
	expiresAt     time.Time
	mutex         sync.Mutex
}

var (
	instance *Client
	once     sync.Once
)

func GetInstance() (*Client, error) {
	var initErr error
	once.Do(func() {
		partnerID := os.Getenv("WM_PARTNER_ID")
		clientID := os.Getenv("WALMART_CLIENT_ID")
		clientSecret := os.Getenv("WALMART_CLIENT_SECRET")

		if partnerID == "" || clientID == "" || clientSecret == "" {
			initErr = errors.New("missing required environment variables")
			return
		}

		instance = &Client{
			partnerID:     partnerID,
			correlationID: generateCorrelationID(),
			serviceName:   "Walmart Marketplace",
			clientID:      clientID,
			clientSecret:  clientSecret,
		}
	})

	if initErr != nil {
		return nil, initErr
	}

	return instance, nil
}

func NewClient() (*Client, error) {
	return GetInstance()
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

type apiResponse struct {
	ItemResponse []interface{} `json:"ItemResponse"`
}

func generateCorrelationID() string {
	return uuid.New().String()
}

func (c *Client) GetAccessToken() (string, int64, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.accessToken != "" && time.Now().Before(c.expiresAt) {
		remainingTime := int64(time.Until(c.expiresAt).Seconds())
		return c.accessToken, remainingTime, nil
	}

	token, err := c.requestToken()
	if err != nil {
		return "", 0, err
	}

	c.accessToken = token.AccessToken
	c.expiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)

	return c.accessToken, token.ExpiresIn, nil
}

func (c *Client) requestToken() (*tokenResponse, error) {
	urlEndpoint := "https://marketplace.walmartapis.com/v3/token"

	form := url.Values{}
	form.Set("grant_type", "client_credentials")

	auth := base64.StdEncoding.EncodeToString([]byte(c.clientID + ":" + c.clientSecret))

	req, err := http.NewRequest("POST", urlEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("WM_PARTNER.ID", c.partnerID)
	req.Header.Set("WM_SVC.NAME", "Walmart Marketplace")
	req.Header.Set("WM_QOS.CORRELATION_ID", generateCorrelationID())
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.New("failed to fetch access token: " + string(body))
	}

	var token tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}

	return &token, nil
}

func (c *Client) FetchWalmartItems() (map[string]map[string]interface{}, error) {
	accessToken, _, err := c.GetAccessToken()
	if err != nil {
		return nil, errors.New("failed to get access token: " + err.Error())
	}

	baseURL := "https://marketplace.walmartapis.com/v3/items?offset=0&limit=50"
	productMap := make(map[string]map[string]interface{})
	var nextCursor string = "*"

	for {
		urlEndpoint := baseURL
		if nextCursor != "" {
			urlEndpoint += "&nextCursor=" + nextCursor
		}

		req, err := http.NewRequest("GET", urlEndpoint, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Accept", "application/json")
		req.Header.Set("WM_SEC.ACCESS_TOKEN", accessToken)
		req.Header.Set("WM_QOS.CORRELATION_ID", generateCorrelationID())
		req.Header.Set("WM_SVC.NAME", "Walmart Marketplace")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			if strings.Contains(err.Error(), "context deadline exceeded") {
				return nil, fmt.Errorf("request timed out after 30 seconds: %w", err)
			}
			return nil, err
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return nil, errors.New("failed to fetch items: " + string(body))
		}

		var apiResp map[string]interface{}
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return nil, err
		}

		itemResponse, ok := apiResp["ItemResponse"]
		if !ok {
			return nil, errors.New("unexpected API response format: missing 'ItemResponse'")
		}

		items, ok := itemResponse.([]interface{})
		if !ok {
			return nil, fmt.Errorf("unexpected data type for 'ItemResponse': %T", itemResponse)
		}

		for _, rawItem := range items {
			item, ok := rawItem.(map[string]interface{})
			if !ok {
				continue
			}

			sku, _ := item["sku"].(string)
			if sku == "" {
				continue
			}

			product := map[string]interface{}{
				"productName": item["productName"],
				"sku":         sku,
				"upc":         item["upc"],
				"gtin":        item["gtin"],
				"wpid":        item["wpid"],
				"availability": item["availability"],
				"publishedStatus": item["publishedStatus"],
				"lifecycleStatus": item["lifecycleStatus"],
			}

			if price, exists := item["price"].(map[string]interface{}); exists {
				if amount, ok := price["amount"].(float64); ok {
					product["price"] = amount
				}
			}

			productMap[sku] = product
		}

		if cursor, exists := apiResp["nextCursor"].(string); exists && cursor != "" {
			nextCursor = cursor
		} else {
			break
		}
	}

	return productMap, nil
}

func FetchWalmartOrderStats(client *Client) ([]OrderStats, error) {
	accessToken, _, err := client.GetAccessToken()
	if err != nil {
		return nil, err
	}

	baseURL := "https://marketplace.walmartapis.com/v3/orders?sku=00037000949619&createdStartDate=2025-05-01&createdEndDate=2025-05-06&productInfo=true&shipNodeType=WFSFulfilled"

	type skuData struct {
		ProductName string
		OrderCount  int
		Units       int
	}
	skuMap := make(map[string]skuData)

	var nextCursor string

	for {
		urlEndpoint := baseURL
		if nextCursor != "" {
			urlEndpoint += "&nextCursor=" + url.QueryEscape(nextCursor)
		}

		req, _ := http.NewRequest("GET", urlEndpoint, nil)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("WM_SEC.ACCESS_TOKEN", accessToken)
		req.Header.Set("WM_QOS.CORRELATION_ID", generateCorrelationID())
		req.Header.Set("WM_SVC.NAME", "Walmart Marketplace")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return nil, errors.New("failed to fetch orders: " + string(body))
		}

		var data struct {
			List struct {
				Meta struct {
					NextCursor string `json:"nextCursor"`
				} `json:"meta"`
				Elements struct {
					Order []struct {
						OrderLines struct {
							OrderLine []struct {
								Item struct {
									Sku         string `json:"sku"`
									ProductName string `json:"productName"`
								} `json:"item"`
								OrderLineQuantity struct {
									Amount string `json:"amount"`
								} `json:"orderLineQuantity"`
							} `json:"orderLine"`
						} `json:"orderLines"`
					} `json:"order"`
				} `json:"elements"`
			} `json:"list"`
		}

		if err := json.Unmarshal(body, &data); err != nil {
			return nil, err
		}

		for _, order := range data.List.Elements.Order {
			for _, line := range order.OrderLines.OrderLine {
				sku := line.Item.Sku
				name := line.Item.ProductName

				amount, err := strconv.Atoi(line.OrderLineQuantity.Amount)
				if err != nil {
					amount = 0
				}

				entry := skuMap[sku]
				entry.ProductName = name
				entry.OrderCount++
				entry.Units += amount
				skuMap[sku] = entry
			}
		}

		if data.List.Meta.NextCursor == "" {
			break
		}
		nextCursor = data.List.Meta.NextCursor
	}

	var result []OrderStats
	for sku, data := range skuMap {
		result = append(result, OrderStats{
			SKU:         sku,
			ProductName: data.ProductName,
			OrderCount:  data.OrderCount,
			UnitsSold:   data.Units,
		})
	}
	return result, nil
}

func (c *Client) FetchWalmartInventory() (map[string]int, error) {
	accessToken, _, err := c.GetAccessToken()
	if err != nil {
		return nil, errors.New("failed to get access token: " + err.Error())
	}

	urlEndpoint := "https://marketplace.walmartapis.com/v3/fulfillment/inventory?limit=1000&offset=0"

	req, err := http.NewRequest("GET", urlEndpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("WM_SEC.ACCESS_TOKEN", accessToken)
	req.Header.Set("WM_QOS.CORRELATION_ID", generateCorrelationID())
	req.Header.Set("WM_SVC.NAME", "Walmart Marketplace")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "context deadline exceeded") {
			return nil, fmt.Errorf("request timed out after 30 seconds: %w", err)
		}
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch inventory: " + string(body))
	}

	fmt.Println("Raw Inventory API Response:", string(body))

	var apiResp map[string]interface{}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	payload, payloadExists := apiResp["payload"].(map[string]interface{})
	if !payloadExists {
		return nil, errors.New("missing 'payload' in API response")
	}

	inventoryList, inventoryExists := payload["inventory"].([]interface{})
	if !inventoryExists {
		return nil, errors.New("missing 'inventory' in API response")
	}

	inventoryMap := make(map[string]int)

	for _, rawInv := range inventoryList {
		inv, ok := rawInv.(map[string]interface{})
		if !ok {
			continue
		}

		sku, skuExists := inv["sku"].(string)
		if !skuExists || sku == "" {
			continue
		}

		totalQty := 0
		if shipNodes, exists := inv["shipNodes"].([]interface{}); exists {
			for _, node := range shipNodes {
				if shipNode, ok := node.(map[string]interface{}); ok {
					if qty, ok := shipNode["availToSellQty"].(float64); ok {
						totalQty += int(qty)
					}
				}
			}
		}

		inventoryMap[sku] = totalQty
	}

	fmt.Println("Processed Inventory Data:", inventoryMap)
	return inventoryMap, nil
}

func (c *Client) ItemSearch(productName, upc, gtin string) (string, error) {
	accessToken, _, err := c.GetAccessToken()
	if err != nil {
		return "", errors.New("failed to get access token: " + err.Error())
	}

	queryParams := url.Values{}
	if gtin != "" {
		queryParams.Set("gtin", gtin)
	} else if upc != "" {
		queryParams.Set("upc", upc)
	} else if productName != "" {
		queryParams.Set("query", productName)
	} else {
		return "", errors.New("no query parameter provided")
	}

	urlEndpoint := "https://marketplace.walmartapis.com/v3/items/walmart/search?" + queryParams.Encode()

	// 📌 LOG DE LA URL Y PARAMETROS
	fmt.Println("🟡 ItemSearch - Querying Walmart API with:")
	fmt.Println("🔹 productName:", productName)
	fmt.Println("🔹 upc:", upc)
	fmt.Println("🔹 gtin:", gtin)
	fmt.Println("🔹 URL:", urlEndpoint)

	req, err := http.NewRequest("GET", urlEndpoint, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("WM_SEC.ACCESS_TOKEN", accessToken)
	req.Header.Set("WM_QOS.CORRELATION_ID", generateCorrelationID())
	req.Header.Set("WM_SVC.NAME", "Walmart Marketplace")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 🔍 Leer el cuerpo para logging
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// LOG DEL CUERPO COMPLETO DE LA RESPUESTA
	fmt.Println("🔵 Walmart API Response:")
	fmt.Println(string(bodyBytes))

	// Volver a leer desde buffer para decodificar
	var result struct {
		Items []struct {
			Title  string `json:"title"`
			Images []struct {
				URL string `json:"url"`
			} `json:"images"`
		} `json:"items"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", err
	}

	// 🧠 Coincidencia de título
	for _, item := range result.Items {
		if strings.EqualFold(item.Title, productName) && len(item.Images) > 0 {
			fmt.Println("✅ Imagen encontrada para:", item.Title)
			fmt.Println("🖼️ URL de la imagen:", item.Images[0].URL) // 🔍 Aquí se imprime la URL

			return item.Images[0].URL, nil
		}
	}

	return "", errors.New("no matching image found")
}
