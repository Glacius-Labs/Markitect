package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// Actions and queued consultations belong to one provider response. A parent
// answering another question cannot accidentally alter a waiting review.
type assignmentCall struct {
	mu        sync.Mutex
	ctx       context.Context
	actions   []assignmentAction
	questions []*childQuestion
	refusals  []string
}
type assignmentCallKey struct{}
type assignmentMessageStackKey struct{}
type assignmentAnswerKey struct{}

const assignmentDiagnosticBytes = 1024
const assignmentRefusalEntries = 8
const omittedAssignmentRefusals = "[additional tool refusals omitted]"

// Only Host diagnostics may be shortened. Task, correction and answer content
// must reach the provider intact or fail through its context-limit contract.
func assignmentDiagnostic(text string) string {
	const suffix = " [diagnostic truncated]"
	if len(text) <= assignmentDiagnosticBytes {
		return text
	}
	return strings.ToValidUTF8(text[:assignmentDiagnosticBytes-len(suffix)], "") + suffix
}

type assignmentMailbox struct {
	mu        sync.Mutex
	path      string
	scheduler *assignmentScheduler
	pending   []*childQuestion
	running   bool
}

type childReply struct {
	text string
	err  error
}
type childQuestion struct {
	ctx                     context.Context
	parent, child, question string
	done                    chan childReply
	cancel                  context.CancelFunc
	stop                    func() bool
}

