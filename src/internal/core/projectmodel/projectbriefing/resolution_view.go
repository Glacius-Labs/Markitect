package projectbriefing

// ResolutionView is a copied, process-only projection used to report one
// event's accepted resolution or delivered-but-uncommitted hint. It is not a
// persistence DTO and contains no briefing history or other store fields.
type ResolutionView struct {
	accepted map[string]Resolution
	hints    map[string]Resolution
}

// NewResolutionView copies only the first accepted resolution per event and
// the delivered-uncommitted hints supplied by Application. Accepted evidence
// always takes precedence over a hint for the same event.
func NewResolutionView(store Store, hints []Resolution) ResolutionView {
	view := ResolutionView{accepted: make(map[string]Resolution), hints: make(map[string]Resolution, len(hints))}
	for _, resolution := range store.Resolutions {
		if _, exists := view.accepted[resolution.EventID]; !exists {
			view.accepted[resolution.EventID] = cloneResolution(resolution)
		}
	}
	for _, resolution := range hints {
		if _, exists := view.hints[resolution.EventID]; !exists {
			view.hints[resolution.EventID] = cloneResolution(resolution)
		}
	}
	return view
}

// EventResolutionStatus reports accepted resolution separately from display-
// only delivery hints. A hint never means that an event is resolved.
func EventResolutionStatus(view ResolutionView, eventID string) ResolutionStatus {
	if resolution, ok := view.accepted[eventID]; ok {
		copy := cloneResolution(resolution)
		return ResolutionStatus{Status: StatusResolved, Resolution: &copy}
	}
	if resolution, ok := view.hints[eventID]; ok {
		copy := cloneResolution(resolution)
		return ResolutionStatus{Status: StatusDeliveredUncommitted, Resolution: &copy}
	}
	return ResolutionStatus{Status: StatusUnresolved}
}

func cloneResolution(resolution Resolution) Resolution {
	resolution.Evidence.EventIDs = cloneStrings(resolution.Evidence.EventIDs)
	resolution.Evidence.CoveredManagerIDs = cloneStrings(resolution.Evidence.CoveredManagerIDs)
	resolution.Evidence.EvidenceRefs = cloneStrings(resolution.Evidence.EvidenceRefs)
	if resolution.Evidence.Delivered != nil {
		resolution.Evidence.Delivered = append([]DeliveredFile{}, resolution.Evidence.Delivered...)
	}
	return resolution
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string{}, values...)
}
