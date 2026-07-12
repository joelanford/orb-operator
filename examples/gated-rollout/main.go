package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

const (
	codName   = "gated-rollout"
	namespace = "default"

	gateOpen     = "has(self.data) && has(self.data.gate) && self.data.gate == 'open'"
	gateV2       = "has(self.data) && has(self.data.gate) && self.data.gate == 'v2-open'"
	upgradeCheck = "has(self.data) && has(self.data.upgrade) && self.data.upgrade == 'done'"
)

var (
	bold   = lipgloss.NewStyle().Bold(true)
	dim    = lipgloss.NewStyle().Faint(true)
	green  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	yellow = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	red    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	cyan   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
)

func main() {
	s := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(orbv1alpha1.AddToScheme(s))

	cfg, err := ctrl.GetConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting kubeconfig: %v\n", err)
		os.Exit(1)
	}
	c, err := client.New(cfg, client.Options{Scheme: s})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	fmt.Print("Cleaning up previous run...")
	if err := cleanup(context.Background(), c); err != nil {
		fmt.Fprintf(os.Stderr, "\nError during cleanup: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(" done")

	m := newModel(c)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func cleanup(ctx context.Context, c client.Client) error {
	cod := &orbv1alpha1.ClusterObjectDeployment{}
	cod.Name = codName
	if err := client.IgnoreNotFound(c.Delete(ctx, cod)); err != nil {
		return err
	}
	for {
		var list orbv1alpha1.ClusterObjectSetList
		if err := c.List(ctx, &list); err != nil {
			return err
		}
		found := false
		for i := range list.Items {
			if list.Items[i].Spec.Group == codName {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// ── Bubbletea model ─────────────────────────────────────────────────────────

type step struct {
	description string
	action      func(ctx context.Context, c client.Client) error
	waitFor     func(m *model) bool
	waitMsg     string
}

type model struct {
	client client.Client

	steps       []step
	currentStep int

	cos *orbv1alpha1.ClusterObjectSet
	cod *orbv1alpha1.ClusterObjectDeployment

	waiting   bool
	executing bool
	err       error
	done      bool
}

func newModel(c client.Client) model {
	return model{
		client: c,
		steps:  buildSteps(c),
	}
}

type tickMsg time.Time

type refreshMsg struct {
	cos *orbv1alpha1.ClusterObjectSet
	cod *orbv1alpha1.ClusterObjectDeployment
	err error
}

type stepDoneMsg struct{ err error }

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func refreshCmd(c client.Client) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		cod := &orbv1alpha1.ClusterObjectDeployment{}
		if err := c.Get(ctx, types.NamespacedName{Name: codName}, cod); err != nil {
			if !errors.IsNotFound(err) {
				return refreshMsg{err: err}
			}
			cod = nil
		}

		var cos *orbv1alpha1.ClusterObjectSet
		var list orbv1alpha1.ClusterObjectSetList
		if err := c.List(ctx, &list); err != nil {
			return refreshMsg{err: err}
		}
		for i := range list.Items {
			item := &list.Items[i]
			if item.Spec.Group != codName {
				continue
			}
			if item.Spec.LifecycleState == orbv1alpha1.LifecycleStateArchived {
				continue
			}
			if cos == nil || item.Spec.Revision > cos.Spec.Revision {
				cos = item
			}
		}
		return refreshMsg{cos: cos, cod: cod}
	}
}

func executeStepCmd(c client.Client, s step) tea.Cmd {
	return func() tea.Msg {
		return stepDoneMsg{err: s.action(context.Background(), c)}
	}
}

func (m model) Init() tea.Cmd { return tickCmd() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.err != nil {
				m.err = nil
				m.executing = true
				return m, executeStepCmd(m.client, m.steps[m.currentStep])
			}
			if !m.waiting && !m.executing && !m.done && m.currentStep < len(m.steps) {
				m.executing = true
				return m, executeStepCmd(m.client, m.steps[m.currentStep])
			}
		}

	case stepDoneMsg:
		m.executing = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		if m.steps[m.currentStep].waitFor == nil {
			m.currentStep++
			if m.currentStep >= len(m.steps) {
				m.done = true
			}
		} else {
			m.waiting = true
		}
		return m, refreshCmd(m.client)

	case refreshMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.cos = msg.cos
			m.cod = msg.cod
		}
		if m.waiting && m.currentStep < len(m.steps) {
			if wf := m.steps[m.currentStep].waitFor; wf != nil && wf(&m) {
				m.waiting = false
				m.currentStep++
				if m.currentStep >= len(m.steps) {
					m.done = true
				}
			}
		}
		return m, tickCmd()

	case tickMsg:
		return m, refreshCmd(m.client)
	}
	return m, nil
}

// ── View ────────────────────────────────────────────────────────────────────

func (m model) View() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	if m.cos != nil {
		b.WriteString(m.viewPhaseTable())
		b.WriteString(m.viewObjectDetails())
	}
	b.WriteString("\n")
	b.WriteString(m.viewPrompt())

	return b.String()
}

