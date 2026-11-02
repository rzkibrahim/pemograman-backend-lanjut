package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"modul10/app/model"
)

// Sentinel errors untuk domain ulasan kelas
var (
	ErrReviewNotFound      = errors.New("REVIEW_NOT_FOUND")
	ErrAlreadyReviewed     = errors.New("ALREADY_REVIEWED")
	ErrMaxRepliesReached   = errors.New("MAX_REPLIES_REACHED")
	ErrAlreadyHelpful      = errors.New("ALREADY_MARKED_HELPFUL")
	ErrOwnReviewHelpful    = errors.New("OWN_REVIEW_HELPFUL")
	ErrNotReviewOwner      = errors.New("NOT_REVIEW_OWNER")
	ErrInvalidReviewCursor = errors.New("INVALID_REVIEW_CURSOR")
)

// reviewDoc adalah struct internal dokumen MongoDB untuk ulasan kelas.
type reviewDoc struct {
	ID           bson.ObjectID `bson:"_id,omitempty"`
	CourseID     int           `bson:"course_id"`
	UserID       int           `bson:"user_id"`
	Username     string        `bson:"username"`
	Rating       int           `bson:"rating"`
	Comment      string        `bson:"comment"`
	Tags         []string      `bson:"tags"`
	HelpfulCount int           `bson:"helpful_count"`
	HelpfulUsers []int         `bson:"helpful_users"`
	Replies      []replyDoc    `bson:"replies"`
	CreatedAt    time.Time     `bson:"created_at"`
	UpdatedAt    time.Time     `bson:"updated_at"`
}

type replyDoc struct {
	ID        bson.ObjectID `bson:"id"`
	UserID    int           `bson:"user_id"`
	Username  string        `bson:"username"`
	Comment   string        `bson:"comment"`
	CreatedAt time.Time     `bson:"created_at"`
}

type ReviewRepository struct {
	collection *mongo.Collection
}

func NewReviewRepository(db *mongo.Database) *ReviewRepository {
	return &ReviewRepository{
		collection: db.Collection("course_reviews"),
	}
}

// EnsureIndexes membuat indeks unik gabungan dan indeks multikey.
func (r *ReviewRepository) EnsureIndexes(ctx context.Context) error {
	indices := []mongo.IndexModel{
		// 1. Indeks unik gabungan: satu ulasan per mahasiswa per kelas
		{
			Keys: bson.D{
				{Key: "course_id", Value: 1},
				{Key: "user_id", Value: 1},
			},
			Options: options.Index().SetUnique(true).SetName("uniq_course_user_review"),
		},
		// 2. Indeks multikey pada array tags
		{
			Keys: bson.D{
				{Key: "course_id", Value: 1},
				{Key: "tags", Value: 1},
			},
			Options: options.Index().SetName("idx_course_tags"),
		},
		// 3. Indeks cursor pagination (_id descending)
		{
			Keys: bson.D{
				{Key: "course_id", Value: 1},
				{Key: "_id", Value: -1},
			},
			Options: options.Index().SetName("idx_course_cursor"),
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indices)
	if err != nil {
		return fmt.Errorf("membuat indeks ulasan: %w", err)
	}
	return nil
}

// Create menyisipkan dokumen ulasan baru.
func (r *ReviewRepository) Create(ctx context.Context, review model.Review) (model.Review, error) {
	now := time.Now().Truncate(time.Millisecond) // Cegah jebakan BSON Date nanodetik vs milidetik

	doc := reviewDoc{
		ID:           bson.NewObjectID(),
		CourseID:     review.CourseID,
		UserID:       review.UserID,
		Username:     review.Username,
		Rating:       review.Rating,
		Comment:      review.Comment,
		Tags:         review.Tags,
		HelpfulCount: 0,
		HelpfulUsers: []int{},
		Replies:      []replyDoc{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if doc.Tags == nil {
		doc.Tags = []string{}
	}

	_, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return model.Review{}, ErrAlreadyReviewed
		}
		return model.Review{}, fmt.Errorf("menyisipkan ulasan: %w", err)
	}

	return toReviewModel(doc), nil
}

// FindByID mengambil ulasan berdasarkan string hex ObjectID.
func (r *ReviewRepository) FindByID(ctx context.Context, id string) (model.Review, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return model.Review{}, ErrReviewNotFound
	}

	var doc reviewDoc
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Review{}, ErrReviewNotFound
		}
		return model.Review{}, fmt.Errorf("mengambil ulasan: %w", err)
	}

	return toReviewModel(doc), nil
}

