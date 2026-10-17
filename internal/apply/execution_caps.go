package apply

func (d *executionDecoder) limits(field executionField) ExecutionLimits {
	r := ExecutionLimits{}
	if field.node == nil {
		return r
	}
	f := d.object(field, "maxRuns", "maxSessionStarts", "maxSteps", "maxExecutionTime", "maxStepTimeout", "hostOperationTimeout", "maxReviewRounds", "maxContextBytes", "maxSessionSteps", "maxTransientRetries", "creationRoots")
	r.MaxRuns, _ = d.integer(d.required(f, field, "maxRuns"))
	r.MaxSessionStarts, _ = d.integer(d.required(f, field, "maxSessionStarts"))
	r.MaxSteps, _ = d.integer(d.required(f, field, "maxSteps"))
	r.MaxExecutionTime, _ = d.duration(d.required(f, field, "maxExecutionTime"))
	r.MaxStepTimeout, _ = d.duration(d.required(f, field, "maxStepTimeout"))
	r.HostOperationTimeout, _ = d.duration(d.required(f, field, "hostOperationTimeout"))
	r.MaxReviewRounds, _ = d.count(d.required(f, field, "maxReviewRounds"))
	r.MaxContextBytes, _ = d.integer(d.required(f, field, "maxContextBytes"))
	r.MaxSessionSteps, _ = d.integer(d.required(f, field, "maxSessionSteps"))
	r.MaxTransientRetries, _ = d.integer(d.required(f, field, "maxTransientRetries"))
	r.CreationRoots, _ = d.directories(d.required(f, field, "creationRoots"))
	return r
}

func (d *executionDecoder) assignment(field executionField) AssignmentDefaults {
	r := AssignmentDefaults{}
	if field.node == nil {
		return r
	}
	f := d.object(field, "runs", "sessionStarts", "steps", "executionTime")
	r.Runs, _ = d.integer(d.required(f, field, "runs"))
	r.SessionStarts, _ = d.integer(d.required(f, field, "sessionStarts"))
	r.Steps, _ = d.integer(d.required(f, field, "steps"))
	r.ExecutionTime, _ = d.duration(d.required(f, field, "executionTime"))
	return r
}

func (d *executionDecoder) reserve(field executionField) CompletionReserveDeclaration {
	r := CompletionReserveDeclaration{}
	if field.node == nil {
		return r
	}
	f := d.object(field, "sessionStarts", "steps", "executionTime")
	r.SessionStarts, _ = d.integer(d.required(f, field, "sessionStarts"))
	r.Steps, _ = d.integer(d.required(f, field, "steps"))
	r.ExecutionTime, _ = d.duration(d.required(f, field, "executionTime"))
	return r
}
