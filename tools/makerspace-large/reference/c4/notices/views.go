package notices

import "makerspace/policies"

func noticeView(n *Notice) policies.Rec {
	var ref any
	if n.Ref != nil {
		ref = *n.Ref
	}
	return policies.Rec{"noticeId": n.ID, "memberId": n.MemberID, "kind": n.Kind, "subject": n.Subject,
		"ref": ref, "at": policies.FormatTime(n.At), "acknowledgedAt": policies.OptTime(n.AcknowledgedAt)}
}