// FindByCourse mengambil ulasan kelas dengan filter tag, cursor pagination, dan limit.
func (r *ReviewRepository) FindByCourse(ctx context.Context, courseID int, tag string, beforeID string, limit int) ([]model.Review, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	filter := bson.M{"course_id": courseID}
	if tag != "" {
		filter["tags"] = tag
	}

	if beforeID != "" {
		oid, err := bson.ObjectIDFromHex(beforeID)
		if err != nil {
			return nil, ErrInvalidReviewCursor
		}
		filter["_id"] = bson.M{"$lt": oid}
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "_id", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("mencari ulasan: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []reviewDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("membaca kursor ulasan: %w", err)
	}

	reviews := make([]model.Review, 0, len(docs))
	for _, d := range docs {
		reviews = append(reviews, toReviewModel(d))
	}
	return reviews, nil
}

// UpdateOwn mengubah rating, komentar, atau tag oleh pemilik ulasan.
func (r *ReviewRepository) UpdateOwn(ctx context.Context, reviewID string, userID int, rating *int, comment *string, tags []string) (model.Review, error) {
	oid, err := bson.ObjectIDFromHex(reviewID)
	if err != nil {
		return model.Review{}, ErrReviewNotFound
	}

	setFields := bson.M{
		"updated_at": time.Now().Truncate(time.Millisecond),
	}
	if rating != nil {
		setFields["rating"] = *rating
	}
	if comment != nil {
		setFields["comment"] = *comment
	}
	if tags != nil {
		setFields["tags"] = tags
	}

	filter := bson.M{
		"_id":     oid,
		"user_id": userID,
	}

	res, err := r.collection.UpdateOne(ctx, filter, bson.M{"$set": setFields})
	if err != nil {
		return model.Review{}, fmt.Errorf("memperbarui ulasan: %w", err)
	}

	if res.MatchedCount == 0 {
		// Periksa apakah dokumen ada tapi beda pemilik
		var existing reviewDoc
		if err := r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&existing); err == nil {
			return model.Review{}, ErrNotReviewOwner
		}
		return model.Review{}, ErrReviewNotFound
	}

	return r.FindByID(ctx, reviewID)
}

// Delete menghapus ulasan (bisa oleh pemilik atau moderator).
func (r *ReviewRepository) Delete(ctx context.Context, reviewID string, userID int, isModerator bool) error {
	oid, err := bson.ObjectIDFromHex(reviewID)
	if err != nil {
		return ErrReviewNotFound
	}

	filter := bson.M{"_id": oid}
	if !isModerator {
		filter["user_id"] = userID
	}

	res, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("menghapus ulasan: %w", err)
	}

	if res.DeletedCount == 0 {
		var existing reviewDoc
		if err := r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&existing); err == nil {
			return ErrNotReviewOwner
		}
		return ErrReviewNotFound
	}

	return nil
}

// AddReply menambahkan balasan bersarang (maksimal 50 balasan atomik menggunakan replies.49).
func (r *ReviewRepository) AddReply(ctx context.Context, reviewID string, reply model.ReviewReply) error {
	oid, err := bson.ObjectIDFromHex(reviewID)
	if err != nil {
		return ErrReviewNotFound
	}

	now := time.Now().Truncate(time.Millisecond)
	rep := replyDoc{
		ID:        bson.NewObjectID(),
		UserID:    reply.UserID,
		Username:  reply.Username,
		Comment:   reply.Comment,
		CreatedAt: now,
	}

	// Filter atomik: pastikan elemen ke-50 (indeks 49) belum ada
	filter := bson.M{
		"_id":        oid,
		"replies.49": bson.M{"$exists": false},
	}

	update := bson.M{
		"$push": bson.M{"replies": rep},
		"$set":  bson.M{"updated_at": now},
	}

	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("menambahkan balasan: %w", err)
	}

	if res.MatchedCount == 0 {
		var existing reviewDoc
		if err := r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&existing); err == nil {
			return ErrMaxRepliesReached
		}
		return ErrReviewNotFound
	}

	return nil
}

