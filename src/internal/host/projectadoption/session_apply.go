package projectadoption

import (
	"errors"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
)

// ApplyAndRecordSessionAdoption derives the plan from immutable session stages,
// requires the reviewed plan digest, runs the guarded model-only Apply path,
// then records only the receipt returned by that Apply.
func ApplyAndRecordSessionAdoption(sourceRoot, targetRoot string, target *projectwork.Project, session BrownfieldSession, iterationID, expectedPlanDigest, schemaDigest, buildDigest string) (BrownfieldSession, AdoptionPlan, AdoptionReceipt, error) {
	if err := ValidateBrownfieldSession(session); err != nil {
		return BrownfieldSession{}, AdoptionPlan{}, AdoptionReceipt{}, err
	}
	if target == nil || target.Digest != session.Target.ProjectDigest || target.Revision != session.Target.Revision || !sameFilesystemPath(target.Root, session.Target.Root) {
		return BrownfieldSession{}, AdoptionPlan{}, AdoptionReceipt{}, errors.New("adoption target differs from the session's fixed target basis")
	}
	plan, err := PlanSessionAdoption(sourceRoot, target, session, iterationID, schemaDigest, buildDigest)
	if err != nil {
		return BrownfieldSession{}, AdoptionPlan{}, AdoptionReceipt{}, err
	}
	if expectedPlanDigest == "" || expectedPlanDigest != plan.PlanDigest {
		return BrownfieldSession{}, AdoptionPlan{}, AdoptionReceipt{}, errors.New("apply requires the exact reviewed adoption plan digest")
	}
	iteration, exists := findIteration(session, iterationID)
	if !exists || iteration.Integration == nil || iteration.Resolution == nil {
		return BrownfieldSession{}, AdoptionPlan{}, AdoptionReceipt{}, errors.New("adoption requires an integrated and owner-resolved iteration")
	}
	receipt, err := ApplyAdoption(sourceRoot, targetRoot, target, session.Source, iteration.Integration.Report, *iteration.Resolution, plan, expectedPlanDigest, schemaDigest, buildDigest)
	if err != nil {
		return BrownfieldSession{}, plan, AdoptionReceipt{}, err
	}
	updated, err := recordSessionAdoption(sourceRoot, session, iterationID, plan, receipt)
	if err != nil {
		return BrownfieldSession{}, plan, receipt, err
	}
	return updated, plan, receipt, nil
}
