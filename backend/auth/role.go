package auth

import "chat-app/entity"

// IsAdmin はロールが管理者かどうか（認可補助）。
func IsAdmin(role entity.UserRole) bool {
	return role == entity.UserRoleAdmin
}
