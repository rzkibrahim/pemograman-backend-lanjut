package helper

import (
	"testing"
)

func TestPermissionSet_FailClosed(t *testing.T) {
	mapping := map[string][]string{
		"admin": {"user:list", "user:read:any", "student:create"},
		"staff": {"user:list", "student:read:any"},
	}
	ps := NewPermissionSet(mapping)

	tests := []struct {
		name       string
		ps         *PermissionSet
		role       string
		permission string
		expected   bool
	}{
		{
			name:       "Admin memiliki user:list -> True",
			ps:         ps,
			role:       "admin",
			permission: "user:list",
			expected:   true,
		},
		{
			name:       "Admin case-insensitive -> True",
			ps:         ps,
			role:       "ADMIN",
			permission: "USER:LIST",
			expected:   true,
		},
		{
			name:       "Staff tidak memiliki student:create -> False",
			ps:         ps,
			role:       "staff",
			permission: "student:create",
			expected:   false,
		},
		{
			name:       "Role user kosong -> False",
			ps:         ps,
			role:       "user",
			permission: "user:list",
			expected:   false,
		},
		{
			name:       "Fail-closed: Role tidak dikenal -> False",
			ps:         ps,
			role:       "hacker",
			permission: "user:list",
			expected:   false,
		},
		{
			name:       "Fail-closed: Permission tidak terdaftar -> False",
			ps:         ps,
			role:       "admin",
			permission: "system:shutdown",
			expected:   false,
		},
		{
			name:       "Fail-closed: Role string kosong -> False",
			ps:         ps,
			role:       "",
			permission: "user:list",
			expected:   false,
		},
		{
			name:       "Fail-closed: Permission string kosong -> False",
			ps:         ps,
			role:       "admin",
			permission: "",
			expected:   false,
		},
		{
			name:       "Fail-closed: PermissionSet adalah nil -> False",
			ps:         nil,
			role:       "admin",
			permission: "user:list",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ps.Can(tt.role, tt.permission)
			if got != tt.expected {
				t.Errorf("Can(%q, %q) = %v; want %v", tt.role, tt.permission, got, tt.expected)
			}
		})
	}
}
