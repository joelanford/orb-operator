package e2e

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

	coss           map[string]*orbv1alpha1.ClusterObjectSet
	lastCreatedCOS string
	cos            *cosBuilder

	cods           map[string]*orbv1alpha1.ClusterObjectDeployment
	lastCreatedCOD string
	cod            *codBuilder

	crds                 []string
	createdObjects       []metav1.PartialObjectMetadata
	trackedConfigMapUIDs map[string]types.UID
	trackedCompletedAt   *metav1.Time
}

type templateSpecBuilder struct {
	collisionProtection *orbv1alpha1.CollisionProtection
	phases              []orbv1alpha1.Phase
}

func (b *templateSpecBuilder) build() orbv1alpha1.ClusterObjectDeploymentTemplateSpec {
	return orbv1alpha1.ClusterObjectDeploymentTemplateSpec{
		CollisionProtection: b.collisionProtection,
		Phases:              b.phases,
	}
}

type cosBuilder struct {
	nameOverride string
	group        string
	revision     uint32
	tmpl         *templateSpecBuilder
}

type codBuilder struct {
	name                    string
	revisionHistoryLimit    *int32
	progressDeadlineMinutes *int32
	labels                  map[string]string
	annotations             map[string]string
	tmpl                    *templateSpecBuilder
}

func newTestContext(c client.Client) *testContext {
	return &testContext{
		client:               c,
		coss:                 make(map[string]*orbv1alpha1.ClusterObjectSet),
		cods:                 make(map[string]*orbv1alpha1.ClusterObjectDeployment),
		trackedConfigMapUIDs: make(map[string]types.UID),
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
	for _, cod := range tc.cods {
		_ = tc.client.Delete(ctx, cod)
	}
	for _, cos := range tc.coss {
		_ = tc.client.Delete(ctx, cos)
	}

	var allCOSs orbv1alpha1.ClusterObjectSetList
	if err := tc.client.List(ctx, &allCOSs); err == nil {
		for i := range allCOSs.Items {
			if strings.HasPrefix(allCOSs.Items[i].Spec.Group, tc.namespace+"-") {
				_ = tc.client.Delete(ctx, &allCOSs.Items[i])
			}
		}
	}

	for _, name := range tc.crds {
		crd := &apiextensionsv1.CustomResourceDefinition{}
		crd.Name = name
		_ = tc.client.Delete(ctx, crd)
	}
	for i := len(tc.createdObjects) - 1; i >= 0; i-- {
		if err := tc.client.Delete(ctx, &tc.createdObjects[i]); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}

	ns := &corev1.Namespace{}
	ns.Name = tc.namespace
	return tc.client.Delete(ctx, ns)
}

func (tc *testContext) resetCOSBuilder(group string, revision uint32) {
	tc.tmpl = &templateSpecBuilder{}
	tc.cos = &cosBuilder{
		group:    tc.namespace + "-" + group,
		revision: revision,
		tmpl:     tc.tmpl,
	}
	tc.cod = nil
}

func (tc *testContext) resetCODBuilder(name string) {
	tc.tmpl = &templateSpecBuilder{}
	tc.cod = &codBuilder{
		name: tc.namespace + "-" + name,
		tmpl: tc.tmpl,
	}
	tc.cos = nil
}

func (tc *testContext) buildCOS() *orbv1alpha1.ClusterObjectSet {
	name := tc.cos.nameOverride
	if name == "" {
		name = fmt.Sprintf("%s-%d", tc.cos.group, tc.cos.revision)
	}
	return &orbv1alpha1.ClusterObjectSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: orbv1alpha1.ClusterObjectSetSpec{
			Group:                               tc.cos.group,
			Revision:                            tc.cos.revision,
			LifecycleState:                      orbv1alpha1.LifecycleStateActive,
			ClusterObjectDeploymentTemplateSpec: tc.cos.tmpl.build(),
		},
	}
}

