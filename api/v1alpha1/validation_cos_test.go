package v1alpha1_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func TestCOS_Status_CompletedAt_ImmutableOnceSet(t *testing.T) {
	ctx := context.Background()

	t.Run("setting completedAt on a new COS succeeds", func(t *testing.T) {
		cos := newCOS("completed-at-set")
		createCOS(t, ctx, cos)

		now := metav1.Now()
		cos.Status.CompletedAt = &now
		require.NoError(t, k8sClient.Status().Update(ctx, cos))
	})

	t.Run("clearing completedAt after it has been set is rejected", func(t *testing.T) {
		cos := newCOS("completed-at-clear")
		createCOS(t, ctx, cos)

		now := metav1.Now()
		cos.Status.CompletedAt = &now
		require.NoError(t, k8sClient.Status().Update(ctx, cos))

		cos.Status.CompletedAt = nil
		requireStatusError(t, k8sClient.Status().Update(ctx, cos),
			"status", "completedAt is immutable once set")
	})

	t.Run("changing completedAt to a different value is rejected", func(t *testing.T) {
		cos := newCOS("completed-at-change")
		createCOS(t, ctx, cos)

		now := metav1.Now()
		cos.Status.CompletedAt = &now
		require.NoError(t, k8sClient.Status().Update(ctx, cos))

		later := metav1.NewTime(now.Add(time.Hour))
		cos.Status.CompletedAt = &later
		requireStatusError(t, k8sClient.Status().Update(ctx, cos),
			"status", "completedAt is immutable once set")
	})

	t.Run("completedAt remains unset when never written", func(t *testing.T) {
		cos := newCOS("completed-at-unset")
		createCOS(t, ctx, cos)

		cos.Status.Conditions = []metav1.Condition{{
			Type:               "Available",
			Status:             metav1.ConditionFalse,
			Reason:             "Testing",
			LastTransitionTime: metav1.Now(),
		}}
		require.NoError(t, k8sClient.Status().Update(ctx, cos))
		assert.Nil(t, cos.Status.CompletedAt)
	})
}

