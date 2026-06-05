package e2e

import (
	"context"
	"fmt"
	"strings"
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

	tmpl *templateSpecBuilder

	cosrs           map[string]*orbv1alpha1.ClusterObjectSetRevision
	lastCreatedCOSR string
	cosr            *cosrBuilder

	coss           map[string]*orbv1alpha1.ClusterObjectSet
	lastCreatedCOS string
	cos            *cosBuilder

	crds        []string
	trackedUIDs map[string]types.UID
}

type templateSpecBuilder struct {
	collisionProtection *orbv1alpha1.CollisionProtection
	phases              []orbv1alpha1.Phase
}

func (b *templateSpecBuilder) build() orbv1alpha1.ClusterObjectSetTemplateSpec {
	return orbv1alpha1.ClusterObjectSetTemplateSpec{
		CollisionProtection: b.collisionProtection,
		Phases:              b.phases,
	}
}

type cosrBuilder struct {
	nameOverride string
	group        string
	revision     uint32
	tmpl         *templateSpecBuilder
}

type cosBuilder struct {
	name                 string
	revisionHistoryLimit *int32
	labels               map[string]string
	annotations          map[string]string
	tmpl                 *templateSpecBuilder
}

func newTestContext(c client.Client) *testContext {
	return &testContext{
		client:      c,
		cosrs:       make(map[string]*orbv1alpha1.ClusterObjectSetRevision),
		coss:        make(map[string]*orbv1alpha1.ClusterObjectSet),
		trackedUIDs: make(map[string]types.UID),
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
	for _, cos := range tc.coss {
		_ = tc.client.Delete(ctx, cos)
	}
	for _, cosr := range tc.cosrs {
		_ = tc.client.Delete(ctx, cosr)
	}

	var allCOSRs orbv1alpha1.ClusterObjectSetRevisionList
	if err := tc.client.List(ctx, &allCOSRs); err == nil {
		for i := range allCOSRs.Items {
			if strings.HasPrefix(allCOSRs.Items[i].Spec.Group, tc.namespace+"-") {
				_ = tc.client.Delete(ctx, &allCOSRs.Items[i])
			}
		}
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

func (tc *testContext) resetCOSRBuilder(group string, revision uint32) {
	tc.tmpl = &templateSpecBuilder{}
	tc.cosr = &cosrBuilder{
		group:    tc.namespace + "-" + group,
		revision: revision,
		tmpl:     tc.tmpl,
	}
	tc.cos = nil
}

func (tc *testContext) resetCOSBuilder(name string) {
	tc.tmpl = &templateSpecBuilder{}
	tc.cos = &cosBuilder{
		name: tc.namespace + "-" + name,
		tmpl: tc.tmpl,
	}
	tc.cosr = nil
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
			Group:                        tc.cosr.group,
			Revision:                     tc.cosr.revision,
			LifecycleState:               orbv1alpha1.LifecycleStateActive,
			ClusterObjectSetTemplateSpec: tc.cosr.tmpl.build(),
		},
	}
}

func (tc *testContext) buildCOS() *orbv1alpha1.ClusterObjectSet {
	return &orbv1alpha1.ClusterObjectSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: tc.cos.name,
		},
		Spec: orbv1alpha1.ClusterObjectSetSpec{
			RevisionHistoryLimit: tc.cos.revisionHistoryLimit,
			Template: orbv1alpha1.ClusterObjectSetTemplate{
				Metadata: orbv1alpha1.ClusterObjectSetTemplateMetadata{
					Labels:      tc.cos.labels,
					Annotations: tc.cos.annotations,
				},
				Spec: tc.cos.tmpl.build(),
			},
		},
	}
}

func (tc *testContext) addPhase(name string) {
	tc.tmpl.phases = append(tc.tmpl.phases, orbv1alpha1.Phase{Name: name})
}

func (tc *testContext) currentPhase() *orbv1alpha1.Phase {
	if len(tc.tmpl.phases) == 0 {
		return nil
	}
	return &tc.tmpl.phases[len(tc.tmpl.phases)-1]
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

func (tc *testContext) createCOS(ctx context.Context) error {
	cos := tc.buildCOS()
	if err := tc.client.Create(ctx, cos); err != nil {
		return err
	}
	tc.coss[cos.Name] = cos
	tc.lastCreatedCOS = cos.Name
	return nil
}

type conditionAccessor func(client.Object) []metav1.Condition

func cosrConditions(obj client.Object) []metav1.Condition {
	return obj.(*orbv1alpha1.ClusterObjectSetRevision).Status.Conditions
}

func (tc *testContext) pollForConditionOn(ctx context.Context, obj client.Object, key types.NamespacedName, accessor conditionAccessor, condType string, status metav1.ConditionStatus) error {
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		if err := tc.client.Get(ctx, key, obj); err != nil {
			return false, nil
		}
		for _, c := range accessor(obj) {
			if c.Type == condType && c.Status == status {
				return true, nil
			}
		}
		return false, nil
	})
}

func (tc *testContext) pollForCOSRCondition(ctx context.Context, name string, condType string, status metav1.ConditionStatus) error {
	return tc.pollForConditionOn(ctx, &orbv1alpha1.ClusterObjectSetRevision{}, types.NamespacedName{Name: name}, cosrConditions, condType, status)
}

func (tc *testContext) pollForConditionWithReasonOn(ctx context.Context, obj client.Object, key types.NamespacedName, accessor conditionAccessor, condType string, status metav1.ConditionStatus, reason string) error {
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		if err := tc.client.Get(ctx, key, obj); err != nil {
			return false, nil
		}
		for _, c := range accessor(obj) {
			if c.Type == condType && c.Status == status && c.Reason == reason {
				return true, nil
			}
		}
		return false, nil
	})
}

func (tc *testContext) pollForCOSConditionWithReason(ctx context.Context, name string, condType string, status metav1.ConditionStatus, reason string) error {
	cos := &orbv1alpha1.ClusterObjectSet{}
	key := types.NamespacedName{Name: name}
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		if err := tc.client.Get(ctx, key, cos); err != nil {
			return false, nil
		}
		for _, c := range cos.Status.Conditions {
			if c.Type == condType && c.Status == status && c.Reason == reason && c.ObservedGeneration == cos.Generation {
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

func (tc *testContext) lastCreatedCOSName() string {
	return tc.lastCreatedCOS
}