func (tc *testContext) buildCOD() *orbv1alpha1.ClusterObjectDeployment {
	return &orbv1alpha1.ClusterObjectDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: tc.cod.name,
		},
		Spec: orbv1alpha1.ClusterObjectDeploymentSpec{
			RevisionHistoryLimit:    tc.cod.revisionHistoryLimit,
			ProgressDeadlineMinutes: tc.cod.progressDeadlineMinutes,
			Template: orbv1alpha1.ClusterObjectDeploymentTemplate{
				Metadata: orbv1alpha1.ClusterObjectDeploymentTemplateMetadata{
					Labels:      tc.cod.labels,
					Annotations: tc.cod.annotations,
				},
				Spec: tc.cod.tmpl.build(),
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

func (tc *testContext) lastObject() *orbv1alpha1.PhaseObject {
	phase := tc.currentPhase()
	return &phase.Objects[len(phase.Objects)-1]
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

func (tc *testContext) createCOS(ctx context.Context) error {
	cos := tc.buildCOS()
	if err := tc.client.Create(ctx, cos); err != nil {
		return err
	}
	tc.coss[cos.Name] = cos
	tc.lastCreatedCOS = cos.Name
	return nil
}

func (tc *testContext) createCOD(ctx context.Context) error {
	cod := tc.buildCOD()
	if err := tc.client.Create(ctx, cod); err != nil {
		return err
	}
	tc.cods[cod.Name] = cod
	tc.lastCreatedCOD = cod.Name
	return nil
}

type conditionAccessor func(client.Object) ([]metav1.Condition, int64)

func cosConditions(obj client.Object) ([]metav1.Condition, int64) {
	o := obj.(*orbv1alpha1.ClusterObjectSet)
	return o.Status.Conditions, o.Generation
}

func codConditions(obj client.Object) ([]metav1.Condition, int64) {
	o := obj.(*orbv1alpha1.ClusterObjectDeployment)
	return o.Status.Conditions, o.Generation
}

func (tc *testContext) pollForConditionOn(ctx context.Context, obj client.Object, key types.NamespacedName, accessor conditionAccessor, condType string, status metav1.ConditionStatus) error {
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		if err := tc.client.Get(ctx, key, obj); err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		conds, gen := accessor(obj)
		for _, c := range conds {
			if c.Type == condType && c.Status == status && c.ObservedGeneration == gen {
				return true, nil
			}
		}
		return false, nil
	})
}

func (tc *testContext) pollForConditionWithReasonOn(ctx context.Context, obj client.Object, key types.NamespacedName, accessor conditionAccessor, condType string, status metav1.ConditionStatus, reason string) error {
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		if err := tc.client.Get(ctx, key, obj); err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		conds, gen := accessor(obj)
		for _, c := range conds {
			if c.Type == condType && c.Status == status && c.Reason == reason && c.ObservedGeneration == gen {
				return true, nil
			}
		}
		return false, nil
	})
}

func (tc *testContext) pollForCOSCondition(ctx context.Context, name string, condType string, status metav1.ConditionStatus) error {
	return tc.pollForConditionOn(ctx, &orbv1alpha1.ClusterObjectSet{}, types.NamespacedName{Name: name}, cosConditions, condType, status)
}

func (tc *testContext) pollForCODConditionWithReason(ctx context.Context, name string, condType string, status metav1.ConditionStatus, reason string) error {
	return tc.pollForConditionWithReasonOn(ctx, &orbv1alpha1.ClusterObjectDeployment{}, types.NamespacedName{Name: name}, codConditions, condType, status, reason)
}

func (tc *testContext) pollForObject(ctx context.Context, key types.NamespacedName, obj client.Object) error {
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		if err := tc.client.Get(ctx, key, obj); err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

func pollForObjectMatching(tc *testContext, obj client.Object, key types.NamespacedName, match func() bool) error {
	return wait.PollUntilContextTimeout(context.Background(), pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		if err := tc.client.Get(ctx, key, obj); err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return match(), nil
	})
}

func (tc *testContext) pollForObjectAbsence(ctx context.Context, key types.NamespacedName, obj client.Object) error {
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		if err := tc.client.Get(ctx, key, obj); err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
}

func (tc *testContext) lastCreatedCOSName() string {
	return tc.lastCreatedCOS
}

func (tc *testContext) lastCreatedCODName() string {
	return tc.lastCreatedCOD
}

func pollMutateUpdate[T any, PT interface {
	*T
	client.Object
}](tc *testContext, key types.NamespacedName, mutate func(PT)) error {
	return wait.PollUntilContextTimeout(context.Background(), pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		obj := PT(new(T))
		if err := tc.client.Get(ctx, key, obj); err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		mutate(obj)
		if err := tc.client.Update(ctx, obj); err != nil {
			return false, nil
		}
		return true, nil
	})
}

func deleteObject[T any, PT interface {
	*T
	client.Object
}](tc *testContext, key types.NamespacedName, opts ...client.DeleteOption) error {
	obj := PT(new(T))
	obj.SetName(key.Name)
	obj.SetNamespace(key.Namespace)
	return tc.client.Delete(context.Background(), obj, opts...)
}

func expectError(err error, msg string) error {
	if err == nil {
		return fmt.Errorf("expected %s to fail, but it succeeded", msg)
	}
	return nil
}