// MarkHelpful menandai ulasan membantu secara atomik:
// - Hanya sekali per orang ($addToSet)
// - Bukan ulasan sendiri (user_id != userID)
func (r *ReviewRepository) MarkHelpful(ctx context.Context, reviewID string, userID int) error {
	oid, err := bson.ObjectIDFromHex(reviewID)
	if err != nil {
		return ErrReviewNotFound
	}

	filter := bson.M{
		"_id":           oid,
		"user_id":       bson.M{"$ne": userID},
		"helpful_users": bson.M{"$ne": userID},
	}

	update := bson.M{
		"$addToSet": bson.M{"helpful_users": userID},
		"$inc":      bson.M{"helpful_count": 1},
	}

	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("menandai helpful: %w", err)
	}

	if res.MatchedCount == 0 {
		var existing reviewDoc
		err := r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&existing)
		if err != nil {
			return ErrReviewNotFound
		}
		if existing.UserID == userID {
			return ErrOwnReviewHelpful
		}
		return ErrAlreadyHelpful
	}

	return nil
}

// GetSummary menghitung rata-rata, sebaran bintang, dan top tags menggunakan aggregation pipeline $facet.
func (r *ReviewRepository) GetSummary(ctx context.Context, courseID int) (model.ReviewSummary, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"course_id": courseID}}},
		{{Key: "$facet", Value: bson.M{
			"stats": []bson.M{
				{
					"$group": bson.M{
						"_id":       nil,
						"avgRating": bson.M{"$avg": "$rating"},
						"total":     bson.M{"$sum": 1},
					},
				},
			},
			"ratings": []bson.M{
				{
					"$group": bson.M{
						"_id":   "$rating",
						"count": bson.M{"$sum": 1},
					},
				},
			},
			"tags": []bson.M{
				{"$unwind": "$tags"},
				{
					"$group": bson.M{
						"_id":   "$tags",
						"count": bson.M{"$sum": 1},
					},
				},
				{"$sort": bson.M{"count": -1}},
				{"$limit": 5},
			},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return model.ReviewSummary{}, fmt.Errorf("agregasi summary: %w", err)
	}
	defer cursor.Close(ctx)

	type facetResult struct {
		Stats []struct {
			AvgRating float64 `bson:"avgRating"`
			Total     int     `bson:"total"`
		} `bson:"stats"`
		Ratings []struct {
			Rating int `bson:"_id"`
			Count  int `bson:"count"`
		} `bson:"ratings"`
		Tags []struct {
			Tag   string `bson:"_id"`
			Count int    `bson:"count"`
		} `bson:"tags"`
	}

	var results []facetResult
	if err := cursor.All(ctx, &results); err != nil {
		return model.ReviewSummary{}, fmt.Errorf("membaca hasil agregasi: %w", err)
	}

	summary := model.ReviewSummary{
		RatingDistribution: map[string]int{"1": 0, "2": 0, "3": 0, "4": 0, "5": 0},
		TopTags:            []model.TagCount{},
	}

	if len(results) > 0 {
		res := results[0]
		if len(res.Stats) > 0 {
			summary.AverageRating = res.Stats[0].AvgRating
			summary.TotalReviews = res.Stats[0].Total
		}
		for _, r := range res.Ratings {
			summary.RatingDistribution[fmt.Sprintf("%d", r.Rating)] = r.Count
		}
		for _, t := range res.Tags {
			summary.TopTags = append(summary.TopTags, model.TagCount{
				Tag:   t.Tag,
				Count: t.Count,
			})
		}
	}

	return summary, nil
}

func toReviewModel(d reviewDoc) model.Review {
	replies := make([]model.ReviewReply, 0, len(d.Replies))
	for _, rep := range d.Replies {
		replies = append(replies, model.ReviewReply{
			ID:        rep.ID.Hex(),
			UserID:    rep.UserID,
			Username:  rep.Username,
			Comment:   rep.Comment,
			CreatedAt: rep.CreatedAt,
		})
	}

	return model.Review{
		ID:           d.ID.Hex(),
		CourseID:     d.CourseID,
		UserID:       d.UserID,
		Username:     d.Username,
		Rating:       d.Rating,
		Comment:      d.Comment,
		Tags:         d.Tags,
		HelpfulCount: d.HelpfulCount,
		Replies:      replies,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}
