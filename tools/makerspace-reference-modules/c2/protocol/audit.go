package protocol

import "makerspace/policies"

type auditEntry struct {
	seq       int
	at        int
	actor     string
	actorRole string
	op        string
	target    string
	outcome   string // accepted or refused
}

func auditView(e *auditEntry) policies.Rec {
	var target any = e.target
	if e.outcome == "refused" {
		target = nil
	}
	return policies.Rec{"seq": e.seq, "at": policies.FormatTime(e.at), "actor": e.actor, "actorRole": e.actorRole,
		"op": e.op, "target": target, "outcome": e.outcome}
}

// auditForbidden records a refusal for missing permission (section 4.2). It
// does not move the request time.
func (sv *Server) auditForbidden(c *policies.Call, op *policies.Operation) {
	sv.audit = append(sv.audit, &auditEntry{seq: len(sv.audit) + 1, at: c.At, actor: c.ActorID,
		actorRole: c.ActorRole, op: op.Name, outcome: "refused"})
}

// auditOperations lists the audit operation: the log belongs to the protocol
// module, which appends every entry.
func (sv *Server) auditOperations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "listAuditEntries", Perms: "nna",
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				out := make([]policies.Rec, 0, len(sv.audit))
				for _, e := range sv.audit {
					out = append(out, auditView(e))
				}
				return policies.Rec{"entries": out}, "", nil
			}},
	}
}
