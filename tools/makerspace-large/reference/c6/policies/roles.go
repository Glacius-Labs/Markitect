package policies

var roleIndex = map[string]int{"member": 0, "staff": 1, "admin": 2}

// RoleIndex is the position of a role in the permission string of an
// operation: one letter per role, in the order member, staff, admin.
func RoleIndex(role string) int { return roleIndex[role] }
