package application

import (
	"fmt"
	"net/http"
	"os"
	"walmart-inventory-manager/internal/infrastructure/dependencies"
	"walmart-inventory-manager/internal/walmart"
	"walmart-inventory-manager/platform/web"
)

type applicationDefault struct {
	r *web.Router
	Application
	deps *dependencies.HandlerContainer
}

func NewApplication() (Application, error) {
	deps := dependencies.Start()

	return &applicationDefault{
		r:    web.NewRouter(),
		deps: deps,
	}, nil
}

func (a *applicationDefault) Run() (err error) {
	fmt.Println("Server running on http://localhost:8081")
	return http.ListenAndServe(":8081", a.r)
}

func (a *applicationDefault) SetUp() (err error) {
	serverAddress := os.Getenv("SERVER_ADDRESS")
	if serverAddress == "" {
		serverAddress = "localhost:8081"
	}

	a.setUpRoutes()
	walmart.StartCronJob(a.deps.WalmartClient, a.deps.InventoryRepository)
	walmart.OrdersCronjob(a.deps.WalmartClient, a.deps.InventoryRepository)

	return nil
}

func (a *applicationDefault) TearDown() (err error) {
	return nil
}

func (a *applicationDefault) setUpRoutes() {
	a.r.Route("/api/v1/inventory", func(rg *web.RouterGroup) {
		rg.Handle("GET", "", a.deps.InventoryHandler.FindAll)
	})

	a.r.Route("/api/v1/token", func(rg *web.RouterGroup) {
		rg.Handle("GET", "", a.deps.WalmartHandler.GetToken)
	})
}
