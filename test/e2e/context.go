package e2e

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

const pollTimeout = 15 * time.Second

const pollInterval = 100 * time.Millisecond

type testContext struct {
	client    client.Client
	namespace string

	cosrs           map[string]*orbv1alpha1.ClusterObjectSetRevision
	lastCreatedCOSR string
	cosr            *cosrBuilder
	crds            []string
}

type cosrBuilder struct {
	nameOverride string
	group        string
	revision     int32
	phases       []orbv1alpha1.Phase
}

func newTestContext(c client.Client) *testContext {
	return &testContext{
		client: c,
		cosrs:  make(map[string]*orbv1alpha1.ClusterObjectSetRevision),
	}
}

func (tc *testContext) setup(ctx context.Context) error {
	tc.namespace = fmt.Sprintf("e2e-%s", rand.String(8))
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: tc.namespace},
	}
	return tc.client.Create(ctx, ns)
}

func (tc *testContext) teardown(ctx context.Context) error {
	for _, cosr := range tc.cosrs {
		_ = tc.client.Delete(ctx, cosr)
	}
	for _, name := range tc.crds {
		crd := &apiextensionsv1.CustomResourceDefinition{}
		crd.Name = name
		_ = tc.client.Delete(ctx, crd)
	}

	ns := &corev1.Namespace{}
	ns.Name = tc.namespace
	return tc.client.Delete(ctx, ns)
}

func (tc *testContext) resetBuilder(group string, revision int32) {
	tc.cosr = &cosrBuilder{
		group:    tc.namespace + "-" + group,
		revision: revision,
	}
}

func (tc *testContext) buildCOSR() *orbv1alpha1.ClusterObjectSetRevision {
	name := tc.cosr.nameOverride
	if name == "" {
		name = fmt.Sprintf("%s-%d", tc.cosr.group, tc.cosr.revision)
	}
	return &orbv1alpha1.ClusterObjectSetRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: orbv1alpha1.ClusterObjectSetRevisionSpec{
			Group:    tc.cosr.group,
			Revision: tc.cosr.revision,
			Phases:   tc.cosr.phases,
		},
	}
}

func (tc *testContext) addPhase(name string) {
	tc.cosr.phases = append(tc.cosr.phases, orbv1alpha1.Phase{Name: name})
}

func (tc *testContext) currentPhase() *orbv1alpha1.Phase {
	if len(tc.cosr.phases) == 0 {
		return nil
	}
	return &tc.cosr.phases[len(tc.cosr.phases)-1]
}

func (tc *testContext) addObjectToPhase(obj runtime.Object) {
	phase := tc.currentPhase()
	phase.Objects = append(phase.Objects, orbv1alpha1.PhaseObject{
		Object: runtime.RawExtension{Object: obj},
	})
}

func (tc *testContext) addObjectWithAssertions(obj runtime.Object, assertions []orbv1alpha1.Assertion) {
	phase := tc.currentPhase()
	phase.Objects = append(phase.Objects, orbv1alpha1.PhaseObject{
		Object:     runtime.RawExtension{Object: obj},
		Assertions: assertions,
	})
}

func (tc *testContext) createCOSR(ctx context.Context) error {
	cosr := tc.buildCOSR()
	if err := tc.client.Create(ctx, cosr); err != nil {
		return err
	}
	tc.cosrs[cosr.Name] = cosr
	tc.lastCreatedCOSR = cosr.Name
	return nil
}

func (tc *testContext) pollForCondition(ctx context.Context, name string, condType string, status metav1.ConditionStatus) error {
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		cosr := &orbv1alpha1.ClusterObjectSetRevision{}
		if err := tc.client.Get(ctx, types.NamespacedName{Name: name}, cosr); err != nil {
			return false, nil
		}
		for _, c := range cosr.Status.Conditions {
			if c.Type == condType && c.Status == status {
				return true, nil
			}
		}
		return false, nil
	})
}

func (tc *testContext) pollForObject(ctx context.Context, key types.NamespacedName, obj client.Object) error {
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		if err := tc.client.Get(ctx, key, obj); err != nil {
			return false, nil
		}
		return true, nil
	})
}

func (tc *testContext) pollForObjectAbsence(ctx context.Context, key types.NamespacedName, obj client.Object) error {
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		err := tc.client.Get(ctx, key, obj)
		if err != nil {
			return true, nil
		}
		return false, nil
	})
}

func (tc *testContext) lastCreatedCOSRName() string {
	return tc.lastCreatedCOSR
}
