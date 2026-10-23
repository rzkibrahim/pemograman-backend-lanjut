package model

import "time"

// Review adalah model domain murni untuk ulasan kelas (bebas dari tipe data driver BSON).
type Review struct {
	ID           string        `json:"id"`
	CourseID     int           `json:"course_id"`
	UserID       int           `json:"user_id"`
	Username     string        `json:"username"`
	Rating       int           `json:"rating"`
	Comment      string        `json:"comment"`
	Tags         []string      `json:"tags"`
	HelpfulCount int           `json:"helpful_count"`
	Replies      []ReviewReply `json:"replies"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// ReviewReply merepresentasikan balasan bertingkat pada ulasan (maksimal 50 balasan embedded).
type ReviewReply struct {
	ID        string    `json:"id"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateReviewRequest adalah payload pembuatan ulasan baru.
type CreateReviewRequest struct {
	Rating  int      `json:"rating"  validate:"required,min=1,max=5"`
	Comment string   `json:"comment" validate:"required,min=5,max=1000"`
	Tags    []string `json:"tags"    validate:"omitempty,max=5,dive,min=2,max=30"`
}

// UpdateReviewRequest adalah payload pembaruan ulasan oleh pemilik.
type UpdateReviewRequest struct {
	Rating  *int     `json:"rating"  validate:"omitempty,min=1,max=5"`
	Comment *string  `json:"comment" validate:"omitempty,min=5,max=1000"`
	Tags    []string `json:"tags"    validate:"omitempty,max=5,dive,min=2,max=30"`
}

// AddReplyRequest adalah payload untuk menambahkan balasan pada ulasan.
type AddReplyRequest struct {
	Comment string `json:"comment" validate:"required,min=2,max=500"`
}

// ReviewSummary adalah ringkasan agregasi ulasan kelas menggunakan $facet.
type ReviewSummary struct {
	AverageRating      float64        `json:"average_rating"`
	TotalReviews       int            `json:"total_reviews"`
	RatingDistribution map[string]int `json:"rating_distribution"` // "1": n, "2": n, dst
	TopTags            []TagCount     `json:"top_tags"`
}

// TagCount menyimpan jumlah kemunculan sebuah tag.
type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}
