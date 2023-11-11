package bfrest

// Define a custom type for the transaction to ensure it only accepts specific values
type TransactionType string

const (
    GET    TransactionType = "GET"
    PUT    TransactionType = "PUT"
    POST   TransactionType = "POST"
    DELETE TransactionType = "DELETE"
)

// Bfrest struct definition
type Bfrest struct {
    TransactionID int64           // 64-bit integer, renamed from ID to TransactionID
    URL           string          // String
    Payload       string          // String
    Transaction   TransactionType // Custom type for transaction
}
