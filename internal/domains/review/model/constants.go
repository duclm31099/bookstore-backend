package model

const (
	// Review eligibility
	ReviewWindowDays = 90 // Can review within 90 days of delivery

	// Edit/Delete windows
	EditWindowDays   = 7  // Can edit within 7 days
	DeleteWindowDays = 30 // Can delete within 30 days

	// Content limits
	MinContentLength = 10
	MaxContentLength = 2000
	MaxImages        = 5

	// Rating
	MinRating = 1
	MaxRating = 5
)
