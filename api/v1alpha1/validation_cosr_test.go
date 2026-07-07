package v1alpha1_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func TestCOSR_Status_CompletedAt_ImmutableOnceSet(t *testing.T) {
	ctx := context.Background()

	t.Run("setting completedAt on a new COSR succeeds", func(t *testing.T) {
		cosr := newCOSR("completed-at-set")
		createCOSR(t, ctx, cosr)

		now := metav1.Now()
		cosr.Status.CompletedAt = &now
		require.NoError(t, k8sClient.Status().Update(ctx, cosr))
	})

	t.Run("clearing completedAt after it has been set is rejected", func(t *testing.T) {
		cosr := newCOSR("completed-at-clear")
		createCOSR(t, ctx, cosr)

		now := metav1.Now()
		cosr.Status.CompletedAt = &now
		require.NoError(t, k8sClient.Status().Update(ctx, cosr))

		cosr.Status.CompletedAt = nil
		requireStatusError(t, k8sClient.Status().Update(ctx, cosr),
			"status", "completedAt is immutable once set")
	})

	t.Run("completedAt remains unset when never written", func(t *testing.T) {
		cosr := newCOSR("completed-at-unset")
		createCOSR(t, ctx, cosr)

		cosr.Status.Conditions = []metav1.Condition{{
			Type:               "Available",
			Status:             metav1.ConditionFalse,
			Reason:             "Testing",
			LastTransitionTime: metav1.Now(),
		}}
		require.NoError(t, k8sClient.Status().Update(ctx, cosr))
		assert.Nil(t, cosr.Status.CompletedAt)
	})
}

func TestCOSR_Status_ObservedPhase_Validation(t *testing.T) {
	ctx := context.Background()

	t.Run("valid observed phase succeeds", func(t *testing.T) {
		cosr := newCOSR("op-valid")
		createCOSR(t, ctx, cosr)

		cosr.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
			Name:   "install",
			Status: orbv1alpha1.PhaseStatusReconciling,
		}}
		require.NoError(t, k8sClient.Status().Update(ctx, cosr))
	})

	t.Run("empty phase name is rejected", func(t *testing.T) {
		cosr := newCOSR("op-empty-name")
		createCOSR(t, ctx, cosr)

		cosr.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
			Name:   "",
			Status: orbv1alpha1.PhaseStatusUnknown,
		}}
		requireStatusError(t, k8sClient.Status().Update(ctx, cosr),
			"status.observedPhases[0].name", "should be at least 1 chars long")
	})

	t.Run("phase name exceeding 63 chars is rejected", func(t *testing.T) {
		cosr := newCOSR("op-long-name")
		createCOSR(t, ctx, cosr)

		cosr.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
			Name:   strings.Repeat("a", 64),
			Status: orbv1alpha1.PhaseStatusUnknown,
		}}
		requireStatusError(t, k8sClient.Status().Update(ctx, cosr),
			"status.observedPhases[0].name", "may not be more than 63")
	})

	for _, status := range []orbv1alpha1.PhaseStatus{
		orbv1alpha1.PhaseStatusReconciling,
		orbv1alpha1.PhaseStatusAvailable,
		orbv1alpha1.PhaseStatusUnknown,
		orbv1alpha1.PhaseStatusSuperseded,
		orbv1alpha1.PhaseStatusTearingDown,
		orbv1alpha1.PhaseStatusTeardownComplete,
	} {
		t.Run(fmt.Sprintf("phase status %q is accepted", status), func(t *testing.T) {
			cosr := newCOSR(fmt.Sprintf("op-status-%s", strings.ToLower(string(status))))
			createCOSR(t, ctx, cosr)

			cosr.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
				Name:   "install",
				Status: status,
			}}
			require.NoError(t, k8sClient.Status().Update(ctx, cosr))
		})
	}

	t.Run("invalid phase status enum is rejected", func(t *testing.T) {
		cosr := newCOSR("op-bad-status")
		createCOSR(t, ctx, cosr)

		cosr.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
			Name:   "install",
			Status: orbv1alpha1.PhaseStatus("Invalid"),
		}}
		requireStatusError(t, k8sClient.Status().Update(ctx, cosr),
			"status.observedPhases[0].status", "Unsupported value")
	})

	t.Run("phase error at 1024 chars succeeds", func(t *testing.T) {
		cosr := newCOSR("op-max-err")
		createCOSR(t, ctx, cosr)

		cosr.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
			Name:   "install",
			Status: orbv1alpha1.PhaseStatusReconciling,
			Error:  strings.Repeat("e", 1024),
		}}
		require.NoError(t, k8sClient.Status().Update(ctx, cosr))
	})

	t.Run("phase error exceeding 1024 chars is rejected", func(t *testing.T) {
		cosr := newCOSR("op-long-err")
		createCOSR(t, ctx, cosr)

		cosr.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
			Name:   "install",
			Status: orbv1alpha1.PhaseStatusReconciling,
			Error:  strings.Repeat("e", 1025),
		}}
		requireStatusError(t, k8sClient.Status().Update(ctx, cosr),
			"status.observedPhases[0].error", "may not be more than 1024")
	})

	t.Run("observedPhases exceeding 20 is rejected", func(t *testing.T) {
		cosr := newCOSR("op-too-many-phases")
		createCOSR(t, ctx, cosr)

		phases := make([]orbv1alpha1.ObservedPhase, 21)
		for i := range phases {
			phases[i] = orbv1alpha1.ObservedPhase{
				Name:   fmt.Sprintf("phase-%d", i),
				Status: orbv1alpha1.PhaseStatusUnknown,
			}
		}
		cosr.Status.ObservedPhases = phases
		requireStatusError(t, k8sClient.Status().Update(ctx, cosr),
			"status.observedPhases", "must have at most 20 items")
	})

	t.Run("incompleteObjects at max (50) succeeds", func(t *testing.T) {
		cosr := newCOSR("op-max-objects")
		createCOSR(t, ctx, cosr)

		objects := make([]orbv1alpha1.ObjectStatus, 50)
		for i := range objects {
			objects[i] = orbv1alpha1.ObjectStatus{
				Version: "v1",
				Kind:    "ConfigMap",
				Name:    "cm",
			}
		}
		cosr.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
			Name:              "install",
			Status:            orbv1alpha1.PhaseStatusReconciling,
			IncompleteObjects: objects,
		}}
		require.NoError(t, k8sClient.Status().Update(ctx, cosr))
	})

	t.Run("incompleteObjects exceeding 50 is rejected", func(t *testing.T) {
		cosr := newCOSR("op-too-many-obj")
		createCOSR(t, ctx, cosr)

		objects := make([]orbv1alpha1.ObjectStatus, 51)
		for i := range objects {
			objects[i] = orbv1alpha1.ObjectStatus{
				Version: "v1",
				Kind:    "ConfigMap",
				Name:    "cm",
			}
		}
		cosr.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
			Name:              "install",
			Status:            orbv1alpha1.PhaseStatusReconciling,
			IncompleteObjects: objects,
		}}
		requireStatusError(t, k8sClient.Status().Update(ctx, cosr),
			"status.observedPhases[0].incompleteObjects", "must have at most 50 items")
	})
}