func (m model) viewHeader() string {
	var b strings.Builder

	now := time.Now().Format("15:04:05")
	b.WriteString(bold.Render(fmt.Sprintf("  Gated Rollout Demo%s%s", strings.Repeat(" ", 42), now)))
	b.WriteString("\n\n")

	if m.cod != nil {
		b.WriteString(m.viewCODStatus())
	}
	b.WriteString("\n")

	if m.cos == nil {
		b.WriteString(dim.Render("  No COS yet"))
		b.WriteString("\n\n")
		return b.String()
	}

	availStatus, availReason := conditionInfo(m.cos.Status.Conditions, orbv1alpha1.ConditionTypeAvailable)
	b.WriteString(fmt.Sprintf("  COS: %s | Rev %d | Available: %s (%s)",
		bold.Render(m.cos.Name), m.cos.Spec.Revision, colorCondition(availStatus), availReason))
	b.WriteString("\n")

	completedAt := "not set"
	if m.cos.Status.CompletedAt != nil {
		completedAt = m.cos.Status.CompletedAt.Time.Format("15:04:05")
	}
	b.WriteString(fmt.Sprintf("  completedAt: %s", completedAt))
	b.WriteString("\n\n")

	return b.String()
}

func (m model) viewPhaseTable() string {
	var b strings.Builder
	phases := m.cos.Status.ObservedPhases
	if len(phases) == 0 {
		b.WriteString(dim.Render("  No observed phases yet"))
		b.WriteString("\n")
		return b.String()
	}

	header := fmt.Sprintf("  %-12s %-24s %5s %6s %5s  %s", "PHASE", "STATUS", "TOTAL", "SYNCED", "AVAIL", "MESSAGE")
	sep := fmt.Sprintf("  %-12s %-24s %5s %6s %5s  %s", "─────", "──────", "─────", "──────", "─────", "───────")
	b.WriteString(bold.Render(header))
	b.WriteString("\n")
	b.WriteString(dim.Render(sep))
	b.WriteString("\n")

	var totalT, totalS, totalA int64
	for _, op := range phases {
		totalT += op.ObjectCounts.Total
		totalS += op.ObjectCounts.Synced
		totalA += op.ObjectCounts.Available

		rawStatus := string(op.Status)
		statusCell := colorStatus(rawStatus) + strings.Repeat(" ", max(0, 24-len(rawStatus)))
		msg := op.Message
		if len(msg) > 40 {
			msg = msg[:37] + "..."
		}
		b.WriteString(fmt.Sprintf("  %-12s %s %5d %6d %5d  %s",
			op.Name, statusCell, op.ObjectCounts.Total, op.ObjectCounts.Synced, op.ObjectCounts.Available, msg))
		b.WriteString("\n")
	}

	b.WriteString(dim.Render(fmt.Sprintf("  %-12s %-24s %5s %6s %5s", "", "", "─────", "──────", "─────")))
	b.WriteString("\n")
	totalsLabel := bold.Render("TOTALS") + strings.Repeat(" ", 18)
	b.WriteString(fmt.Sprintf("  %-12s %s %5d %6d %5d", "", totalsLabel, totalT, totalS, totalA))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewObjectDetails() string {
	var b strings.Builder
	hasDetails := false
	for _, op := range m.cos.Status.ObservedPhases {
		if len(op.ObjectDetails) == 0 {
			continue
		}
		if !hasDetails {
			b.WriteString("\n")
			b.WriteString(bold.Render("  Object details:"))
			b.WriteString("\n")
			hasDetails = true
		}
		b.WriteString(fmt.Sprintf("    %s (%s):\n", op.Name, colorStatus(string(op.Status))))
		for _, od := range op.ObjectDetails {
			label := objectLabel(od)
			msg := strings.Join(od.Messages, "; ")
			if strings.Contains(msg, "\n") {
				lines := strings.Split(msg, "\n")
				b.WriteString(fmt.Sprintf("      %s: %s\n", label, lines[0]))
				for _, line := range lines[1:] {
					b.WriteString(fmt.Sprintf("        %s\n", line))
				}
			} else {
				b.WriteString(fmt.Sprintf("      %s: %s\n", label, msg))
			}
		}
	}
	return b.String()
}

func (m model) viewPrompt() string {
	var b strings.Builder
	b.WriteString(dim.Render("  " + strings.Repeat("─", 64)))
	b.WriteString("\n")

	switch {
	case m.err != nil:
		b.WriteString(red.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString("\n")
		b.WriteString("  Press ENTER to retry, q to quit\n")
	case m.done:
		b.WriteString(green.Render("  Done! All steps completed."))
		b.WriteString("\n")
		b.WriteString(dim.Render(fmt.Sprintf("  Cleanup: kubectl delete clusterobjectdeployment %s", codName)))
		b.WriteString("\n")
		b.WriteString("  Press q to quit\n")
	case m.executing:
		b.WriteString(yellow.Render(fmt.Sprintf("  Step %d/%d: %s",
			m.currentStep+1, len(m.steps), m.steps[m.currentStep].description)))
		b.WriteString("\n")
		b.WriteString(dim.Render("  Executing..."))
		b.WriteString("\n")
	case m.waiting:
		b.WriteString(yellow.Render(fmt.Sprintf("  Step %d/%d: %s",
			m.currentStep+1, len(m.steps), m.steps[m.currentStep].description)))
		b.WriteString("\n")
		b.WriteString(dim.Render(fmt.Sprintf("  %s", m.steps[m.currentStep].waitMsg)))
		b.WriteString("\n")
	default:
		b.WriteString(cyan.Render(fmt.Sprintf("  Step %d/%d: %s",
			m.currentStep+1, len(m.steps), m.steps[m.currentStep].description)))
		b.WriteString("\n")
		b.WriteString("  Press ENTER to continue, q to quit\n")
	}
	return b.String()
}

func (m model) viewCODStatus() string {
	var b strings.Builder

	availStatus, availReason := conditionInfo(m.cod.Status.Conditions, orbv1alpha1.ConditionTypeAvailable)
	progStatus, progReason := conditionInfo(m.cod.Status.Conditions, orbv1alpha1.ConditionTypeProgressing)

	b.WriteString(fmt.Sprintf("  COD: %s | Available: %s (%s) | Progressing: %s (%s)",
		bold.Render(m.cod.Name), colorCondition(availStatus), availReason, colorCondition(progStatus), progReason))
	b.WriteString("\n")

	if m.cod.Spec.ProgressDeadlineMinutes != nil && m.cos != nil {
		cosComplete := m.cos.Status.CompletedAt != nil
		remaining := m.deadlineRemaining()
		secs := int(remaining.Seconds())
		var dl string
		switch {
		case cosComplete:
			dl = green.Render("n/a (rollout complete)")
		case progReason == orbv1alpha1.ReasonProgressDeadlineExceeded:
			dl = red.Render("EXCEEDED")
		case remaining > 15*time.Second:
			dl = green.Render(fmt.Sprintf("%ds remaining", secs))
		case remaining > 0:
			dl = yellow.Render(fmt.Sprintf("%ds remaining", secs))
		default:
			dl = red.Render(fmt.Sprintf("EXPIRED (%ds ago)", -secs))
		}
		b.WriteString(fmt.Sprintf("  Progress deadline: %s (resets on each phase completion)", dl))
		b.WriteString("\n")
	}

	return b.String()
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func colorCondition(status string) string {
	switch status {
	case "True":
		return green.Render(status)
	case "False":
		return yellow.Render(status)
	case "Unknown":
		return red.Render(status)
	default:
		return status
	}
}

func objectLabel(od orbv1alpha1.ObjectStatus) string {
	ns := od.Namespace
	if ns == "" {
		return fmt.Sprintf("%s %s", od.Kind, od.Name)
	}
	return fmt.Sprintf("%s %s/%s", od.Kind, ns, od.Name)
}

func colorStatus(s string) string {
	switch s {
	case "Available", "TeardownComplete":
		return green.Render(s)
	case "WaitingForAssertions", "Reconciling", "Pending":
		return yellow.Render(s)
	case "Invalid", "Unknown":
		return red.Render(s)
	case "Superseded", "TearingDown":
		return dim.Render(s)
	default:
		return s
	}
}

func conditionInfo(conditions []metav1.Condition, condType string) (string, string) {
	cond := meta.FindStatusCondition(conditions, condType)
	if cond == nil {
		return "Unknown", ""
	}
	return string(cond.Status), cond.Reason
}

func phaseStatus(cos *orbv1alpha1.ClusterObjectSet, name string) orbv1alpha1.PhaseStatus {
	for _, op := range cos.Status.ObservedPhases {
		if op.Name == name {
			return op.Status
		}
	}
	return ""
}

func (m model) deadlineRemaining() time.Duration {
	if m.cod == nil || m.cod.Spec.ProgressDeadlineMinutes == nil || m.cos == nil {
		return 0
	}
	deadline := time.Duration(*m.cod.Spec.ProgressDeadlineMinutes) * time.Minute
	lastProgress := m.cos.CreationTimestamp.Time
	for _, op := range m.cos.Status.ObservedPhases {
		if op.CompletedAt != nil && op.CompletedAt.Time.After(lastProgress) {
			lastProgress = op.CompletedAt.Time
		}
	}
	return deadline - time.Since(lastProgress)
}

// ── Steps ───────────────────────────────────────────────────────────────────

func buildSteps(c client.Client) []step {
	return []step{
		{
			description: "Create COD v1 (3 phases, 7 gated objects, progressDeadlineMinutes=1)",
			action: func(ctx context.Context, c client.Client) error {
				return c.Create(ctx, buildCODv1())
			},
			waitFor: func(m *model) bool {
				return m.cos != nil && phaseStatus(m.cos, "phase-1") == orbv1alpha1.PhaseStatusWaitingForAssertions
			},
			waitMsg: "Waiting for phase-1 = WaitingForAssertions...",
		},
		{
			description: "Open gate: cm-p1-o1",
			action:      patchAction("cm-p1-o1", "gate", "open"),
		},
		{
			description: "Open gate: cm-p1-o2",
			action:      patchAction("cm-p1-o2", "gate", "open"),
			waitFor: func(m *model) bool {
				return m.cos != nil && phaseStatus(m.cos, "phase-1") == orbv1alpha1.PhaseStatusAvailable
			},
			waitMsg: "Waiting for phase-1 = Available...",
		},
		{
			description: "Open gate: cm-p2-o1",
			action:      patchAction("cm-p2-o1", "gate", "open"),
		},
		{
			description: "Open gate: cm-p2-o2",
			action:      patchAction("cm-p2-o2", "gate", "open"),
			waitFor: func(m *model) bool {
				return m.cos != nil && phaseStatus(m.cos, "phase-2") == orbv1alpha1.PhaseStatusAvailable
			},
			waitMsg: "Waiting for phase-2 = Available...",
		},
		{
			description: "Open gate: cm-p3-o1",
			action:      patchAction("cm-p3-o1", "gate", "open"),
		},
		{
			description: "Open gate: cm-p3-o2",
			action:      patchAction("cm-p3-o2", "gate", "open"),
		},
		{
			description: "Open gate: cm-p3-o3",
			action:      patchAction("cm-p3-o3", "gate", "open"),
			waitFor: func(m *model) bool {
				if m.cos == nil {
					return false
				}
				_, reason := conditionInfo(m.cos.Status.Conditions, orbv1alpha1.ConditionTypeAvailable)
				return reason == orbv1alpha1.ReasonAvailable
			},
			waitMsg: "Waiting for COS Available = True...",
		},
		{
			description: "Apply upgrade (COD v2): phase-1 content+gate change, phase-2 unchanged, phase-3 mixed",
			action: func(ctx context.Context, c client.Client) error {
				cod := &orbv1alpha1.ClusterObjectDeployment{}
				if err := c.Get(ctx, types.NamespacedName{Name: codName}, cod); err != nil {
					return err
				}
				v2 := buildCODv2()
				cod.Spec = v2.Spec
				return c.Update(ctx, cod)
			},
			waitFor: func(m *model) bool {
				return m.cos != nil && m.cos.Spec.Revision == 2 &&
					phaseStatus(m.cos, "phase-1") == orbv1alpha1.PhaseStatusWaitingForAssertions
			},
			waitMsg: "Waiting for rev 2 phase-1 = WaitingForAssertions...",
		},
		{
			description: "Open gate: cm-p1-o1 (v2-open)",
			action:      patchAction("cm-p1-o1", "gate", "v2-open"),
		},
		{
			description: "Open gate: cm-p1-o2 (v2-open)",
			action:      patchAction("cm-p1-o2", "gate", "v2-open"),
			waitFor: func(m *model) bool {
				return m.cos != nil && phaseStatus(m.cos, "phase-1") == orbv1alpha1.PhaseStatusAvailable
			},
			waitMsg: "Waiting for phase-1 = Available...",
		},
		{
			description: "Open gate: cm-p3-o1 (v2-open)",
			action:      patchAction("cm-p3-o1", "gate", "v2-open"),
		},
		{
			description: "Satisfy assertion: cm-p3-o3 (upgrade=done)",
			action:      patchAction("cm-p3-o3", "upgrade", "done"),
			waitFor: func(m *model) bool {
				if m.cos == nil {
					return false
				}
				_, reason := conditionInfo(m.cos.Status.Conditions, orbv1alpha1.ConditionTypeAvailable)
				return reason == orbv1alpha1.ReasonAvailable
			},
			waitMsg: "Waiting for COS Available = True...",
		},
	}
}

func patchAction(cmName, key, value string) func(context.Context, client.Client) error {
	return func(ctx context.Context, c client.Client) error {
		cm := &corev1.ConfigMap{}
		if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: cmName}, cm); err != nil {
			return err
		}
		old := cm.DeepCopy()
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		cm.Data[key] = value
		return c.Patch(ctx, cm, client.MergeFrom(old))
	}
}

// ── COD builders ────────────────────────────────────────────────────────────

func buildCODv1() *orbv1alpha1.ClusterObjectDeployment {
	return &orbv1alpha1.ClusterObjectDeployment{
		ObjectMeta: metav1.ObjectMeta{Name: codName},
		Spec: orbv1alpha1.ClusterObjectDeploymentSpec{
			ProgressDeadlineMinutes: ptr(int32(1)),
			Template: orbv1alpha1.ClusterObjectDeploymentTemplate{
				Spec: orbv1alpha1.ClusterObjectDeploymentTemplateSpec{
					Phases: []orbv1alpha1.Phase{
						newPhase("phase-1",
							gatedCM("cm-p1-o1", "bar", gateOpen),
							gatedCM("cm-p1-o2", "bar", gateOpen),
						),
						newPhase("phase-2",
							gatedCM("cm-p2-o1", "bar", gateOpen),
							gatedCM("cm-p2-o2", "bar", gateOpen),
						),
						newPhase("phase-3",
							gatedCM("cm-p3-o1", "bar", gateOpen),
							gatedCM("cm-p3-o2", "bar", gateOpen),
							gatedCM("cm-p3-o3", "bar", gateOpen),
						),
					},
				},
			},
		},
	}
}

func buildCODv2() *orbv1alpha1.ClusterObjectDeployment {
	return &orbv1alpha1.ClusterObjectDeployment{
		ObjectMeta: metav1.ObjectMeta{Name: codName},
		Spec: orbv1alpha1.ClusterObjectDeploymentSpec{
			ProgressDeadlineMinutes: ptr(int32(1)),
			Template: orbv1alpha1.ClusterObjectDeploymentTemplate{
				Spec: orbv1alpha1.ClusterObjectDeploymentTemplateSpec{
					Phases: []orbv1alpha1.Phase{
						// Phase 1: content changed (foo: bar → baz), gate assertion → v2-open
						newPhase("phase-1",
							gatedCM("cm-p1-o1", "baz", gateV2),
							gatedCM("cm-p1-o2", "baz", gateV2),
						),
						// Phase 2: completely unchanged
						newPhase("phase-2",
							gatedCM("cm-p2-o1", "bar", gateOpen),
							gatedCM("cm-p2-o2", "bar", gateOpen),
						),
						// Phase 3: mixed changes
						newPhase("phase-3",
							gatedCM("cm-p3-o1", "baz", gateV2),                                   // content changed + gate v2
							gatedCM("cm-p3-o2", "bar", gateOpen),                                  // unchanged
							gatedCMExtra("cm-p3-o3", "bar", gateOpen, upgradeCheck, "upgrade is not done"), // same content, added assertion
						),
					},
				},
			},
		},
	}
}

func newPhase(name string, objects ...orbv1alpha1.PhaseObject) orbv1alpha1.Phase {
	return orbv1alpha1.Phase{Name: name, Objects: objects}
}

func gatedCM(name, fooValue, gateExpr string) orbv1alpha1.PhaseObject {
	return gatedCMExtra(name, fooValue, gateExpr, "", "")
}

func gatedCMExtra(name, fooValue, gateExpr, extraExpr, extraMsg string) orbv1alpha1.PhaseObject {
	raw, _ := json.Marshal(map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]any{"name": name, "namespace": namespace},
		"data":       map[string]any{"foo": fooValue},
	})

	assertions := []orbv1alpha1.Assertion{{
		CELExpression: &orbv1alpha1.CELExpressionAssertion{
			Expression: gateExpr,
			Message:    "gate is not open",
		},
	}}
	if extraExpr != "" {
		assertions = append(assertions, orbv1alpha1.Assertion{
			CELExpression: &orbv1alpha1.CELExpressionAssertion{
				Expression: extraExpr,
				Message:    extraMsg,
			},
		})
	}

	return orbv1alpha1.PhaseObject{
		Object:     runtime.RawExtension{Raw: raw},
		Assertions: assertions,
	}
}

func ptr[T any](v T) *T { return &v }
