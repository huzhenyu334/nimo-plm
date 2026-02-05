package repository

import (
	"context"
	"errors"
	"time"

	"github.com/bitfantasy/nimo-plm/internal/model/entity"
	"gorm.io/gorm"
)

// UserRepository 用户仓库
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓库
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// FindByID 根据ID查找用户
func (r *UserRepository) FindByID(ctx context.Context, id string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// FindByFeishuUserID 根据飞书用户ID查找用户
func (r *UserRepository) FindByFeishuUserID(ctx context.Context, feishuUserID string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).
		Where("feishu_user_id = ? AND deleted_at IS NULL", feishuUserID).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// FindByEmail 根据邮箱查找用户
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).
		Where("email = ? AND deleted_at IS NULL", email).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// Create 创建用户
func (r *UserRepository) Create(ctx context.Context, user *entity.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// Update 更新用户
func (r *UserRepository) Update(ctx context.Context, user *entity.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// Delete 软删除用户
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ?", id).
		Update("deleted_at", time.Now()).Error
}

// List 获取用户列表
func (r *UserRepository) List(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]entity.User, int64, error) {
	var users []entity.User
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.User{}).Where("deleted_at IS NULL")

	// 应用过滤条件
	if keyword, ok := filters["keyword"].(string); ok && keyword != "" {
		query = query.Where("name LIKE ? OR email LIKE ? OR employee_no LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if departmentID, ok := filters["department_id"].(string); ok && departmentID != "" {
		query = query.Where("department_id = ?", departmentID)
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Preload("Department").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&users).Error

	return users, total, err
}

// AssignRole 分配角色给用户
func (r *UserRepository) AssignRole(ctx context.Context, userID, roleID string) error {
	userRole := entity.UserRole{
		UserID:    userID,
		RoleID:    roleID,
		CreatedAt: time.Now(),
	}
	return r.db.WithContext(ctx).Create(&userRole).Error
}

// RemoveRole 移除用户角色
func (r *UserRepository) RemoveRole(ctx context.Context, userID, roleID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&entity.UserRole{}).Error
}

// LoadRolesAndPermissions 加载用户角色和权限
func (r *UserRepository) LoadRolesAndPermissions(ctx context.Context, user *entity.User) error {
	// 加载角色
	var roles []entity.Role
	err := r.db.WithContext(ctx).
		Table("roles").
		Joins("JOIN user_roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ?", user.ID).
		Find(&roles).Error
	if err != nil {
		return err
	}
	user.Roles = roles

	// 提取角色编码
	roleCodes := make([]string, len(roles))
	roleIDs := make([]string, len(roles))
	for i, role := range roles {
		roleCodes[i] = role.Code
		roleIDs[i] = role.ID
	}
	user.RoleCodes = roleCodes

	// 加载权限
	if len(roleIDs) > 0 {
		var permissions []entity.Permission
		err = r.db.WithContext(ctx).
			Table("permissions").
			Joins("JOIN role_permissions ON permissions.id = role_permissions.permission_id").
			Where("role_permissions.role_id IN ?", roleIDs).
			Distinct().
			Find(&permissions).Error
		if err != nil {
			return err
		}

		permCodes := make([]string, len(permissions))
		for i, perm := range permissions {
			permCodes[i] = perm.Code
		}
		user.PermissionCodes = permCodes
	}

	return nil
}

// GetAllRoles 获取所有角色
func (r *UserRepository) GetAllRoles(ctx context.Context) ([]entity.Role, error) {
	var roles []entity.Role
	err := r.db.WithContext(ctx).
		Order("is_system DESC, name ASC").
		Find(&roles).Error
	return roles, err
}

// GetUserRoles 获取用户角色
func (r *UserRepository) GetUserRoles(ctx context.Context, userID string) ([]entity.Role, error) {
	var roles []entity.Role
	err := r.db.WithContext(ctx).
		Table("roles").
		Joins("JOIN user_roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ?", userID).
		Find(&roles).Error
	return roles, err
}