func TestCOSR_Status_ObjectStatus_Validation(t *testing.T) {
	ctx := context.Background()

	setObjectStatus := func(t *testing.T, name string, os orbv1alpha1.ObjectStatus) error {
		t.Helper()
		cosr := newCOSR(name)
		createCOSR(t, ctx, cosr)
		cosr.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
			Name:              "install",
			Status:            orbv1alpha1.PhaseStatusReconciling,
			IncompleteObjects: []orbv1alpha1.ObjectStatus{os},
		}}
		return k8sClient.Status().Update(ctx, cosr)
	}

	objField := func(f string) string {
		return "status.observedPhases[0].incompleteObjects[0]." + f
	}

	t.Run("valid object status succeeds", func(t *testing.T) {
		require.NoError(t, setObjectStatus(t, "os-valid", orbv1alpha1.ObjectStatus{
			Group:     "apps",
			Version:   "v1",
			Kind:      "Deployment",
			Namespace: "default",
			Name:      "my-deploy",
			Messages:  []string{"not ready"},
		}))
	})

	t.Run("core group (empty) succeeds", func(t *testing.T) {
		require.NoError(t, setObjectStatus(t, "os-core", orbv1alpha1.ObjectStatus{
			Version: "v1",
			Kind:    "ConfigMap",
			Name:    "my-cm",
		}))
	})

	t.Run("cluster-scoped (empty namespace) succeeds", func(t *testing.T) {
		require.NoError(t, setObjectStatus(t, "os-cluster", orbv1alpha1.ObjectStatus{
			Group:   "rbac.authorization.k8s.io",
			Version: "v1",
			Kind:    "ClusterRole",
			Name:    "my-role",
		}))
	})

	t.Run("group exceeding 253 chars is rejected", func(t *testing.T) {
		requireStatusError(t, setObjectStatus(t, "os-long-group", orbv1alpha1.ObjectStatus{
			Group:   strings.Repeat("a", 254),
			Version: "v1",
			Kind:    "ConfigMap",
			Name:    "cm",
		}), objField("group"), "may not be more than 253")
	})

	t.Run("empty version is rejected", func(t *testing.T) {
		requireStatusError(t, setObjectStatus(t, "os-no-ver", orbv1alpha1.ObjectStatus{
			Version: "",
			Kind:    "ConfigMap",
			Name:    "cm",
		}), objField("version"), "should be at least 1 chars long")
	})

	t.Run("version exceeding 63 chars is rejected", func(t *testing.T) {
		requireStatusError(t, setObjectStatus(t, "os-long-ver", orbv1alpha1.ObjectStatus{
			Version: strings.Repeat("v", 64),
			Kind:    "ConfigMap",
			Name:    "cm",
		}), objField("version"), "may not be more than 63")
	})

	t.Run("empty kind is rejected", func(t *testing.T) {
		requireStatusError(t, setObjectStatus(t, "os-no-kind", orbv1alpha1.ObjectStatus{
			Version: "v1",
			Kind:    "",
			Name:    "cm",
		}), objField("kind"), "should be at least 1 chars long")
	})

	t.Run("kind exceeding 63 chars is rejected", func(t *testing.T) {
		requireStatusError(t, setObjectStatus(t, "os-long-kind", orbv1alpha1.ObjectStatus{
			Version: "v1",
			Kind:    strings.Repeat("K", 64),
			Name:    "cm",
		}), objField("kind"), "may not be more than 63")
	})

	t.Run("namespace exceeding 253 chars is rejected", func(t *testing.T) {
		requireStatusError(t, setObjectStatus(t, "os-long-ns", orbv1alpha1.ObjectStatus{
			Version:   "v1",
			Kind:      "ConfigMap",
			Namespace: strings.Repeat("n", 254),
			Name:      "cm",
		}), objField("namespace"), "may not be more than 253")
	})

	t.Run("empty name is rejected", func(t *testing.T) {
		requireStatusError(t, setObjectStatus(t, "os-no-name", orbv1alpha1.ObjectStatus{
			Version: "v1",
			Kind:    "ConfigMap",
			Name:    "",
		}), objField("name"), "should be at least 1 chars long")
	})

	t.Run("name exceeding 253 chars is rejected", func(t *testing.T) {
		requireStatusError(t, setObjectStatus(t, "os-long-name", orbv1alpha1.ObjectStatus{
			Version: "v1",
			Kind:    "ConfigMap",
			Name:    strings.Repeat("n", 254),
		}), objField("name"), "may not be more than 253")
	})

	t.Run("17 messages succeeds", func(t *testing.T) {
		msgs := make([]string, 17)
		for i := range msgs {
			msgs[i] = "msg"
		}
		require.NoError(t, setObjectStatus(t, "os-max-msgs", orbv1alpha1.ObjectStatus{
			Version:  "v1",
			Kind:     "ConfigMap",
			Name:     "cm",
			Messages: msgs,
		}))
	})

	t.Run("exceeding 17 messages is rejected", func(t *testing.T) {
		msgs := make([]string, 18)
		for i := range msgs {
			msgs[i] = "msg"
		}
		requireStatusError(t, setObjectStatus(t, "os-too-many-msgs", orbv1alpha1.ObjectStatus{
			Version:  "v1",
			Kind:     "ConfigMap",
			Name:     "cm",
			Messages: msgs,
		}), objField("messages"), "must have at most 17 items")
	})

	t.Run("message exceeding 1024 chars is rejected", func(t *testing.T) {
		requireStatusError(t, setObjectStatus(t, "os-long-msg", orbv1alpha1.ObjectStatus{
			Version:  "v1",
			Kind:     "ConfigMap",
			Name:     "cm",
			Messages: []string{strings.Repeat("m", 1025)},
		}), objField("messages[0]"), "may not be more than 1024")
	})

	t.Run("message at 1024 chars succeeds", func(t *testing.T) {
		require.NoError(t, setObjectStatus(t, "os-max-msg", orbv1alpha1.ObjectStatus{
			Version:  "v1",
			Kind:     "ConfigMap",
			Name:     "cm",
			Messages: []string{strings.Repeat("m", 1024)},
		}))
	})

	t.Run("empty messages list succeeds", func(t *testing.T) {
		require.NoError(t, setObjectStatus(t, "os-no-msgs", orbv1alpha1.ObjectStatus{
			Version: "v1",
			Kind:    "ConfigMap",
			Name:    "cm",
		}))
	})

	t.Run("1024 multi-byte runes succeeds", func(t *testing.T) {
		require.NoError(t, setObjectStatus(t, "os-mb-ok", orbv1alpha1.ObjectStatus{
			Version:  "v1",
			Kind:     "ConfigMap",
			Name:     "cm",
			Messages: []string{strings.Repeat("é", 1024)},
		}))
	})

	t.Run("1025 multi-byte runes is rejected", func(t *testing.T) {
		requireStatusError(t, setObjectStatus(t, "os-mb-over", orbv1alpha1.ObjectStatus{
			Version:  "v1",
			Kind:     "ConfigMap",
			Name:     "cm",
			Messages: []string{strings.Repeat("é", 1025)},
		}), objField("messages[0]"), "may not be more than 1024")
	})
}

