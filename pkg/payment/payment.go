// Package payment implements payment processing and transaction management.
//
// Features include:
// - Payment gateway integration (pluggable backends)
// - Multi-currency support
// - Wallet/escrow management
// - Transaction logging
// - Payment status tracking
// - Dispute handling
package payment

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

// Currency represents supported payment currencies.
type Currency string

const (
	CurrencyUSD Currency = "USD"
	CurrencyEUR Currency = "EUR"
	CurrencyGBP Currency = "GBP"
	CurrencyJPY Currency = "JPY"
	CurrencyXXX Currency = "XXX" // Test/unknown
)

// PaymentStatus represents transaction state.
type PaymentStatus string

const (
	StatusPending      PaymentStatus = "pending"
	StatusProcessing   PaymentStatus = "processing"
	StatusCompleted    PaymentStatus = "completed"
	StatusFailed       PaymentStatus = "failed"
	StatusRefunding    PaymentStatus = "refunding"
	StatusRefunded     PaymentStatus = "refunded"
	StatusDisputed     PaymentStatus = "disputed"
	StatusCancelled    PaymentStatus = "cancelled"
)

// PaymentMethod defines the payment source type.
type PaymentMethod string

const (
	MethodCreditCard PaymentMethod = "credit_card"
	MethodBankTransfer PaymentMethod = "bank_transfer"
	MethodWallet     PaymentMethod = "wallet"
	MethodEscrow     PaymentMethod = "escrow"
	MethodStablecoin PaymentMethod = "stablecoin"
)

// Gateway is the interface for payment processors.
type Gateway interface {
	Name() string
	ProcessPayment(ctx context.Context, req PaymentRequest) (*PaymentResult, error)
	RefundPayment(ctx context.Context, transactionID string, amount float64) (*RefundResult, error)
	GetStatus(ctx context.Context, transactionID string) (PaymentStatus, error)
}

// PaymentRequest specifies payment parameters.
type PaymentRequest struct {
	TenantID       string
	SettlementID   string
	Amount         float64
	Currency       Currency
	Method         PaymentMethod
	Description    string
	IdempotencyKey string // prevent double-processing
	Metadata       map[string]string
}

// PaymentResult is returned from gateway processing.
type PaymentResult struct {
	TransactionID   string
	Status          PaymentStatus
	Amount          float64
	Currency        Currency
	Timestamp       time.Time
	ConfirmationID  string
	GatewayResponse map[string]interface{}
}

// RefundResult is returned from gateway refund.
type RefundResult struct {
	RefundID      string
	OriginalTxnID string
	Status        PaymentStatus
	Amount        float64
	Currency      Currency
	Timestamp     time.Time
	Reason        string
}

// Transaction represents a payment transaction.
type Transaction struct {
	ID               string
	TenantID         string
	SettlementID     string
	Amount           float64
	Currency         Currency
	Method           PaymentMethod
	Status           PaymentStatus
	GatewayName      string
	GatewayTxnID     string
	ConfirmationID   string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CompletedAt      *time.Time
	FailureReason    string
	Description      string
	IdempotencyKey   string
	Hash             string
}

// Wallet represents operator or tenant account.
type Wallet struct {
	ID              string
	OwnerID         string // tenant or operator ID
	Balance         float64
	Currency        Currency
	Status          string // active, frozen, closed
	HoldAmount      float64 // amount on hold for pending settlements
	AvailableAmount float64 // balance - hold
	LastTopUpAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Escrow represents a payment hold for dispute resolution.
type Escrow struct {
	ID             string
	TransactionID  string
	SourceOwnerID  string // owner ID where funds were held from
	HeldAmount     float64
	Currency       Currency
	HeldUntil      time.Time
	Reason         string
	Status         string // held, released, disputed
	ReleasedAt     *time.Time
}

// Processor manages payment processing and transactions.
type Processor struct {
	mu                  sync.RWMutex
	gateways            map[string]Gateway
	transactions        map[string]*Transaction
	wallets             map[string]*Wallet
	escrows             map[string]*Escrow
	idempotencyCache    map[string]*PaymentResult
	maxRetries          int
	retryIntervalSecs   int64
	escrowRetentionDays int
	defaultCurrency     Currency
	currencyRates       map[Currency]float64
}

// NewProcessor creates a new payment processor.
func NewProcessor() *Processor {
	return &Processor{
		gateways:            make(map[string]Gateway),
		transactions:        make(map[string]*Transaction),
		wallets:             make(map[string]*Wallet),
		escrows:             make(map[string]*Escrow),
		idempotencyCache:    make(map[string]*PaymentResult),
		maxRetries:          3,
		retryIntervalSecs:   5,
		escrowRetentionDays: 90,
		defaultCurrency:     CurrencyUSD,
		currencyRates:       make(map[Currency]float64),
	}
}

// RegisterGateway adds a payment gateway.
func (p *Processor) RegisterGateway(gateway Gateway) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.gateways[gateway.Name()] = gateway
}

