package model

// Cart business constraints
const (
	// MaxItemsPerProduct is the maximum quantity allowed for a single product in cart
	MaxItemsPerProduct = 100

	// DefaultCartExpirationDays is the default number of days before a cart expires
	DefaultCartExpirationDays = 30

	// CartCacheExpirationMinutes is how long to cache cart data
	CartCacheExpirationMinutes = 5
)

// Pagination defaults
const (
	// DefaultPageSize is the default number of items per page
	DefaultPageSize = 20

	// MaxPageSize is the maximum number of items per page
	MaxPageSize = 100
)

// Cache keys
const (
	// CacheKeyCartByUser format: "cart:user:{userID}"
	CacheKeyCartByUser = "cart:user:%s"

	// CacheKeyCartBySession format: "cart:session:{sessionID}"
	CacheKeyCartBySession = "cart:session:%s"

	// CacheKeyCartByID format: "cart:id:{cartID}"
	CacheKeyCartByID = "cart:id:%s"
)