func TestCOSR_Spec_Group_Validation(t *testing.T) {
	ctx := context.Background()

	createWithGroup := func(cosrName, group string) error {
		cosr := newCOSR(cosrName)
		cosr.Spec.Group = group
		return k8sClient.Create(ctx, cosr)
	}
	cleanup := func(t *testing.T, cosrName string) {
		t.Helper()
		t.Cleanup(func() {
			cosr := newCOSR(cosrName)
			require.NoError(t, k8sClient.Delete(ctx, cosr))
		})
	}

	for _, tc := range []struct {
		name  string
		group string
	}{
		{"simple lowercase", "mygroup"},
		{"with hyphens", "my-group"},
		{"with digits", "group1"},
		{"single char", "a"},
		{"letter then digit", "a1"},
		{"max 52 chars", "a" + strings.Repeat("b", 50) + "c"},
	} {
		t.Run(tc.name+" is accepted", func(t *testing.T) {
			cosrName := fmt.Sprintf("grp-ok-%s", tc.group)
			cleanup(t, cosrName)
			require.NoError(t, createWithGroup(cosrName, tc.group))
		})
	}

	for _, tc := range []struct {
		name     string
		cosrName string
		group    string
		msgSub   string
	}{
		{"empty", "grp-bad-empty", "", "should be at least 1"},
		{"exceeds 52 chars", "grp-bad-long", strings.Repeat("a", 53), "may not be more than 52"},
		{"starts with digit", "grp-bad-digit", "1group", "lowercase alphanumeric"},
		{"starts with hyphen", "grp-bad-hyphen", "-group", "lowercase alphanumeric"},
		{"ends with hyphen", "grp-bad-end", "group-", "lowercase alphanumeric"},
		{"uppercase", "grp-bad-upper", "Group", "lowercase alphanumeric"},
		{"contains underscore", "grp-bad-uscore", "my_group", "lowercase alphanumeric"},
		{"contains dot", "grp-bad-dot", "my.group", "lowercase alphanumeric"},
	} {
		t.Run(tc.name+" is rejected", func(t *testing.T) {
			requireStatusError(t, createWithGroup(tc.cosrName, tc.group),
				"spec.group", tc.msgSub)
		})
	}
}

