"""Pure finite-deadline arithmetic; does not launch or authorize anything."""
import math
def role_deadline(*,now,task_start,trial_end,grant_end,runner_seconds,bridge_end,role_seconds,assessment=False):
    if any(type(v) not in (int,float) for v in [now,task_start,trial_end,grant_end,runner_seconds,bridge_end,role_seconds]):raise ValueError('explicit numeric times required')
    if not all(math.isfinite(v) for v in [now,task_start,trial_end,grant_end,runner_seconds,bridge_end,role_seconds]):raise ValueError('finite times required')
    if now<task_start:raise ValueError('task not started')
    if runner_seconds<=0 or role_seconds<=0:raise ValueError('finite positive role/runner seconds required')
    task_end=min(task_start+1200,trial_end,grant_end)
    phase_end=task_end-30 if assessment else min(task_start+840,task_end-360)
    effective=min(now+runner_seconds,now+role_seconds,bridge_end,phase_end)
    if assessment and task_end-now<300:raise ValueError('assessment window unavailable')
    if effective-now<=15:raise ValueError('no bounded launch window')
    return {'effectiveEnd':effective,'interruptRequestBy':effective-10,'taskEnd':task_end,'assessmentReservedFrom':task_start+840,'grantNotExtended':True}