func TestCOS_Status_ObservedPhase_Validation(t *testing.T) {
	ctx := context.Background()

	t.Run("valid observed phase succeeds", func(t *testing.T) {
		cos := newCOS("op-valid")
		createCOS(t, ctx, cos)

		cos.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
			Name:   "install",
			Status: orbv1alpha1.PhaseStatusReconciling,
		}}
		cos.Status.ObjectCounts = &orbv1alpha1.ObjectCounts{}
		require.NoError(t, k8sClient.Status().Update(ctx, cos))
	})

	t.Run("empty phase name is rejected", func(t *testing.T) {
		cos := newCOS("op-empty-name")
		createCOS(t, ctx, cos)

		cos.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
			Name:   "",
			Status: orbv1alpha1.PhaseStatusUnknown,
		}}
		cos.Status.ObjectCounts = &orbv1alpha1.ObjectCounts{}
		requireStatusError(t, k8sClient.Status().Update(ctx, cos),
			"status.observedPhases[0].name", "should be at least 1 chars long")
	})

	t.Run("phase name exceeding 63 chars is rejected", func(t *testing.T) {
		cos := newCOS("op-long-name")
		createCOS(t, ctx, cos)

		cos.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
			Name:   strings.Repeat("a", 64),
			Status: orbv1alpha1.PhaseStatusUnknown,
		}}
		cos.Status.ObjectCounts = &orbv1alpha1.ObjectCounts{}
		requireStatusError(t, k8sClient.Status().Update(ctx, cos),
			"status.observedPhases[0].name", "may not be more than 63")
	})

	for _, status := range []orbv1alpha1.PhaseStatus{
		orbv1alpha1.PhaseStatusInvalid,
		orbv1alpha1.PhaseStatusPending,
		orbv1alpha1.PhaseStatusReconciling,
		orbv1alpha1.PhaseStatusWaitingForAssertions,
		orbv1alpha1.PhaseStatusAvailable,
		orbv1alpha1.PhaseStatusUnknown,
		orbv1alpha1.PhaseStatusSuperseded,
		orbv1alpha1.PhaseStatusTearingDown,
		orbv1alpha1.PhaseStatusTeardownComplete,
	} {
		t.Run(fmt.Sprintf("phase status %q is accepted", status), func(t *testing.T) {
			cos := newCOS(fmt.Sprintf("op-status-%s", strings.ToLower(string(status))))
			createCOS(t, ctx, cos)

			cos.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
				Name:   "install",
				Status: status,
			}}
			cos.Status.ObjectCounts = &orbv1alpha1.ObjectCounts{}
			require.NoError(t, k8sClient.Status().Update(ctx, cos))
		})
	}

	t.Run("invalid phase status enum is rejected", func(t *testing.T) {
		cos := newCOS("op-bad-status")
		createCOS(t, ctx, cos)

		cos.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
			Name:   "install",
			Status: orbv1alpha1.PhaseStatus("Bogus"),
		}}
		cos.Status.ObjectCounts = &orbv1alpha1.ObjectCounts{}
		requireStatusError(t, k8sClient.Status().Update(ctx, cos),
			"status.observedPhases[0].status", "Unsupported value")
	})

	t.Run("phase error at 1024 chars succeeds", func(t *testing.T) {
		cos := newCOS("op-max-err")
		createCOS(t, ctx, cos)

		cos.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
			Name:    "install",
			Status:  orbv1alpha1.PhaseStatusReconciling,
			Message: strings.Repeat("e", 1024),
		}}
		cos.Status.ObjectCounts = &orbv1alpha1.ObjectCounts{}
		require.NoError(t, k8sClient.Status().Update(ctx, cos))
	})

	t.Run("phase error exceeding 1024 chars is rejected", func(t *testing.T) {
		cos := newCOS("op-long-err")
		createCOS(t, ctx, cos)

		cos.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
			Name:    "install",
			Status:  orbv1alpha1.PhaseStatusReconciling,
			Message: strings.Repeat("e", 1025),
		}}
		cos.Status.ObjectCounts = &orbv1alpha1.ObjectCounts{}
		requireStatusError(t, k8sClient.Status().Update(ctx, cos),
			"status.observedPhases[0].message", "may not be more than 1024")
	})

	t.Run("observedPhases exceeding 20 is rejected", func(t *testing.T) {
		cos := newCOS("op-too-many-phases")
		createCOS(t, ctx, cos)

		phases := make([]orbv1alpha1.ObservedPhase, 21)
		for i := range phases {
			phases[i] = orbv1alpha1.ObservedPhase{
				Name:   fmt.Sprintf("phase-%d", i),
				Status: orbv1alpha1.PhaseStatusUnknown,
			}
		}
		cos.Status.ObservedPhases = phases
		cos.Status.ObjectCounts = &orbv1alpha1.ObjectCounts{}
		requireStatusError(t, k8sClient.Status().Update(ctx, cos),
			"status.observedPhases", "must have at most 20 items")
	})

	t.Run("objectDetails at max (50) succeeds", func(t *testing.T) {
		cos := newCOS("op-max-objects")
		createCOS(t, ctx, cos)

		objects := make([]orbv1alpha1.ObjectStatus, 50)
		for i := range objects {
			objects[i] = orbv1alpha1.ObjectStatus{
				Version: "v1",
				Kind:    "ConfigMap",
				Name:    "cm",
			}
		}
		cos.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
			Name:          "install",
			Status:        orbv1alpha1.PhaseStatusReconciling,
			ObjectDetails: objects,
		}}
		cos.Status.ObjectCounts = &orbv1alpha1.ObjectCounts{}
		require.NoError(t, k8sClient.Status().Update(ctx, cos))
	})

	t.Run("objectDetails exceeding 50 is rejected", func(t *testing.T) {
		cos := newCOS("op-too-many-obj")
		createCOS(t, ctx, cos)

		objects := make([]orbv1alpha1.ObjectStatus, 51)
		for i := range objects {
			objects[i] = orbv1alpha1.ObjectStatus{
				Version: "v1",
				Kind:    "ConfigMap",
				Name:    "cm",
			}
		}
		cos.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
			Name:          "install",
			Status:        orbv1alpha1.PhaseStatusReconciling,
			ObjectDetails: objects,
		}}
		cos.Status.ObjectCounts = &orbv1alpha1.ObjectCounts{}
		requireStatusError(t, k8sClient.Status().Update(ctx, cos),
			"status.observedPhases[0].objectDetails", "must have at most 50 items")
	})

	t.Run("setting phase completedAt succeeds", func(t *testing.T) {
		cos := newCOS("op-completed-at-set")
		createCOS(t, ctx, cos)

		now := metav1.Now()
		cos.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
			Name:        "install",
			Status:      orbv1alpha1.PhaseStatusAvailable,
			CompletedAt: &now,
		}}
		cos.Status.ObjectCounts = &orbv1alpha1.ObjectCounts{}
		require.NoError(t, k8sClient.Status().Update(ctx, cos))
	})

	t.Run("clearing phase completedAt after it has been set is rejected", func(t *testing.T) {
		cos := newCOS("op-completed-at-clear")
		createCOS(t, ctx, cos)

		now := metav1.Now()
		cos.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
			Name:        "install",
			Status:      orbv1alpha1.PhaseStatusAvailable,
			CompletedAt: &now,
		}}
		cos.Status.ObjectCounts = &orbv1alpha1.ObjectCounts{}
		require.NoError(t, k8sClient.Status().Update(ctx, cos))

		cos.Status.ObservedPhases[0].CompletedAt = nil
		requireStatusError(t, k8sClient.Status().Update(ctx, cos),
			"status.observedPhases[0]", "completedAt is immutable once set")
	})

	t.Run("changing phase completedAt to a different value is rejected", func(t *testing.T) {
		cos := newCOS("op-completed-at-change")
		createCOS(t, ctx, cos)

		now := metav1.Now()
		cos.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
			Name:        "install",
			Status:      orbv1alpha1.PhaseStatusAvailable,
			CompletedAt: &now,
		}}
		cos.Status.ObjectCounts = &orbv1alpha1.ObjectCounts{}
		require.NoError(t, k8sClient.Status().Update(ctx, cos))

		later := metav1.NewTime(now.Add(time.Hour))
		cos.Status.ObservedPhases[0].CompletedAt = &later
		requireStatusError(t, k8sClient.Status().Update(ctx, cos),
			"status.observedPhases[0]", "completedAt is immutable once set")
	})
}

