package dependencies

import (
	"log"
	"walmart-inventory-manager/internal/config"
	"walmart-inventory-manager/internal/db"
	"walmart-inventory-manager/internal/handler/inventory"
	"walmart-inventory-manager/internal/handler/walmart"
	inventoryRepository "walmart-inventory-manager/internal/repositories/inventory"
	inventoryService "walmart-inventory-manager/internal/service/inventory"
	walmartClient "walmart-inventory-manager/internal/walmart"
)

type HandlerContainer struct {
	InventoryHandler    *inventory.InventoryDefault
	InventoryRepository inventoryRepository.InventoryRepository
	WalmartHandler      *walmart.TokenHandler
	WalmartClient       *walmartClient.Client
}

func NewDependencies() (*HandlerContainer, error) {

	cfg := config.NewConfig()

	db, err := db.ConnectDB(cfg)
	if err != nil {
		return nil, err
	}

	inventoryRepo := inventoryRepository.NewInventoryRepository(db)

	inventoryUsecase := inventoryService.NewInventoryDefault(inventoryRepo)

	inventoryHandler := inventory.NewInventoryDefault(inventoryUsecase)

	walmart_client, err := walmartClient.NewClient()
	if err != nil {
		return nil, err
	}

	walmartHandler := walmart.NewTokenHandler(walmart_client)

	return &HandlerContainer{
		InventoryHandler:    inventoryHandler,
		InventoryRepository: inventoryRepo, 
		WalmartHandler:      walmartHandler,
		WalmartClient:       walmart_client,
	}, nil
}

func Start() *HandlerContainer {
	deps, err := NewDependencies()
	if err != nil {
		log.Fatalf("Error al iniciar las dependencias: %v", err)
	}
	return deps
}
