package services

import (
	"context"
	"strings"
	"testing"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"

	"gorm.io/gorm"
)

func TestMaterialPublicQueryDeduplicatesByMD5(t *testing.T) {
	db := newDryRunDB(t)
	service := NewMaterialService(db)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		var materials []models.Material
		return service.materialPublicQuery(context.Background(), nil, false).
			Order("COALESCE(material_desc_stats.total_hotness, 0) DESC, materials.id DESC").
			Limit(10).
			Find(&materials)
	})

	normalizedSQL := strings.ToLower(sql)
	want := []string{
		"select max(id) as id",
		"group by `md5`",
		"canonical_materials.id = materials.id",
		"left join",
		"material_desc_stats",
		"materials.md5 = material_desc_stats.md5",
	}
	for _, item := range want {
		if !strings.Contains(normalizedSQL, item) {
			t.Fatalf("expected SQL to contain %q, got: %s", item, sql)
		}
	}
	if strings.Contains(normalizedSQL, "group by `file_name`") {
		t.Fatalf("expected SQL not to deduplicate by file_name, got: %s", sql)
	}
}

func TestMaterialSearchQueryUsesCategoryMatchSet(t *testing.T) {
	db := newDryRunDB(t)
	service := NewMaterialService(db)

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		var materials []models.Material
		return service.materialSearchQuery(context.Background(), "%考试%").
			Order("COALESCE(material_desc_stats.total_hotness, 0) DESC, materials.id DESC").
			Limit(10).
			Find(&materials)
	})

	normalizedSQL := strings.ToLower(sql)
	want := []string{
		"material_category_matches",
		"join material_categories on materials.category_id = material_categories.id",
		"material_category_matches.md5 = materials.md5",
		"material_category_matches.md5 is not null",
	}
	for _, item := range want {
		if !strings.Contains(normalizedSQL, item) {
			t.Fatalf("expected SQL to contain %q, got: %s", item, sql)
		}
	}
	if strings.Contains(normalizedSQL, "exists (") {
		t.Fatalf("expected SQL not to use a correlated EXISTS, got: %s", sql)
	}
}