// ProcessPayment processes a payment transaction.
func (p *Processor) ProcessPayment(ctx context.Context, req PaymentRequest) (*Transaction, error) {
	p.mu.Lock()

	// Check idempotency
	if req.IdempotencyKey != "" {
		if cached, ok := p.idempotencyCache[req.IdempotencyKey]; ok {
			p.mu.Unlock()
			return &Transaction{
				ID:            cached.TransactionID,
				TenantID:      req.TenantID,
				SettlementID:  req.SettlementID,
				Amount:        cached.Amount,
				Currency:      cached.Currency,
				Status:        cached.Status,
				ConfirmationID: cached.ConfirmationID,
			}, nil
		}
	}

	gatewayName := "default"
	gateway, ok := p.gateways[gatewayName]
	if !ok {
		if len(p.gateways) == 0 {
			p.mu.Unlock()
			return nil, fmt.Errorf("no payment gateways configured")
		}
		// Use first available gateway
		for name, g := range p.gateways {
			gatewayName = name
			gateway = g
			break
		}
	}

	p.mu.Unlock()

	// Process through gateway
	result, err := gateway.ProcessPayment(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("payment processing failed: %w", err)
	}

	// Record transaction
	txn := &Transaction{
		ID:             generateTransactionID(),
		TenantID:       req.TenantID,
		SettlementID:   req.SettlementID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Method:         req.Method,
		Status:         result.Status,
		GatewayName:    gatewayName,
		GatewayTxnID:   result.TransactionID,
		ConfirmationID: result.ConfirmationID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Description:    req.Description,
		IdempotencyKey: req.IdempotencyKey,
	}

	if result.Status == StatusCompleted {
		now := time.Now()
		txn.CompletedAt = &now
	} else if result.Status == StatusFailed {
		txn.FailureReason = "gateway processing failed"
	}

	txn.Hash = p.hashTransaction(txn)

	p.mu.Lock()
	p.transactions[txn.ID] = txn
	if req.IdempotencyKey != "" {
		p.idempotencyCache[req.IdempotencyKey] = result
	}
	p.mu.Unlock()

	return txn, nil
}

// CreateOrGetWallet creates or retrieves an owner's wallet.
func (p *Processor) CreateOrGetWallet(ctx context.Context, ownerID string, currency Currency) (*Wallet, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := fmt.Sprintf("%s:%s", ownerID, currency)
	if wallet, ok := p.wallets[key]; ok {
		return wallet, nil
	}

	wallet := &Wallet{
		ID:              generateWalletID(),
		OwnerID:         ownerID,
		Balance:         0,
		Currency:        currency,
		Status:          "active",
		AvailableAmount: 0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	p.wallets[key] = wallet
	return wallet, nil
}

// TopUpWallet adds funds to a wallet.
func (p *Processor) TopUpWallet(ctx context.Context, ownerID string, amount float64, currency Currency) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if amount <= 0 {
		return fmt.Errorf("invalid topup amount: %.2f", amount)
	}

	key := fmt.Sprintf("%s:%s", ownerID, currency)
	wallet, ok := p.wallets[key]
	if !ok {
		return fmt.Errorf("wallet not found: %s", ownerID)
	}

	wallet.Balance += amount
	wallet.AvailableAmount = wallet.Balance - wallet.HoldAmount
	now := time.Now()
	wallet.LastTopUpAt = &now
	wallet.UpdatedAt = now

	return nil
}

// GetWallet retrieves wallet details.
func (p *Processor) GetWallet(ctx context.Context, ownerID string, currency Currency) (*Wallet, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", ownerID, currency)
	wallet, ok := p.wallets[key]
	if !ok {
		return nil, fmt.Errorf("wallet not found: %s", ownerID)
	}

	return wallet, nil
}

