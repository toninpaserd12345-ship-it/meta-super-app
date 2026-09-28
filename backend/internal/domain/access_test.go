package domain

import "testing"

func TestMembershipCan(t *testing.T) {
	tests := []struct {
		name     string
		claims   []Permission
		required Permission
		want     bool
	}{
		{name: "exact claim", claims: []Permission{"orders:read"}, required: "orders:read", want: true},
		{name: "wildcard", claims: []Permission{ClaimAll}, required: "users:remove", want: true},
		{name: "missing claim", claims: []Permission{"orders:read"}, required: "orders:delete", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := (Membership{Claims: tt.claims}).Can(tt.required); got != tt.want {
				t.Fatalf("Can() = %v, want %v", got, tt.want)
			}
		})
	}
}
