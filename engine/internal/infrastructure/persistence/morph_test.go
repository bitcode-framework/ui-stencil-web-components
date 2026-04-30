package persistence

import (
	"context"
	"testing"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupMorphTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	sqlDB, _ := db.DB()

	sqlDB.Exec(`CREATE TABLE posts (
		id TEXT PRIMARY KEY,
		title TEXT,
		active INTEGER DEFAULT 1,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	sqlDB.Exec(`CREATE TABLE videos (
		id TEXT PRIMARY KEY,
		title TEXT,
		active INTEGER DEFAULT 1,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	sqlDB.Exec(`CREATE TABLE comments (
		id TEXT PRIMARY KEY,
		body TEXT,
		commentable_type TEXT,
		commentable_id TEXT,
		active INTEGER DEFAULT 1,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	sqlDB.Exec(`CREATE INDEX idx_comments_commentable ON comments (commentable_type, commentable_id)`)

	sqlDB.Exec(`CREATE TABLE images (
		id TEXT PRIMARY KEY,
		url TEXT,
		imageable_type TEXT,
		imageable_id TEXT,
		active INTEGER DEFAULT 1,
		created_at DATETIME,
		updated_at DATETIME
	)`)

	sqlDB.Exec(`CREATE TABLE tags (
		id TEXT PRIMARY KEY,
		name TEXT,
		active INTEGER DEFAULT 1,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	sqlDB.Exec(`CREATE TABLE taggables (
		id TEXT PRIMARY KEY,
		tag_id TEXT NOT NULL,
		taggable_id TEXT NOT NULL,
		taggable_type TEXT NOT NULL
	)`)
	sqlDB.Exec(`CREATE INDEX idx_taggables_tag_id ON taggables (tag_id)`)
	sqlDB.Exec(`CREATE INDEX idx_taggables_morph ON taggables (taggable_type, taggable_id)`)

	sqlDB.Exec(`INSERT INTO posts (id, title, active) VALUES ('p1', 'First Post', 1), ('p2', 'Second Post', 1)`)
	sqlDB.Exec(`INSERT INTO videos (id, title, active) VALUES ('v1', 'Cool Video', 1)`)
	sqlDB.Exec(`INSERT INTO comments (id, body, commentable_type, commentable_id, active) VALUES
		('c1', 'Great post!', 'post', 'p1', 1),
		('c2', 'Nice video!', 'video', 'v1', 1),
		('c3', 'Another comment', 'post', 'p1', 1)`)
	sqlDB.Exec(`INSERT INTO images (id, url, imageable_type, imageable_id, active) VALUES
		('img1', '/avatar.png', 'post', 'p1', 1),
		('img2', '/thumb.png', 'video', 'v1', 1)`)
	sqlDB.Exec(`INSERT INTO tags (id, name, active) VALUES ('t1', 'go', 1), ('t2', 'tutorial', 1), ('t3', 'video', 1)`)
	sqlDB.Exec(`INSERT INTO taggables (id, tag_id, taggable_id, taggable_type) VALUES
		('j1', 't1', 'p1', 'post'),
		('j2', 't2', 'p1', 'post'),
		('j3', 't3', 'v1', 'video')`)

	return db
}

func postModelDef() *parser.ModelDefinition {
	return &parser.ModelDefinition{
		Name: "post",
		Fields: map[string]parser.FieldDefinition{
			"title":    {Type: parser.FieldString},
			"comments": {Type: parser.FieldMorphMany, Model: "comment", Morph: "commentable"},
			"image":    {Type: parser.FieldMorphOne, Model: "image", Morph: "imageable"},
			"tags":     {Type: parser.FieldMorphToMany, Model: "tag", Morph: "taggable"},
		},
	}
}

func commentModelDef() *parser.ModelDefinition {
	return &parser.ModelDefinition{
		Name: "comment",
		Fields: map[string]parser.FieldDefinition{
			"body":        {Type: parser.FieldText},
			"commentable": {Type: parser.FieldMorphTo, Models: []string{"post", "video"}},
		},
	}
}

func tagModelDef() *parser.ModelDefinition {
	return &parser.ModelDefinition{
		Name: "tag",
		Fields: map[string]parser.FieldDefinition{
			"name":   {Type: parser.FieldString},
			"posts":  {Type: parser.FieldMorphByMany, Model: "post", Morph: "taggable"},
			"videos": {Type: parser.FieldMorphByMany, Model: "video", Morph: "taggable"},
		},
	}
}

type testTableResolver struct{}

func (r *testTableResolver) TableName(model string) string {
	switch model {
	case "post":
		return "posts"
	case "video":
		return "videos"
	case "comment":
		return "comments"
	case "image":
		return "images"
	case "tag":
		return "tags"
	}
	return model
}

func TestMorphMany_LoadComments(t *testing.T) {
	db := setupMorphTestDB(t)
	repo := NewGenericRepositoryWithModel(db, "posts", postModelDef())
	repo.SetModelName("post")
	repo.SetTableNameResolver(&testTableResolver{})

	query := NewQuery()
	query.With = append(query.With, WithClause{Relation: "comments"})

	results, _, err := repo.FindAll(context.Background(), query, 1, 20)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(results))
	}

	for _, rec := range results {
		pid := rec["id"].(string)
		comments, ok := rec["_comments"].([]map[string]any)
		if !ok {
			t.Fatalf("post %s: _comments not loaded", pid)
		}
		if pid == "p1" && len(comments) != 2 {
			t.Errorf("post p1: expected 2 comments, got %d", len(comments))
		}
		if pid == "p2" && len(comments) != 0 {
			t.Errorf("post p2: expected 0 comments, got %d", len(comments))
		}
	}
}

func TestMorphOne_LoadImage(t *testing.T) {
	db := setupMorphTestDB(t)
	repo := NewGenericRepositoryWithModel(db, "posts", postModelDef())
	repo.SetModelName("post")
	repo.SetTableNameResolver(&testTableResolver{})

	query := NewQuery()
	query.With = append(query.With, WithClause{Relation: "image"})

	results, _, err := repo.FindAll(context.Background(), query, 1, 20)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}

	for _, rec := range results {
		pid := rec["id"].(string)
		if pid == "p1" {
			img, ok := rec["_image"].(map[string]any)
			if !ok {
				t.Fatal("post p1: _image not loaded")
			}
			if img["url"] != "/avatar.png" {
				t.Errorf("expected /avatar.png, got %v", img["url"])
			}
		}
	}
}

func TestMorphTo_LoadParent(t *testing.T) {
	db := setupMorphTestDB(t)
	repo := NewGenericRepositoryWithModel(db, "comments", commentModelDef())
	repo.SetModelName("comment")
	repo.SetTableNameResolver(&testTableResolver{})

	query := NewQuery()
	query.With = append(query.With, WithClause{Relation: "commentable"})

	results, _, err := repo.FindAll(context.Background(), query, 1, 20)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 comments, got %d", len(results))
	}

	for _, rec := range results {
		cid := rec["id"].(string)
		parent, ok := rec["_commentable"].(map[string]any)
		if !ok {
			t.Fatalf("comment %s: _commentable not loaded", cid)
		}
		parentType, _ := rec["_commentable_type"].(string)

		switch cid {
		case "c1", "c3":
			if parentType != "post" {
				t.Errorf("comment %s: expected type post, got %s", cid, parentType)
			}
			if parent["title"] != "First Post" {
				t.Errorf("comment %s: expected title 'First Post', got %v", cid, parent["title"])
			}
		case "c2":
			if parentType != "video" {
				t.Errorf("comment c2: expected type video, got %s", parentType)
			}
			if parent["title"] != "Cool Video" {
				t.Errorf("comment c2: expected title 'Cool Video', got %v", parent["title"])
			}
		}
	}
}

