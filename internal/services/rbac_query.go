package services

import (
	"fmt"

	"gorm.io/gorm"
)

func whereNotDeleted(db *gorm.DB, tableAliases ...string) *gorm.DB {
	for _, tableAlias := range tableAliases {
		db = db.Where(fmt.Sprintf("%s.deleted_at IS NULL", tableAlias))
	}
	return db
}

func userRoleTagsQuery(db *gorm.DB, userID uint) *gorm.DB {
	return whereNotDeleted(
		db.Table("roles").
			Select("DISTINCT roles.role_tag").
			Joins("JOIN user_roles ur ON ur.role_id = roles.id").
			Where("ur.user_id = ?", userID),
		"roles", "ur",
	)
}

func userPermissionTagsQuery(db *gorm.DB, userID uint) *gorm.DB {
	return whereNotDeleted(
		db.Table("permissions").
			Select("DISTINCT permissions.permission_tag").
			Joins("JOIN role_permissions rp ON rp.permission_id = permissions.id").
			Joins("JOIN user_roles ur ON ur.role_id = rp.role_id").
			Where("ur.user_id = ?", userID),
		"permissions", "rp", "ur",
	)
}

func usersByRoleTagsQuery(db *gorm.DB, roleTags []string) *gorm.DB {
	return whereNotDeleted(
		db.Table("users").
			Select("DISTINCT users.*").
			Joins("JOIN user_roles ur ON ur.user_id = users.id").
			Joins("JOIN roles r ON r.id = ur.role_id").
			Where("r.role_tag IN ?", roleTags),
		"users", "ur", "r",
	)
}

func backofficeUsersByPhoneQuery(db *gorm.DB, phone string) *gorm.DB {
	return whereNotDeleted(
		db.Table("users").
			Distinct("users.id").
			Joins("JOIN user_roles ur ON ur.user_id = users.id").
			Joins("JOIN roles ON roles.id = ur.role_id").
			Where("users.phone = ?", phone).
			Where("roles.role_tag IN ?", backofficeRoleTags),
		"users", "ur", "roles",
	)
}
