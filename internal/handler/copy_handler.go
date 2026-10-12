package handler

type CopyRequest struct {
	InventoryNo string `json:"inventory_no"`
	Status      string `json:"status"`
}