func TestMorphToMany_LoadTags(t *testing.T) {
	db := setupMorphTestDB(t)
	repo := NewGenericRepositoryWithModel(db, "posts", postModelDef())
	repo.SetModelName("post")
	repo.SetTableNameResolver(&testTableResolver{})

	query := NewQuery()
	query.With = append(query.With, WithClause{Relation: "tags"})

	results, _, err := repo.FindAll(context.Background(), query, 1, 20)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}

	for _, rec := range results {
		pid := rec["id"].(string)
		tags, ok := rec["_tags"].([]map[string]any)
		if !ok {
			t.Fatalf("post %s: _tags not loaded", pid)
		}
		if pid == "p1" && len(tags) != 2 {
			t.Errorf("post p1: expected 2 tags, got %d", len(tags))
		}
		if pid == "p2" && len(tags) != 0 {
			t.Errorf("post p2: expected 0 tags, got %d", len(tags))
		}
	}
}

func TestMorphByMany_LoadPosts(t *testing.T) {
	db := setupMorphTestDB(t)
	repo := NewGenericRepositoryWithModel(db, "tags", tagModelDef())
	repo.SetModelName("tag")
	repo.SetTableNameResolver(&testTableResolver{})

	query := NewQuery()
	query.With = append(query.With, WithClause{Relation: "posts"})

	results, _, err := repo.FindAll(context.Background(), query, 1, 20)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}

	for _, rec := range results {
		tid := rec["id"].(string)
		posts, ok := rec["_posts"].([]map[string]any)
		if !ok {
			t.Fatalf("tag %s: _posts not loaded", tid)
		}
		if tid == "t1" && len(posts) != 1 {
			t.Errorf("tag t1 (go): expected 1 post, got %d", len(posts))
		}
		if tid == "t3" && len(posts) != 0 {
			t.Errorf("tag t3 (video): expected 0 posts (it's on video not post), got %d", len(posts))
		}
	}
}

