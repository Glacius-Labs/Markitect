package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/host/government"
	"github.com/Glacius-Labs/Markitect/internal/host/government/execution"
	"github.com/Glacius-Labs/Markitect/internal/host/government/inventory"
)

// Government has a closed command contract. Execution requires a separate,
// trusted runtime and explicit write intent; reads retain their G1 contract.
func runGovernment(args []string, out, errout io.Writer) int {
	fs := flag.NewFlagSet("government", flag.ContinueOnError)
	fs.SetOutput(errout)
	repo := fs.String("repo", ".", "explicit native repository root")
	config := fs.String("config", "", "repository-relative trusted GovernmentSource YAML")
	orderPath := fs.String("order", "", "repository-relative Order YAML bound to prior Constitution")
	action := fs.String("action", "inspect", "schema, inspect, plan, run, queue or resume (experimental)")
	runtimePath := fs.String("runtime", "", "absolute external runtime JSON for run")
	backlogPath := fs.String("backlog", "", "absolute external finite backlog JSON for queue or resume")
	queuePath := fs.String("queue", "", "absolute existing queue directory for resume")
	write := fs.Bool("write", false, "explicitly execute and promote the scoped candidate")
	fs.Usage = func() {
		fmt.Fprintln(out, "usage: markitect government --repo PATH --config FILE --action inspect|plan [--order FILE]\n       markitect government --action schema\n       markitect government --repo PATH --config FILE --order FILE --action run --runtime ABS_JSON --write\n       markitect government --repo PATH --action queue --backlog ABS_JSON --write\n       markitect government --repo PATH --action resume --backlog ABS_JSON --queue ABS_DIR --write")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	fail := func(err error) int { fmt.Fprintln(errout, err); return 2 }
	emit := func(v any) int {
		data, err := host.YAML(v)
		if err != nil {
			return fail(err)
		}
		if _, err = out.Write(data); err != nil {
			return fail(err)
		}
		return 0
	}
	if fs.NArg() != 0 {
		return fail(errors.New("government accepts no positional arguments"))
	}
	if *action == "queue" || *action == "resume" {
		if !*write || !filepath.IsAbs(*backlogPath) || *runtimePath != "" || *config != "" || *orderPath != "" {
			return fail(errors.New("queue and resume require an absolute backlog JSON and --write; config, order and runtime belong to its jobs"))
		}
		if (*action == "queue" && *queuePath != "") || (*action == "resume" && !filepath.IsAbs(*queuePath)) {
			return fail(errors.New("queue creates a new queue; resume requires an absolute existing --queue directory"))
		}
		opts := execution.QueueOptions{Repo: *repo, BacklogPath: *backlogPath, QueueDirectory: *queuePath}
		var result execution.QueueReport
		var runErr error
		if *action == "queue" {
			result, runErr = execution.StartQueue(context.Background(), opts)
		} else {
			result, runErr = execution.ResumeQueue(context.Background(), opts)
		}
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			return fail(err)
		}
		if runErr != nil {
			fmt.Fprintln(errout, runErr)
			return 1
		}
		return 0
	}
	queueFlagPresent := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "backlog" || f.Name == "queue" {
			queueFlagPresent = true
		}
	})
	if queueFlagPresent {
		return fail(errors.New("backlog and queue are only valid for actions queue and resume"))
	}
	if *action == "run" {
		if !*write || *runtimePath == "" || !filepath.IsAbs(*runtimePath) || *config == "" || *orderPath == "" {
			return fail(errors.New("run requires config, order, an absolute runtime JSON path and --write"))
		}
		runtime, err := execution.ReadRuntime(*repo, *runtimePath)
		if err != nil {
			return fail(err)
		}
		result, runErr := execution.Run(context.Background(), execution.Options{Repo: *repo, ConfigPath: *config, OrderPath: *orderPath, Runtime: runtime})
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			return fail(err)
		}
		if runErr != nil {
			fmt.Fprintln(errout, runErr)
			return 1
		}
		return 0
	}
	executionFlagPresent := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "write" || f.Name == "runtime" {
			executionFlagPresent = true
		}
	})
	if executionFlagPresent {
		return fail(errors.New("write and runtime are only valid for action run"))
	}
	if *action == "schema" {
		if *config != "" || *orderPath != "" {
			return fail(errors.New("schema accepts neither config nor order"))
		}
		return emit(government.Schema())
	}
	if *action != "inspect" && *action != "plan" {
		return fail(errors.New("government action must be schema, inspect, plan, run, queue or resume"))
	}
	if *config == "" || (*action == "plan" && *orderPath == "") || (*action == "inspect" && *orderPath != "") {
		return fail(errors.New("inspect requires config; plan requires config and order"))
	}
	data, err := inventory.ReadInput(*repo, *config, 8<<20)
	if err != nil {
		return fail(err)
	}
	var source government.Source
	if err := government.Decode(data, &source); err != nil {
		return fail(fmt.Errorf("Government source: %w", err))
	}
	// Assign real provenance instead of trusting source fields supplied by YAML.
	for i := range source.Schemas {
		source.Schemas[i].Source.Path = *config
		source.Schemas[i].Source.Digest = government.BytesDigest(data)
	}
	for i := range source.Definitions {
		source.Definitions[i].Source.Path = *config
		source.Definitions[i].Source.Digest = government.BytesDigest(data)
	}
	m := government.Compile(source)
	if len(m.Findings) > 0 {
		if code := emit(m); code != 0 {
			return code
		}
		return 1
	}
	observation, err := inventory.Capture(*repo, source.Observation)
	if err != nil {
		return fail(err)
	}
	if err := capturedInput(observation, *config, data); err != nil {
		return fail(err)
	}
	if *action == "inspect" {
		survey := government.SurveyRepository(m, observation)
		result := struct {
			APIVersion string            `yaml:"apiVersion"`
			Model      government.Model  `yaml:"model"`
			Survey     government.Survey `yaml:"survey"`
		}{"markitect.government-inspection/v1alpha1", m, survey}
		if code := emit(result); code != 0 {
			return code
		}
		if survey.Coverage == "incomplete" {
			return 1
		}
		return 0
	}
	orderBytes, err := inventory.ReadInput(*repo, *orderPath, 8<<20)
	if err != nil {
		return fail(err)
	}
	if err := capturedInput(observation, *orderPath, orderBytes); err != nil {
		return fail(err)
	}
	var order government.Order
	if err := government.Decode(orderBytes, &order); err != nil {
		return fail(fmt.Errorf("Government order: %w", err))
	}
	p := government.BuildPlan(m, order, observation, *config)
	if code := emit(p); code != 0 {
		return code
	}
	if p.Status == "blocked" {
		return 1
	}
	return 0
}

func capturedInput(r inventory.Report, path string, data []byte) error {
	for _, e := range r.Entries {
		if e.Path == path && e.Status == "observed" && e.Digest == government.BytesDigest(data) {
			return nil
		}
	}
	return fmt.Errorf("Government input %q must be admitted, readable and unchanged in the declared native inventory", path)
}
