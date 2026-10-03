package payment

import (
	"context"
	"testing"
	"time"
)

// MockGateway implements Gateway for testing.
type MockGateway struct {
	successRate float64
	callCount   int
}

func NewMockGateway(successRate float64) *MockGateway {
	return &MockGateway{successRate: successRate}
}

func (m *MockGateway) Name() string {
	return "mock"
}

func (m *MockGateway) ProcessPayment(ctx context.Context, req PaymentRequest) (*PaymentResult, error) {
	m.callCount++

	status := StatusCompleted
	if float64(m.callCount%100) < (100 * (1 - m.successRate)) {
		status = StatusFailed
	}

	return &PaymentResult{
		TransactionID:  "mock-txn-" + time.Now().String(),
		Status:         status,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Timestamp:      time.Now(),
		ConfirmationID: "mock-conf-" + time.Now().String(),
	}, nil
}

func (m *MockGateway) RefundPayment(ctx context.Context, transactionID string, amount float64) (*RefundResult, error) {
	return &RefundResult{
		RefundID:      "refund-" + time.Now().String(),
		OriginalTxnID: transactionID,
		Status:        StatusRefunded,
		Amount:        amount,
		Timestamp:     time.Now(),
	}, nil
}

func (m *MockGateway) GetStatus(ctx context.Context, transactionID string) (PaymentStatus, error) {
	return StatusCompleted, nil
}

func TestProcessPaymentWithGateway(t *testing.T) {
	processor := NewProcessor()
	gateway := NewMockGateway(0.95)
	processor.RegisterGateway(gateway)

	ctx := context.Background()

	req := PaymentRequest{
		TenantID:     "tenant-1",
		SettlementID: "settle-1",
		Amount:       100.0,
		Currency:     CurrencyUSD,
		Method:       MethodCreditCard,
		Description:  "Test payment",
	}

	txn, err := processor.ProcessPayment(ctx, req)
	if err != nil {
		t.Fatalf("ProcessPayment failed: %v", err)
	}

	if txn.ID == "" {
		t.Error("Transaction ID is empty")
	}

	if txn.Amount != 100.0 {
		t.Errorf("Amount: %.2f, want 100.0", txn.Amount)
	}

	if txn.Currency != CurrencyUSD {
		t.Errorf("Currency: %s", txn.Currency)
	}

	if txn.Status == "" {
		t.Error("Status is empty")
	}

	if txn.Hash == "" {
		t.Error("Hash is empty")
	}
}

func TestIdempotencyKey(t *testing.T) {
	processor := NewProcessor()
	gateway := NewMockGateway(0.95)
	processor.RegisterGateway(gateway)

	ctx := context.Background()
	idempotencyKey := "idempotent-test-1"

	req := PaymentRequest{
		TenantID:       "tenant-1",
		SettlementID:   "settle-1",
		Amount:         50.0,
		Currency:       CurrencyUSD,
		Method:         MethodCreditCard,
		IdempotencyKey: idempotencyKey,
	}

	callsBefore := gateway.callCount
	txn1, _ := processor.ProcessPayment(ctx, req)
	callsAfterFirst := gateway.callCount

	// Same request with same key should return cached result
	txn2, _ := processor.ProcessPayment(ctx, req)
	callsAfterSecond := gateway.callCount

	// First call should invoke gateway
	if callsAfterFirst != callsBefore+1 {
		t.Error("First payment should call gateway")
	}

	// Second call with same key should NOT invoke gateway (cached)
	if callsAfterSecond != callsAfterFirst {
		t.Error("Idempotency should prevent second gateway call")
	}

	// Both should have same settlement ID and amount
	if txn1.SettlementID != txn2.SettlementID {
		t.Error("Settlement ID should be the same")
	}
	if txn1.Amount != txn2.Amount {
		t.Errorf("Amount mismatch: %.2f vs %.2f", txn1.Amount, txn2.Amount)
	}
}

func TestCreateWallet(t *testing.T) {
	processor := NewProcessor()
	ctx := context.Background()

	wallet, err := processor.CreateOrGetWallet(ctx, "operator-1", CurrencyUSD)
	if err != nil {
		t.Fatalf("CreateOrGetWallet failed: %v", err)
	}

	if wallet.ID == "" {
		t.Error("Wallet ID is empty")
	}

	if wallet.OwnerID != "operator-1" {
		t.Errorf("Owner ID: %s", wallet.OwnerID)
	}

	if wallet.Currency != CurrencyUSD {
		t.Errorf("Currency: %s", wallet.Currency)
	}

	if wallet.Status != "active" {
		t.Errorf("Status: %s", wallet.Status)
	}

	if wallet.Balance != 0 {
		t.Errorf("Initial balance: %.2f, want 0", wallet.Balance)
	}
}