func TestMorphAttachDetachSync(t *testing.T) {
	db := setupMorphTestDB(t)
	repo := NewGenericRepositoryWithModel(db, "posts", postModelDef())
	repo.SetModelName("post")
	repo.SetTableNameResolver(&testTableResolver{})

	ctx := context.Background()

	if err := repo.MorphAttach(ctx, "taggable", "tag", "p2", []string{"t1", "t3"}); err != nil {
		t.Fatalf("MorphAttach failed: %v", err)
	}

	query := NewQuery()
	query.With = append(query.With, WithClause{Relation: "tags"})
	results, _, _ := repo.FindAll(ctx, query, 1, 20)
	for _, rec := range results {
		if rec["id"].(string) == "p2" {
			tags := rec["_tags"].([]map[string]any)
			if len(tags) != 2 {
				t.Errorf("after attach: expected 2 tags on p2, got %d", len(tags))
			}
		}
	}

	if err := repo.MorphDetach(ctx, "taggable", "tag", "p2", []string{"t3"}); err != nil {
		t.Fatalf("MorphDetach failed: %v", err)
	}

	results, _, _ = repo.FindAll(ctx, query, 1, 20)
	for _, rec := range results {
		if rec["id"].(string) == "p2" {
			tags := rec["_tags"].([]map[string]any)
			if len(tags) != 1 {
				t.Errorf("after detach: expected 1 tag on p2, got %d", len(tags))
			}
		}
	}

	if err := repo.MorphSync(ctx, "taggable", "tag", "p2", []string{"t2"}); err != nil {
		t.Fatalf("MorphSync failed: %v", err)
	}

	results, _, _ = repo.FindAll(ctx, query, 1, 20)
	for _, rec := range results {
		if rec["id"].(string) == "p2" {
			tags := rec["_tags"].([]map[string]any)
			if len(tags) != 1 {
				t.Errorf("after sync: expected 1 tag on p2, got %d", len(tags))
			}
			if len(tags) > 0 && tags[0]["name"] != "tutorial" {
				t.Errorf("after sync: expected tag 'tutorial', got %v", tags[0]["name"])
			}
		}
	}
}

func TestMorphMigration_CreatesColumnsAndJunction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	commentModel := &parser.ModelDefinition{
		Name: "comment",
		Fields: map[string]parser.FieldDefinition{
			"body":        {Type: parser.FieldText},
			"commentable": {Type: parser.FieldMorphTo},
		},
	}

	postModel := &parser.ModelDefinition{
		Name: "post",
		Fields: map[string]parser.FieldDefinition{
			"title": {Type: parser.FieldString},
			"tags":  {Type: parser.FieldMorphToMany, Model: "tag", Morph: "taggable"},
		},
	}

	resolver := &testTableResolver{}

	if err := MigrateModel(db, commentModel, resolver); err != nil {
		t.Fatalf("MigrateModel(comment) failed: %v", err)
	}
	if err := MigrateModel(db, postModel, resolver); err != nil {
		t.Fatalf("MigrateModel(post) failed: %v", err)
	}

	if !db.Migrator().HasTable("comments") {
		t.Error("expected comments table to exist")
	}
	if !db.Migrator().HasTable("taggables") {
		t.Error("expected taggables junction table to exist")
	}

	var count int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info('comments') WHERE name IN ('commentable_type', 'commentable_id')").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 morph columns (commentable_type, commentable_id), got %d", count)
	}
}