func TestCOSR_Spec_PhaseName_DNS1035Validation(t *testing.T) {
	ctx := context.Background()

	createWithPhaseName := func(cosrName, phaseName string) error {
		cosr := newCOSR(cosrName)
		cosr.Spec.Phases[0].Name = phaseName
		return k8sClient.Create(ctx, cosr)
	}
	cleanup := func(t *testing.T, cosrName string) {
		t.Helper()
		t.Cleanup(func() {
			cosr := newCOSR(cosrName)
			require.NoError(t, k8sClient.Delete(ctx, cosr))
		})
	}

	for _, tc := range []struct {
		name      string
		phaseName string
	}{
		{"simple lowercase", "install"},
		{"with hyphens", "my-phase"},
		{"with digits", "phase1"},
		{"single char", "a"},
		{"letter then digit", "a1"},
		{"max 63 chars", "a" + strings.Repeat("b", 61) + "c"},
	} {
		t.Run(tc.name+" is accepted", func(t *testing.T) {
			cosrName := fmt.Sprintf("pn-ok-%s", tc.phaseName)
			if len(cosrName) > 63 {
				cosrName = cosrName[:63]
			}
			cleanup(t, cosrName)
			require.NoError(t, createWithPhaseName(cosrName, tc.phaseName))
		})
	}

	for _, tc := range []struct {
		name      string
		cosrName  string
		phaseName string
	}{
		{"starts with digit", "pn-bad-digit", "1phase"},
		{"starts with hyphen", "pn-bad-hyphen", "-phase"},
		{"ends with hyphen", "pn-bad-end", "phase-"},
		{"uppercase", "pn-bad-upper", "Phase"},
		{"contains underscore", "pn-bad-uscore", "my_phase"},
		{"contains dot", "pn-bad-dot", "my.phase"},
	} {
		t.Run(tc.name+" is rejected", func(t *testing.T) {
			err := createWithPhaseName(tc.cosrName, tc.phaseName)
			requireStatusError(t, err,
				"spec.phases[0].name", "must be a valid DNS-1035 label")
		})
	}
}