func TestTopUpWallet(t *testing.T) {
	processor := NewProcessor()
	ctx := context.Background()

	processor.CreateOrGetWallet(ctx, "operator-1", CurrencyUSD)

	if err := processor.TopUpWallet(ctx, "operator-1", 1000.0, CurrencyUSD); err != nil {
		t.Fatalf("TopUpWallet failed: %v", err)
	}

	wallet, _ := processor.GetWallet(ctx, "operator-1", CurrencyUSD)

	if wallet.Balance != 1000.0 {
		t.Errorf("Balance: %.2f, want 1000.0", wallet.Balance)
	}

	if wallet.AvailableAmount != 1000.0 {
		t.Errorf("Available: %.2f, want 1000.0", wallet.AvailableAmount)
	}

	if wallet.LastTopUpAt == nil {
		t.Error("LastTopUpAt is nil")
	}
}

func TestHoldFunds(t *testing.T) {
	processor := NewProcessor()
	ctx := context.Background()

	processor.CreateOrGetWallet(ctx, "operator-1", CurrencyUSD)
	processor.TopUpWallet(ctx, "operator-1", 1000.0, CurrencyUSD)

	escrow, err := processor.HoldFunds(ctx, "operator-1", 500.0, CurrencyUSD)
	if err != nil {
		t.Fatalf("HoldFunds failed: %v", err)
	}

	if escrow.ID == "" {
		t.Error("Escrow ID is empty")
	}

	if escrow.HeldAmount != 500.0 {
		t.Errorf("Held amount: %.2f, want 500.0", escrow.HeldAmount)
	}

	if escrow.Status != "held" {
		t.Errorf("Status: %s", escrow.Status)
	}

	wallet, _ := processor.GetWallet(ctx, "operator-1", CurrencyUSD)

	if wallet.HoldAmount != 500.0 {
		t.Errorf("Hold amount: %.2f, want 500.0", wallet.HoldAmount)
	}

	if wallet.AvailableAmount != 500.0 {
		t.Errorf("Available: %.2f, want 500.0", wallet.AvailableAmount)
	}
}

func TestReleaseFunds(t *testing.T) {
	processor := NewProcessor()
	ctx := context.Background()

	processor.CreateOrGetWallet(ctx, "operator-1", CurrencyUSD)
	processor.CreateOrGetWallet(ctx, "operator-2", CurrencyUSD)
	processor.TopUpWallet(ctx, "operator-1", 1000.0, CurrencyUSD)

	escrow, _ := processor.HoldFunds(ctx, "operator-1", 500.0, CurrencyUSD)

	if err := processor.ReleaseFunds(ctx, escrow.ID, "operator-2"); err != nil {
		t.Fatalf("ReleaseFunds failed: %v", err)
	}

	wallet2, _ := processor.GetWallet(ctx, "operator-2", CurrencyUSD)
	if wallet2.Balance != 500.0 {
		t.Errorf("Recipient balance: %.2f, want 500.0", wallet2.Balance)
	}

	wallet1, _ := processor.GetWallet(ctx, "operator-1", CurrencyUSD)
	if wallet1.HoldAmount != 0 {
		t.Errorf("Sender hold: %.2f, want 0", wallet1.HoldAmount)
	}
}

func TestInsufficientFunds(t *testing.T) {
	processor := NewProcessor()
	ctx := context.Background()

	processor.CreateOrGetWallet(ctx, "operator-1", CurrencyUSD)
	processor.TopUpWallet(ctx, "operator-1", 100.0, CurrencyUSD)

	_, err := processor.HoldFunds(ctx, "operator-1", 500.0, CurrencyUSD)
	if err == nil {
		t.Error("Expected insufficient funds error")
	}
}

func TestListTransactions(t *testing.T) {
	processor := NewProcessor()
	gateway := NewMockGateway(0.95)
	processor.RegisterGateway(gateway)

	ctx := context.Background()

	// Create multiple transactions
	for i := 0; i < 5; i++ {
		req := PaymentRequest{
			TenantID:     "tenant-1",
			SettlementID: "settle-1",
			Amount:       100.0,
			Currency:     CurrencyUSD,
		}
		processor.ProcessPayment(ctx, req)
	}

	txns := processor.ListTransactions(ctx, "tenant-1")
	if len(txns) != 5 {
		t.Errorf("Expected 5 transactions, got %d", len(txns))
	}
}