func TestCOS_Status_ObjectStatus_Validation(t *testing.T) {
	ctx := context.Background()

	setObjectStatus := func(t *testing.T, name string, os orbv1alpha1.ObjectStatus) error {
		t.Helper()
		cos := newCOS(name)
		createCOS(t, ctx, cos)
		cos.Status.ObservedPhases = []orbv1alpha1.ObservedPhase{{
			Name:          "install",
			Status:        orbv1alpha1.PhaseStatusReconciling,
			ObjectDetails: []orbv1alpha1.ObjectStatus{os},
		}}
		cos.Status.ObjectCounts = &orbv1alpha1.ObjectCounts{}
		return k8sClient.Status().Update(ctx, cos)
	}

	objField := func(f string) string {
		return "status.observedPhases[0].objectDetails[0]." + f
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

func TestCOS_Spec_Group_Validation(t *testing.T) {
	ctx := context.Background()

	createWithGroup := func(cosName, group string) error {
		cos := newCOS(cosName)
		cos.Spec.Group = group
		return k8sClient.Create(ctx, cos)
	}
	cleanup := func(t *testing.T, cosName string) {
		t.Helper()
		t.Cleanup(func() {
			cos := newCOS(cosName)
			require.NoError(t, k8sClient.Delete(ctx, cos))
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
			cosName := fmt.Sprintf("grp-ok-%s", tc.group)
			cleanup(t, cosName)
			require.NoError(t, createWithGroup(cosName, tc.group))
		})
	}

	for _, tc := range []struct {
		name    string
		cosName string
		group   string
		msgSub  string
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
			requireStatusError(t, createWithGroup(tc.cosName, tc.group),
				"spec.group", tc.msgSub)
		})
	}
}

func TestCOS_Spec_PhaseName_DNS1035Validation(t *testing.T) {
	ctx := context.Background()

	createWithPhaseName := func(cosName, phaseName string) error {
		cos := newCOS(cosName)
		cos.Spec.Phases[0].Name = phaseName
		return k8sClient.Create(ctx, cos)
	}
	cleanup := func(t *testing.T, cosName string) {
		t.Helper()
		t.Cleanup(func() {
			cos := newCOS(cosName)
			require.NoError(t, k8sClient.Delete(ctx, cos))
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
			cosName := fmt.Sprintf("pn-ok-%s", tc.phaseName)
			if len(cosName) > 63 {
				cosName = cosName[:63]
			}
			cleanup(t, cosName)
			require.NoError(t, createWithPhaseName(cosName, tc.phaseName))
		})
	}

	for _, tc := range []struct {
		name      string
		cosName   string
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
			err := createWithPhaseName(tc.cosName, tc.phaseName)
			requireStatusError(t, err,
				"spec.phases[0].name", "must be a valid DNS-1035 label")
		})
	}
}

func TestCOS_Status_ResolvedContentHash_ImmutableOnceSet(t *testing.T) {
	ctx := context.Background()

	t.Run("setting resolvedContentHash on a new COS succeeds", func(t *testing.T) {
		cos := newCOS("rch-set")
		createCOS(t, ctx, cos)

		cos.Status.ResolvedContentHash = "abc123"
		require.NoError(t, k8sClient.Status().Update(ctx, cos))
	})

	t.Run("clearing resolvedContentHash after it has been set is rejected", func(t *testing.T) {
		cos := newCOS("rch-clear")
		createCOS(t, ctx, cos)

		cos.Status.ResolvedContentHash = "abc123"
		require.NoError(t, k8sClient.Status().Update(ctx, cos))

		cos.Status.ResolvedContentHash = ""
		requireStatusError(t, k8sClient.Status().Update(ctx, cos),
			"status", "resolvedContentHash is immutable once set")
	})

	t.Run("changing resolvedContentHash to a different value is rejected", func(t *testing.T) {
		cos := newCOS("rch-change")
		createCOS(t, ctx, cos)

		cos.Status.ResolvedContentHash = "abc123"
		require.NoError(t, k8sClient.Status().Update(ctx, cos))

		cos.Status.ResolvedContentHash = "def456"
		requireStatusError(t, k8sClient.Status().Update(ctx, cos),
			"status", "resolvedContentHash is immutable once set")
	})

	t.Run("resolvedContentHash remains unset when never written", func(t *testing.T) {
		cos := newCOS("rch-unset")
		createCOS(t, ctx, cos)

		cos.Status.Conditions = []metav1.Condition{{
			Type:               "Available",
			Status:             metav1.ConditionFalse,
			Reason:             "Testing",
			LastTransitionTime: metav1.Now(),
		}}
		require.NoError(t, k8sClient.Status().Update(ctx, cos))
		assert.Empty(t, cos.Status.ResolvedContentHash)
	})
}
