package billing

import "makerspace/policies"

// overdueBlockMinutes is the grace period after the due date: 30 days.
const overdueBlockMinutes = 30 * policies.MinPerDay

// Blocked reports whether the member has an open invoice more than 30 days
// past its due date at the given time (section 11.1).
func (b *Module) Blocked(memberID string, at int) bool {
	for _, i := range b.invoiceList {
		if i.MemberID == memberID && i.Status() == "open" && at > i.DueAt+overdueBlockMinutes {
			return true
		}
	}
	return false
}
