package entity

import (
	"time"
)

// User 用户实体
type User struct {
	ID            string     `json:"id" gorm:"primaryKey;size:32"`
	FeishuUserID  string     `json:"feishu_user_id" gorm:"size:64;uniqueIndex"`
	FeishuUnionID string     `json:"feishu_union_id" gorm:"size:64"`
	FeishuOpenID  string     `json:"feishu_open_id" gorm:"size:64"`
	EmployeeNo    string     `json:"employee_no" gorm:"size:32;index"`
	Username      string     `json:"username" gorm:"size:64;not null;uniqueIndex"`
	Name          string     `json:"name" gorm:"size:64;not null"`
	Email         string     `json:"email" gorm:"size:128;uniqueIndex"`
	Mobile        string     `json:"mobile" gorm:"size:20"`
	AvatarURL     string     `json:"avatar_url" gorm:"size:512"`
	DepartmentID  string     `json:"department_id" gorm:"size:32"`
	Status        string     `json:"status" gorm:"size:16;not null;default:active"`
	LastLoginAt   *time.Time `json:"last_login_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at" gorm:"index"`

	// 关联
	Department *Department `json:"department,omitempty" gorm:"foreignKey:DepartmentID"`
	Roles      []Role      `json:"roles,omitempty" gorm:"many2many:user_roles;"`

	// 非数据库字段
	RoleCodes       []string `json:"role_codes,omitempty" gorm:"-"`
	PermissionCodes []string `json:"permission_codes,omitempty" gorm:"-"`
}

func (User) TableName() string {
	return "users"
}

// Department 部门实体
type Department struct {
	ID           string    `json:"id" gorm:"primaryKey;size:32"`
	FeishuDeptID string    `json:"feishu_dept_id" gorm:"size:64;uniqueIndex"`
	Name         string    `json:"name" gorm:"size:128;not null"`
	ParentID     string    `json:"parent_id" gorm:"size:32"`
	Path         string    `json:"path" gorm:"size:512"`
	Level        int       `json:"level" gorm:"not null;default:1"`
	SortOrder    int       `json:"sort_order" gorm:"not null;default:0"`
	LeaderID     string    `json:"leader_id" gorm:"size:32"`
	Status       string    `json:"status" gorm:"size:16;not null;default:active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// 关联
	Parent   *Department  `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	Children []Department `json:"children,omitempty" gorm:"foreignKey:ParentID"`
	Leader   *User        `json:"leader,omitempty" gorm:"foreignKey:LeaderID"`
}

func (Department) TableName() string {
	return "departments"
}

// Role 角色实体
type Role struct {
	ID          string    `json:"id" gorm:"primaryKey;size:32"`
	Code        string    `json:"code" gorm:"size:64;not null;uniqueIndex"`
	Name        string    `json:"name" gorm:"size:64;not null"`
	Description string    `json:"description" gorm:"type:text"`
	IsSystem    bool      `json:"is_system" gorm:"not null;default:false"`
	Status      string    `json:"status" gorm:"size:16;not null;default:active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 关联
	Permissions []Permission `json:"permissions,omitempty" gorm:"many2many:role_permissions;"`
}

func (Role) TableName() string {
	return "roles"
}

// Permission 权限实体
type Permission struct {
	ID          string    `json:"id" gorm:"primaryKey;size:32"`
	Code        string    `json:"code" gorm:"size:128;not null;uniqueIndex"`
	Name        string    `json:"name" gorm:"size:64;not null"`
	Module      string    `json:"module" gorm:"size:32;not null"`
	Description string    `json:"description" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
}

func (Permission) TableName() string {
	return "permissions"
}

// UserRole 用户角色关联
type UserRole struct {
	UserID    string    `json:"user_id" gorm:"primaryKey;size:32"`
	RoleID    string    `json:"role_id" gorm:"primaryKey;size:32"`
	CreatedAt time.Time `json:"created_at"`
}

func (UserRole) TableName() string {
	return "user_roles"
}

// RolePermission 角色权限关联
type RolePermission struct {
	RoleID       string    `json:"role_id" gorm:"primaryKey;size:32"`
	PermissionID string    `json:"permission_id" gorm:"primaryKey;size:32"`
	CreatedAt    time.Time `json:"created_at"`
}

func (RolePermission) TableName() string {
	return "role_permissions"
}
