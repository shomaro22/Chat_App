package entity

import "testing"

func TestUserRoleConstants(t *testing.T) {
	if UserRoleUser != "user" || UserRoleAdmin != "admin" {
		t.Fatalf("user roles: %q %q", UserRoleUser, UserRoleAdmin)
	}
}

func TestRoomTypeConstants(t *testing.T) {
	if RoomTypeDirect != "direct" || RoomTypeGroup != "group" {
		t.Fatalf("room types: %q %q", RoomTypeDirect, RoomTypeGroup)
	}
}
