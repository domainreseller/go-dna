package dna

import (
	"context"
	"iter"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// AccountService is the reseller account: balances and the balance ledger.
// CLI: dna account ...
type AccountService service

// Account is the reseller account and its balances.
type Account struct {
	ResellerID        string  `json:"resellerId"`
	ResellerName      string  `json:"resellerName"`
	ResellerGroupName string  `json:"resellerGroupName"`
	TRYBalance        float64 `json:"tryBalance"`
	USDBalance        float64 `json:"usdBalance"`
}

// Balance returns the balance in a currency, "USD" or "TRY" (any case).
// ok is false for other currencies: the account holds only these two.
func (a *Account) Balance(currency string) (amount float64, ok bool) {
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "USD":
		return a.USDBalance, true
	case "TRY", "TL":
		return a.TRYBalance, true
	}
	return 0, false
}

// Transaction is one entry of the balance ledger.
type Transaction struct {
	TraceID        string            `json:"traceId"`
	Type           TransactionType   `json:"type"`
	Status         TransactionStatus `json:"status"`
	ChargeType     ChargeType        `json:"chargeType"`
	Amount         float64           `json:"amount"`
	AmountCurrency string            `json:"amountCurrency"`
	ExchangeAmount float64           `json:"exchangeAmount"`
	// ExchangeCurrency is numeric in the API.
	ExchangeCurrency float64     `json:"exchangeCurrency"`
	ProductType      ProductType `json:"productType"`
	ProductID        string      `json:"productId"`
	Description      string      `json:"description"`
	// Period is the number of years bought, for domain transactions.
	Period       int     `json:"period"`
	CreationTime Time    `json:"creationTime"`
	ResellerName string  `json:"resellerName"`
	Balance      float64 `json:"balance"`    // balance after the transaction
	OldBalance   float64 `json:"oldBalance"` // balance before it
}

// Change returns how the transaction moved the balance: positive for
// credits, negative for charges. It is reliable even when Amount is 0.
func (t Transaction) Change() float64 {
	return math.Round((t.Balance-t.OldBalance)*100) / 100
}

// AccountGetRequest reads the account. CLI: dna account get
type AccountGetRequest struct{ BaseRequest }

func (AccountGetRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/deposit/accounts/me", Safety: Read}
}

// AccountTransactionsRequest lists balance transactions.
// CLI: dna account transactions
type AccountTransactionsRequest struct {
	BaseRequest
	ListOptions
	// TransactionTypes keeps only these kinds (Register, Renew, ...).
	TransactionTypes []TransactionType
	// ProductTypes keeps only these product kinds.
	ProductTypes []ProductType
	// ChargeType keeps only credits (ChargeAdd) or debits (ChargeSubtract).
	ChargeType *ChargeType
	// Currencies keeps only these currencies, e.g. "USD".
	Currencies []string
	// From and To limit the creation time.
	From, To time.Time
	// TraceID finds one transaction.
	TraceID string
	// Description and FilterText search the text.
	Description, FilterText string
	// Amount keeps only transactions of this amount.
	Amount float64
}

func (AccountTransactionsRequest) Endpoint() Endpoint {
	return Endpoint{Method: http.MethodGet, Path: "/api/v1/deposit/transactions", Safety: Read}
}

func (r AccountTransactionsRequest) Query() url.Values {
	p := newParams().
		each("TransactionTypes", intStrings(r.TransactionTypes)).
		each("ProductTypes", intStrings(r.ProductTypes)).
		each("AccountCurrencyTypes", r.Currencies).
		date("StartDate", r.From).
		date("EndDate", r.To).
		str("TraceId", r.TraceID).
		str("Description", r.Description).
		str("FilterText", r.FilterText).
		float("Amount", r.Amount).
		page(r.ListOptions)
	if r.ChargeType != nil {
		p.Set("ChargeType", strconv.Itoa(int(*r.ChargeType)))
	}
	return p.Values
}

func (r AccountTransactionsRequest) Validate() error {
	var v validator
	r.ListOptions.validate(&v)
	v.check(r.From.IsZero() || r.To.IsZero() || !r.To.Before(r.From), "to", "must not be before from")
	return v.err()
}

// Get returns the account and its TRY and USD balances. It is also the
// cheapest way to check that the credentials work.
func (s *AccountService) Get(ctx context.Context) (*Account, error) {
	return call[Account](ctx, s.client, AccountGetRequest{})
}

// Transactions returns one page of the balance ledger.
func (s *AccountService) Transactions(ctx context.Context, req AccountTransactionsRequest) (*Page[Transaction], error) {
	return call[Page[Transaction]](ctx, s.client, req)
}

// AllTransactions iterates over every matching transaction, fetching pages
// as needed:
//
//	for tx, err := range client.Account.AllTransactions(ctx, dna.AccountTransactionsRequest{From: monthStart}) {
//		if err != nil {
//			return err
//		}
//		fmt.Println(tx.CreationTime, tx.Type, tx.Change())
//	}
func (s *AccountService) AllTransactions(ctx context.Context, req AccountTransactionsRequest) iter.Seq2[Transaction, error] {
	return paginate(req.ListOptions, func(o ListOptions) (*Page[Transaction], error) {
		req.ListOptions = o
		return s.Transactions(ctx, req)
	})
}
