package event

const (
	// =========================
	// Cart
	// =========================
	CartItemAdded = "cart.item.added"

	// =========================
	// Checkout
	// =========================
	CheckoutStarted = "checkout.started"
	CheckoutPaid    = "checkout.paid"
	CheckoutFailed  = "checkout.failed"

	// =========================
	// Orders
	// =========================
	OrderCreated        = "order.created"
	OrderPaid           = "order.paid"
	OrderSellerAccepted = "order.seller.accepted"
	OrderSellerRejected = "order.seller.rejected"
	OrderShipped        = "order.shipped"
	OrderDelivered      = "order.delivered"
	OrderRefunded       = "order.refunded"

	// =========================
	// Escrow / Payout
	// =========================
	EscrowHeld     = "escrow.held"
	EscrowReleased = "escrow.released"
	EscrowRefunded = "escrow.refunded"

	// =========================
	// Auth
	// =========================
	UserRegistered = "auth.user.registered"
	UserLoggedIn   = "auth.user.logged_in"
	UserLoggedOut  = "auth.user.logged_out"

	PasswordResetRequested = "auth.password_reset.requested"
	PasswordResetCompleted = "auth.password_reset.completed"

	// Listing
	ListingCreated  = "listing.created"
	ListingUpdated  = "listing.updated"
	ListingRemoved  = "listing.removed"
	ListingRejected = "listing.rejected"

	// Wishlist
	WishlistAdded   = "wishlist.added"
	WishlistRemoved = "wishlist.removed"

	// Refund
	RefundRequested = "refund.requested"
	RefundAccepted  = "refund.accepted"
	RefundRejected  = "refund.rejected"
	RefundForced    = "refund.forced"
	RefundCompleted = "refund.completed"
)
