package dna

import (
	"context"
	"net/http"
	"net/url"
)

// OrderService reads orders. CLI: dna order ...
type OrderService service

// Order is an order and its lines.
type Order struct {
	OrderCode      string         `json:"orderCode"`
	Status         OrderStatus    `json:"status"`
	Amount         float64        `json:"amount"`
	AmountCurrency string         `json:"amountCurrency"`
	Products       []OrderProduct `json:"products"`
}

// OrderProduct is one line of an order.
type OrderProduct struct {
	ProductCode    string             `json:"productCode"`
	ProductType    ProductType        `json:"productType"`
	OrderType      OrderType          `json:"orderType"`
	Status         OrderProductStatus `json:"status"`
	Price          float64            `json:"price"`
	PriceCurrency  string             `json:"priceCurrency"`
	Name           string             `json:"name"`
	Description    string             `json:"description"`
	AdditionalData string             `json:"additionalData"`
	Period         int                `json:"period"`
}

// OrderGetRequest reads an order. CLI: dna order get
type OrderGetRequest struct {
	BaseRequest
	OrderCode string
}

func (r OrderGetRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/order/" + url.PathEscape(trim(r.OrderCode)), Safety: Read}
}

func (r OrderGetRequest) Validate() error {
	var v validator
	v.required("orderCode", r.OrderCode)
	return v.err()
}

// Get returns an order and its lines.
func (s *OrderService) Get(ctx context.Context, orderCode string) (*Order, error) {
	return call[Order](ctx, s.client, OrderGetRequest{OrderCode: orderCode})
}
