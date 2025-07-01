package inventory

import (
	"net/http"
	"walmart-inventory-manager/internal/service/inventory"
	"walmart-inventory-manager/platform/web/response"
)

func NewInventoryDefault(sv inventory.InventoryService) *InventoryDefault {
	return &InventoryDefault{sv: sv}
}

type InventoryDefault struct {
	sv inventory.InventoryService
}

func (h *InventoryDefault) FindAll(w http.ResponseWriter, r *http.Request) error {
	products, err := h.sv.FindAll()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Error al obtener los productos")
		return err
	}

	response.JSON(w, http.StatusOK, products)
	return nil
}