func (m *assignmentMailbox) handle(ctx context.Context, name string, raw json.RawMessage) ToolAnswer {
	call, _ := ctx.Value(assignmentCallKey{}).(*assignmentCall)
	refuse := func(reason string) ToolAnswer {
		reason = assignmentDiagnostic(reason)
		if call != nil {
			call.mu.Lock()
			if len(call.refusals) < assignmentRefusalEntries {
				call.refusals = append(call.refusals, reason)
			} else {
				call.refusals[assignmentRefusalEntries-1] = omittedAssignmentRefusals
			}
			call.mu.Unlock()
		}
		return ToolAnswer{Text: reason, Refused: true}
	}
	if call == nil {
		return refuse("no active assignment message")
	}
	if err := ctx.Err(); err != nil {
		return refuse(err.Error())
	}
	var args struct {
		Child      string `json:"child"`
		Task       string `json:"task"`
		Commit     string `json:"commit"`
		Correction string `json:"correction"`
		Question   string `json:"question"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&args); err != nil {
		return refuse("invalid tool arguments: " + err.Error())
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return refuse("tool arguments must be one JSON object")
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	allowed := map[string]bool{"child": true}
	switch name {
	case DelegateChildTool:
		allowed["task"] = true
	case ReturnChildTool:
		allowed["correction"] = true
	case AskChildTool:
		allowed["question"] = true
	case AcceptChildTool:
		if _, present := fields["commit"]; present {
			return refuse("accept_child takes only child; omit commit. The Host uses the exact fixed handoff currently under review")
		}
	}
	for field := range fields {
		if !allowed[field] {
			return refuse(fmt.Sprintf("%s does not take argument %q", name, field))
		}
	}
	if args.Child == "" || m.scheduler.team[args.Child].Parent != m.path || args.Child == m.path {
		return refuse("name one of your direct children: " + strings.Join(m.scheduler.team[m.path].Children, ", "))
	}
	if answerOnly, _ := ctx.Value(assignmentAnswerKey{}).(bool); answerOnly && name != AskChildTool {
		return refuse("answer this question without new work decisions")
	}
	action := assignmentAction{name: name, child: args.Child, task: strings.TrimSpace(args.Task), correction: strings.TrimSpace(args.Correction)}
	switch name {
	case AskChildTool:
		if strings.TrimSpace(args.Question) == "" || args.Task != "" || args.Commit != "" || args.Correction != "" {
			return refuse("ask_child needs only child and a non-empty question")
		}
		call.mu.Lock()
		delegatedThisResponse := false
		for _, pending := range call.actions {
			if pending.name == DelegateChildTool && pending.child == args.Child {
				delegatedThisResponse = true
				break
			}
		}
		if len(call.questions) >= m.scheduler.options.MaxMessages {
			call.mu.Unlock()
			return refuse("queued questions are limited by the assignment message allowance")
		}
		call.mu.Unlock()
		if delegatedThisResponse {
			return refuse("child " + args.Child + " starts only after you finish this response with Go. Complete shared interfaces and every required delegate_child task, then end this response; ask_child can follow after the Host starts the children")
		}
		question, err := m.scheduler.enqueueQuestion(call.ctx, m.path, args.Child, strings.TrimSpace(args.Question))
		if err != nil {
			return refuse(err.Error())
		}
		call.mu.Lock()
		call.questions = append(call.questions, question)
		call.mu.Unlock()
		return ToolAnswer{Text: "Question queued for " + args.Child + ". End this response when ready; its answer will be delivered in a follow-up before your decision is applied."}
	case DelegateChildTool:
		if action.task == "" || args.Commit != "" || args.Correction != "" || args.Question != "" {
			return refuse("delegate_child needs only child and a concrete task")
		}
	case AcceptChildTool:
		if args.Commit != "" || args.Task != "" || args.Correction != "" || args.Question != "" {
			return refuse("accept_child takes only child; omit commit. The Host uses the exact fixed handoff currently under review")
		}
	case ReturnChildTool:
		if action.correction == "" || args.Task != "" || args.Commit != "" || args.Question != "" {
			return refuse("return_child needs only child and a concrete correction")
		}
	default:
		return refuse("unknown assignment tool")
	}
	call.mu.Lock()
	call.actions = append(call.actions, action)
	call.mu.Unlock()
	if name == DelegateChildTool {
		return ToolAnswer{Text: "Child task recorded for this response. The Host starts all direct children together only after you finish this response with Go. Complete every required delegation and end this response; do not wait for or consult children yet."}
	}
	return ToolAnswer{Text: "Decision received for this response; the Host will validate and apply it."}
}

type assignmentResponse struct {
	observation TurnResult
	actions     []assignmentAction
	questions   []*childQuestion
	refusals    []string
}
type assignmentDecisionError struct{ reason string }

func (e *assignmentDecisionError) Error() string { return e.reason }
func invalidDecision(format string, args ...any) error {
	return &assignmentDecisionError{fmt.Sprintf(format, args...)}
}

// interact releases the Manager queue before awaiting child answers. The
// follow-up continues the same task/review. A valid review decision and Git
// integration still happen together under the Manager's serialization.
func (s *assignmentScheduler) interact(ctx context.Context, path, prompt string, decide func(context.Context, TurnResult, []assignmentAction) error) (TurnResult, error) {
	contextPrompt := prompt
	feedback := 0
	for {
		var response assignmentResponse
		err := s.flow.serial(ctx, path, func(locked context.Context) error {
			var err error
			response, err = s.call(locked, path, prompt)
			if err != nil || len(response.questions) > 0 {
				return err
			}
			if decide != nil {
				return decide(locked, response.observation, response.actions)
			}
			if len(response.actions) != 0 {
				return invalidDecision("this answer needs no new work decision")
			}
			return nil
		})
		if err != nil {
			for _, question := range response.questions {
				question.cancel()
			}
			var decision *assignmentDecisionError
			if !errors.As(err, &decision) {
				return response.observation, err
			}
			if feedback >= 2 {
				return response.observation, fmt.Errorf("Manager %q did not correct its decision after two feedback messages: %w", path, err)
			}
			feedback++
			prompt = contextPrompt + "\nHost decision feedback: " + assignmentDiagnostic(err.Error())
			if len(response.refusals) > 0 {
				prompt += "\nTool feedback: " + strings.Join(response.refusals, "; ")
			}
			prompt += "\nCorrect this in your same assignment using the offered tools. The original message allowance and timeout still apply."
			continue
		}
		if len(response.questions) > 0 {
			replies, err := s.waitReplies(ctx, response.questions)
			if err != nil {
				return response.observation, err
			}
			contextPrompt += replies
			prompt = contextPrompt + "\nContinue the original task or review with these replies. No provisional decision was applied while waiting; make your decision now."
			continue
		}
		return response.observation, nil
	}
}

func (s *assignmentScheduler) waitReplies(ctx context.Context, questions []*childQuestion) (string, error) {
	defer func() {
		for _, question := range questions {
			question.cancel()
		}
	}()
	var replies strings.Builder
	for _, question := range questions {
		select {
		case reply := <-question.done:
			if reply.err != nil {
				return "", fmt.Errorf("queued answer from %q: %w", question.child, reply.err)
			}
			fmt.Fprintf(&replies, "\nQueued child answer from %s to %q: %s", question.child, question.question, reply.text)
		case <-ctx.Done():
			return "", ctx.Err()
		case <-s.questionsCtx.Done():
			return "", s.questionsCtx.Err()
		}
	}
	return replies.String(), nil
}

func (s *assignmentScheduler) call(ctx context.Context, path, prompt string) (assignmentResponse, error) {
	stack, _ := ctx.Value(assignmentMessageStackKey{}).([]string)
	for _, active := range stack {
		if active == path {
			return assignmentResponse{}, fmt.Errorf("Manager %q is already answering in this question chain", path)
		}
	}
	if err := s.reserveMessage(); err != nil {
		return assignmentResponse{}, err
	}
	ctx = context.WithValue(ctx, assignmentMessageStackKey{}, append(append([]string(nil), stack...), path))
	inbox := &assignmentCall{ctx: ctx}
	ctx = context.WithValue(ctx, assignmentCallKey{}, inbox)
	answer, err := s.flow.Message(ctx, path, prompt, s.options.Timeout)
	inbox.mu.Lock()
	response := assignmentResponse{answer, append([]assignmentAction(nil), inbox.actions...), append([]*childQuestion(nil), inbox.questions...), append([]string(nil), inbox.refusals...)}
	inbox.mu.Unlock()
	if err != nil {
		return response, err
	}
	s.mu.Lock()
	problem := s.problem != nil
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return response, errors.New("assignment tree is closing")
	}
	if problem {
		return response, ErrAssignmentModelProblem
	}
	if !answer.Completed {
		return response, fmt.Errorf("Manager %q did not complete its response", path)
	}
	return response, nil
}

func (s *assignmentScheduler) messageResult(ctx context.Context, path, prompt string) (TurnResult, error) {
	return s.interact(ctx, path, prompt, nil)
}
func (s *assignmentScheduler) parentAnswer(ctx context.Context, parent, child, question string) (string, error) {
	answer, err := s.messageResult(context.WithValue(ctx, assignmentAnswerKey{}, true), parent, fmt.Sprintf("Your direct child %s asks: %s", child, question))
	return answer.Final, err
}

func (s *assignmentScheduler) enqueueQuestion(ctx context.Context, parent, child, question string) (*childQuestion, error) {
	stack, _ := ctx.Value(assignmentMessageStackKey{}).([]string)
	for _, active := range stack {
		if active == child {
			return nil, fmt.Errorf("Manager %q is already answering in this question chain; answer its pending question first", child)
		}
	}
	s.flow.mu.Lock()
	assigned := s.flow.team[child].assigned
	s.flow.mu.Unlock()
	if !assigned {
		return nil, fmt.Errorf("child %q has not received its first task; delegate its task and finish that response with Go before consulting it", child)
	}
	// Lock ownership is local to the current serialized call. It must never
	// cross into the new goroutine or survive the provider message boundary.
	ctx = context.WithValue(ctx, assignmentLockKey{}, []*assignmentMember(nil))
	ctx = context.WithValue(ctx, assignmentCallKey{}, (*assignmentCall)(nil))
	ctx, cancel := context.WithCancel(ctx)
	request := &childQuestion{ctx: context.WithValue(ctx, assignmentAnswerKey{}, true), parent: parent, child: child, question: question, done: make(chan childReply, 1), cancel: cancel}
	request.stop = context.AfterFunc(s.questionsCtx, cancel)
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		request.stop()
		cancel()
		return nil, errors.New("assignment tree is closing")
	}
	mailbox := s.mailboxes[child]
	mailbox.mu.Lock()
	mailbox.pending = append(mailbox.pending, request)
	start := !mailbox.running
	if start {
		mailbox.running = true
		s.questionWorkers.Add(1)
	}
	mailbox.mu.Unlock()
	s.mu.Unlock()
	if start {
		go s.deliverQuestions(mailbox)
	}
	return request, nil
}

func (s *assignmentScheduler) deliverQuestions(mailbox *assignmentMailbox) {
	defer s.questionWorkers.Done()
	for {
		mailbox.mu.Lock()
		if len(mailbox.pending) == 0 {
			mailbox.running = false
			mailbox.mu.Unlock()
			return
		}
		question := mailbox.pending[0]
		mailbox.pending = mailbox.pending[1:]
		mailbox.mu.Unlock()
		answer, err := s.messageResult(question.ctx, question.child, fmt.Sprintf("Your parent %s asks: %s. Answer this queued consultation without changing work or making new work decisions.", question.parent, question.question))
		question.stop()
		question.cancel()
		question.done <- childReply{answer.Final, err}
	}
}

func (s *assignmentScheduler) waitQuestions() error {
	done := make(chan struct{})
	go func() { s.questionWorkers.Wait(); close(done) }()
	timer := time.NewTimer(min(s.options.Timeout, time.Minute))
	defer timer.Stop()
	select {
	case <-done:
		return nil
	case <-timer.C:
		return errors.New("queued question cleanup remains uncertain")
	}
}