// HoldFunds places a hold on wallet funds.
func (p *Processor) HoldFunds(ctx context.Context, ownerID string, amount float64, currency Currency) (*Escrow, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := fmt.Sprintf("%s:%s", ownerID, currency)
	wallet, ok := p.wallets[key]
	if !ok {
		return nil, fmt.Errorf("wallet not found: %s", ownerID)
	}

	if wallet.AvailableAmount < amount {
		return nil, fmt.Errorf("insufficient funds: available %.2f, need %.2f", wallet.AvailableAmount, amount)
	}

	escrow := &Escrow{
		ID:            generateEscrowID(),
		SourceOwnerID: ownerID,
		HeldAmount:    amount,
		Currency:      currency,
		HeldUntil:     time.Now().AddDate(0, 0, p.escrowRetentionDays),
		Status:        "held",
	}

	wallet.HoldAmount += amount
	wallet.AvailableAmount = wallet.Balance - wallet.HoldAmount
	wallet.UpdatedAt = time.Now()

	p.escrows[escrow.ID] = escrow

	return escrow, nil
}

// ReleaseFunds releases held escrow funds.
func (p *Processor) ReleaseFunds(ctx context.Context, escrowID string, toWalletOwnerID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	escrow, ok := p.escrows[escrowID]
	if !ok {
		return fmt.Errorf("escrow not found: %s", escrowID)
	}

	if escrow.Status != "held" {
		return fmt.Errorf("escrow not held: %s", escrow.Status)
	}

	// Reduce hold on source wallet
	sourceKey := fmt.Sprintf("%s:%s", escrow.SourceOwnerID, escrow.Currency)
	if sourceWallet, ok := p.wallets[sourceKey]; ok {
		sourceWallet.HoldAmount -= escrow.HeldAmount
		if sourceWallet.HoldAmount < 0 {
			sourceWallet.HoldAmount = 0
		}
		sourceWallet.AvailableAmount = sourceWallet.Balance - sourceWallet.HoldAmount
		sourceWallet.UpdatedAt = time.Now()
	}

	// Transfer to target wallet
	key := fmt.Sprintf("%s:%s", toWalletOwnerID, escrow.Currency)
	wallet, ok := p.wallets[key]
	if !ok {
		return fmt.Errorf("target wallet not found: %s", toWalletOwnerID)
	}

	wallet.Balance += escrow.HeldAmount
	wallet.AvailableAmount = wallet.Balance - wallet.HoldAmount
	wallet.UpdatedAt = time.Now()

	now := time.Now()
	escrow.Status = "released"
	escrow.ReleasedAt = &now

	return nil
}

// GetTransaction retrieves transaction details.
func (p *Processor) GetTransaction(ctx context.Context, txnID string) (*Transaction, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	txn, ok := p.transactions[txnID]
	if !ok {
		return nil, fmt.Errorf("transaction not found: %s", txnID)
	}

	return txn, nil
}

// ListTransactions returns transactions for a tenant.
func (p *Processor) ListTransactions(ctx context.Context, tenantID string, statuses ...PaymentStatus) []*Transaction {
	p.mu.RLock()
	defer p.mu.RUnlock()

	statusMap := make(map[PaymentStatus]bool)
	for _, s := range statuses {
		statusMap[s] = true
	}

	var result []*Transaction
	for _, txn := range p.transactions {
		if txn.TenantID == tenantID {
			if len(statusMap) == 0 || statusMap[txn.Status] {
				result = append(result, txn)
			}
		}
	}

	return result
}

// SetExchangeRate updates currency conversion rates.
func (p *Processor) SetExchangeRate(from, to Currency, rate float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if rate > 0 {
		key := fmt.Sprintf("%s-%s", from, to)
		p.currencyRates[Currency(key)] = rate
	}
}

// ConvertCurrency converts amount from one currency to another.
func (p *Processor) ConvertCurrency(from, to Currency, amount float64) (float64, error) {
	if from == to {
		return amount, nil
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	key := fmt.Sprintf("%s-%s", from, to)
	rate, ok := p.currencyRates[Currency(key)]
	if !ok {
		return 0, fmt.Errorf("no conversion rate for %s-%s", from, to)
	}

	return amount * rate, nil
}

// Helper functions

func (p *Processor) hashTransaction(txn *Transaction) string {
	data := fmt.Sprintf(
		"%s:%s:%.2f:%s:%s:%d",
		txn.SettlementID,
		txn.GatewayTxnID,
		txn.Amount,
		txn.Currency,
		txn.Status,
		txn.CreatedAt.Unix(),
	)

	h := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", h)[:16]
}

func generateTransactionID() string {
	h := sha256.Sum256([]byte(time.Now().String()))
	return fmt.Sprintf("txn-%x", h)[:18]
}

func generateWalletID() string {
	h := sha256.Sum256([]byte(time.Now().String()))
	return fmt.Sprintf("wallet-%x", h)[:22]
}

func generateEscrowID() string {
	h := sha256.Sum256([]byte(time.Now().String()))
	return fmt.Sprintf("escrow-%x", h)[:21]
}
