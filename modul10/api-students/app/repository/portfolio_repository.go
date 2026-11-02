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

var (
	ErrPortfolioNotFound  = errors.New("PORTFOLIO_NOT_FOUND")
	ErrMaxProjectsReached = errors.New("MAX_PROJECTS_REACHED")
	ErrProjectNotFound    = errors.New("PROJECT_NOT_FOUND")
)

type portfolioDoc struct {
	ID           bson.ObjectID    `bson:"_id,omitempty"`
	StudentID    int              `bson:"student_id"`
	Headline     string           `bson:"headline"`
	Bio          string           `bson:"bio"`
	Projects     []projectDoc     `bson:"projects"`
	Certificates []certificateDoc `bson:"certificates"`
	CreatedAt    time.Time        `bson:"created_at"`
	UpdatedAt    time.Time        `bson:"updated_at"`
}

type projectDoc struct {
	ID           bson.ObjectID `bson:"id"`
	Title        string        `bson:"title"`
	Technologies []string      `bson:"technologies"`
	Link         string        `bson:"link,omitempty"`
	Year         int           `bson:"year"`
}

type certificateDoc struct {
	ID     bson.ObjectID `bson:"id"`
	Name   string        `bson:"name"`
	Issuer string        `bson:"issuer"`
	Date   string        `bson:"date"`
}

type PortfolioRepository struct {
	collection *mongo.Collection
}

func NewPortfolioRepository(db *mongo.Database) *PortfolioRepository {
	return &PortfolioRepository{
		collection: db.Collection("student_portfolios"),
	}
}

// EnsureIndexes membuat unique index pada student_id (satu portofolio per mahasiswa).
func (r *PortfolioRepository) EnsureIndexes(ctx context.Context) error {
	index := mongo.IndexModel{
		Keys:    bson.D{{Key: "student_id", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("uniq_student_portfolio"),
	}
	_, err := r.collection.Indexes().CreateOne(ctx, index)
	if err != nil {
		return fmt.Errorf("membuat index portofolio: %w", err)
	}
	return nil
}

// FindByStudentID mengambil portofolio berdasarkan ID mahasiswa.
func (r *PortfolioRepository) FindByStudentID(ctx context.Context, studentID int) (model.StudentPortfolio, error) {
	var doc portfolioDoc
	err := r.collection.FindOne(ctx, bson.M{"student_id": studentID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.StudentPortfolio{}, ErrPortfolioNotFound
		}
		return model.StudentPortfolio{}, fmt.Errorf("mengambil portofolio: %w", err)
	}
	return toPortfolioModel(doc), nil
}

// Upsert membuat atau memperbarui profil portofolio (headline dan bio).
func (r *PortfolioRepository) Upsert(ctx context.Context, studentID int, headline, bio string) (model.StudentPortfolio, error) {
	now := time.Now().Truncate(time.Millisecond)

	filter := bson.M{"student_id": studentID}
	update := bson.M{
		"$set": bson.M{
			"headline":   headline,
			"bio":        bio,
			"updated_at": now,
		},
		"$setOnInsert": bson.M{
			"student_id":   studentID,
			"projects":     []projectDoc{},
			"certificates": []certificateDoc{},
			"created_at":   now,
		},
	}

	opts := options.UpdateOne().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return model.StudentPortfolio{}, fmt.Errorf("upsert portofolio: %w", err)
	}

	return r.FindByStudentID(ctx, studentID)
}

// AddProject menambahkan proyek ke portofolio secara atomik dengan batas maksimal 20 proyek.
func (r *PortfolioRepository) AddProject(ctx context.Context, studentID int, project model.PortfolioProject) (string, error) {
	now := time.Now().Truncate(time.Millisecond)
	pid := bson.NewObjectID()

	pDoc := projectDoc{
		ID:           pid,
		Title:        project.Title,
		Technologies: project.Technologies,
		Link:         project.Link,
		Year:         project.Year,
	}

	// Filter atomik: pastikan elemen ke-20 (indeks 19) belum ada
	filter := bson.M{
		"student_id":  studentID,
		"projects.19": bson.M{"$exists": false},
	}

	update := bson.M{
		"$push": bson.M{"projects": pDoc},
		"$set":  bson.M{"updated_at": now},
	}

	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return "", fmt.Errorf("menambahkan proyek: %w", err)
	}

	if res.MatchedCount == 0 {
		var existing portfolioDoc
		err := r.collection.FindOne(ctx, bson.M{"student_id": studentID}).Decode(&existing)
		if err != nil {
			return "", ErrPortfolioNotFound
		}
		return "", ErrMaxProjectsReached
	}

	return pid.Hex(), nil
}

// DeleteProject menghapus proyek dari array projects ($pull).
func (r *PortfolioRepository) DeleteProject(ctx context.Context, studentID int, projectID string) error {
	pid, err := bson.ObjectIDFromHex(projectID)
	if err != nil {
		return ErrProjectNotFound
	}

	filter := bson.M{
		"student_id":  studentID,
		"projects.id": pid,
	}

	update := bson.M{
		"$pull": bson.M{"projects": bson.M{"id": pid}},
		"$set":  bson.M{"updated_at": time.Now().Truncate(time.Millisecond)},
	}

	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("menghapus proyek: %w", err)
	}

	if res.MatchedCount == 0 {
		var existing portfolioDoc
		if err := r.collection.FindOne(ctx, bson.M{"student_id": studentID}).Decode(&existing); err != nil {
			return ErrPortfolioNotFound
		}
		return ErrProjectNotFound
	}

	return nil
}

// GetTechnologyStats menghitung top 10 teknologi menggunakan aggregation pipeline (dua kali $unwind).
func (r *PortfolioRepository) GetTechnologyStats(ctx context.Context) ([]model.TechnologyStat, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$unwind", Value: "$projects"}},
		{{Key: "$unwind", Value: "$projects.technologies"}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$projects.technologies",
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.D{
			{Key: "count", Value: -1},
			{Key: "_id", Value: 1},
		}}},
		{{Key: "$limit", Value: 10}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("agregasi teknologi: %w", err)
	}
	defer cursor.Close(ctx)

	type resultDoc struct {
		Technology string `bson:"_id"`
		Count      int    `bson:"count"`
	}

	var results []resultDoc
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("membaca hasil agregasi teknologi: %w", err)
	}

	stats := make([]model.TechnologyStat, 0, len(results))
	for _, res := range results {
		stats = append(stats, model.TechnologyStat{
			Technology: res.Technology,
			Count:      res.Count,
		})
	}

	return stats, nil
}

func toPortfolioModel(d portfolioDoc) model.StudentPortfolio {
	projects := make([]model.PortfolioProject, 0, len(d.Projects))
	for _, p := range d.Projects {
		projects = append(projects, model.PortfolioProject{
			ID:           p.ID.Hex(),
			Title:        p.Title,
			Technologies: p.Technologies,
			Link:         p.Link,
			Year:         p.Year,
		})
	}

	certs := make([]model.PortfolioCertificate, 0, len(d.Certificates))
	for _, c := range d.Certificates {
		certs = append(certs, model.PortfolioCertificate{
			ID:     c.ID.Hex(),
			Name:   c.Name,
			Issuer: c.Issuer,
			Date:   c.Date,
		})
	}

	return model.StudentPortfolio{
		ID:           d.ID.Hex(),
		StudentID:    d.StudentID,
		Headline:     d.Headline,
		Bio:          d.Bio,
		Projects:     projects,
		Certificates: certs,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}
