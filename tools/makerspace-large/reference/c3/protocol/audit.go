package protocol

import "makerspace/policies"

type auditEntry struct {
	seq    int
	at     int
	actor  string
	op     string
	target string
}

func auditView(e *auditEntry) policies.Rec {
	return policies.Rec{"seq": e.seq, "at": policies.FormatTime(e.at), "actor": e.actor, "op": e.op,
		"target": e.target}
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