func TestCurrencyConversion(t *testing.T) {
	processor := NewProcessor()

	// Set exchange rates
	processor.SetExchangeRate(CurrencyUSD, CurrencyEUR, 0.92)
	processor.SetExchangeRate(CurrencyEUR, CurrencyUSD, 1.087)
	processor.SetExchangeRate(CurrencyUSD, CurrencyGBP, 0.79)

	// Test conversions
	eurAmount, _ := processor.ConvertCurrency(CurrencyUSD, CurrencyEUR, 100.0)
	expectedEUR := 92.0
	if eurAmount != expectedEUR {
		t.Errorf("USD to EUR: %.2f, want %.2f", eurAmount, expectedEUR)
	}

	gbpAmount, _ := processor.ConvertCurrency(CurrencyUSD, CurrencyGBP, 100.0)
	expectedGBP := 79.0
	if gbpAmount != expectedGBP {
		t.Errorf("USD to GBP: %.2f, want %.2f", gbpAmount, expectedGBP)
	}

	// Same currency
	sameAmount, _ := processor.ConvertCurrency(CurrencyUSD, CurrencyUSD, 100.0)
	if sameAmount != 100.0 {
		t.Errorf("Same currency: %.2f, want 100.0", sameAmount)
	}

	// Missing rate
	_, err := processor.ConvertCurrency(CurrencyUSD, CurrencyJPY, 100.0)
	if err == nil {
		t.Error("Expected error for missing rate")
	}
}

func TestGetTransaction(t *testing.T) {
	processor := NewProcessor()
	gateway := NewMockGateway(0.95)
	processor.RegisterGateway(gateway)

	ctx := context.Background()

	req := PaymentRequest{
		TenantID:     "tenant-1",
		SettlementID: "settle-1",
		Amount:       100.0,
		Currency:     CurrencyUSD,
	}

	txn1, _ := processor.ProcessPayment(ctx, req)

	txn2, err := processor.GetTransaction(ctx, txn1.ID)
	if err != nil {
		t.Fatalf("GetTransaction failed: %v", err)
	}

	if txn1.ID != txn2.ID {
		t.Errorf("Transaction ID mismatch: %s vs %s", txn1.ID, txn2.ID)
	}

	if txn2.Amount != 100.0 {
		t.Errorf("Amount: %.2f", txn2.Amount)
	}
}

func TestInvalidTopUp(t *testing.T) {
	processor := NewProcessor()
	ctx := context.Background()

	processor.CreateOrGetWallet(ctx, "operator-1", CurrencyUSD)

	// Negative amount
	if err := processor.TopUpWallet(ctx, "operator-1", -50.0, CurrencyUSD); err == nil {
		t.Error("Expected error for negative amount")
	}

	// Zero amount
	if err := processor.TopUpWallet(ctx, "operator-1", 0, CurrencyUSD); err == nil {
		t.Error("Expected error for zero amount")
	}
}

func TestMultipleWalletsPerOwner(t *testing.T) {
	processor := NewProcessor()
	ctx := context.Background()

	processor.CreateOrGetWallet(ctx, "operator-1", CurrencyUSD)
	processor.CreateOrGetWallet(ctx, "operator-1", CurrencyEUR)
	processor.CreateOrGetWallet(ctx, "operator-1", CurrencyGBP)

	processor.TopUpWallet(ctx, "operator-1", 1000.0, CurrencyUSD)
	processor.TopUpWallet(ctx, "operator-1", 1000.0, CurrencyEUR)
	processor.TopUpWallet(ctx, "operator-1", 1000.0, CurrencyGBP)

	walletUSD, _ := processor.GetWallet(ctx, "operator-1", CurrencyUSD)
	walletEUR, _ := processor.GetWallet(ctx, "operator-1", CurrencyEUR)
	walletGBP, _ := processor.GetWallet(ctx, "operator-1", CurrencyGBP)

	if walletUSD.Balance != 1000.0 {
		t.Errorf("USD balance: %.2f", walletUSD.Balance)
	}

	if walletEUR.Balance != 1000.0 {
		t.Errorf("EUR balance: %.2f", walletEUR.Balance)
	}

	if walletGBP.Balance != 1000.0 {
		t.Errorf("GBP balance: %.2f", walletGBP.Balance)
	}
}
