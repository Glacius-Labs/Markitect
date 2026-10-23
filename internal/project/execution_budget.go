package project

func (d *executionDecoder) budget(field executionField) ExecutionBudget {
	r := ExecutionBudget{}
	if field.node == nil {
		return r
	}
	f := d.object(field, "executionTime", "turnTimeout", "agentTurns", "passes")
	r.ExecutionTime, _ = d.duration(d.required(f, field, "executionTime"))
	r.TurnTimeout, _ = d.duration(d.required(f, field, "turnTimeout"))
	r.AgentTurns, _ = d.count(d.required(f, field, "agentTurns"))
	r.Passes, _ = d.count(d.required(f, field, "passes"))
	return r
}
