package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cucumber/godog"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/yaml"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func registerAssertSteps(sc *godog.ScenarioContext, tc *testContext) {
	sc.Step(`^the ConfigMap "([^"]*)" should (exist|not exist|be recreated)$`, tc.theConfigMapExistenceCheck)
	sc.Step(`^the ConfigMap "([^"]*)" UID is tracked$`, tc.theConfigMapUIDisTracked)
	sc.Step(`^the ConfigMap "([^"]*)" should not have been deleted and recreated$`, tc.theConfigMapShouldNotHaveBeenRecreated)
	sc.Step(`^a resource should match:$`, tc.aResourceShouldMatch)
	sc.Step(`^the CRD "([^"]*)" should exist$`, tc.theCRDShouldExist)
	sc.Step(`^the "([^"]*)" named "([^"]*)" should exist$`, tc.theCRShouldExist)
	sc.Step(`^the ConfigMap "([^"]*)" should (have|not have) an owner reference$`, tc.theConfigMapOwnerRefCheck)
	sc.Step(`^the ConfigMap "([^"]*)" should have a controller owner reference to COSR with group "([^"]*)" and revision (\d+)$`, tc.theConfigMapShouldBeOwnedByCOSR)
	sc.Step(`^the COSR should not exist$`, tc.theCOSRShouldNotExist)
	sc.Step(`^the COSR should have condition "([^"]*)" with status "([^"]*)"$`, tc.theCOSRShouldHaveCondition)
	sc.Step(`^the COSR should have condition "([^"]*)" with status "([^"]*)" and reason "([^"]*)"$`, tc.theCOSRShouldHaveConditionWithReason)
	sc.Step(`^the COSR in group "([^"]*)" revision (\d+) should have condition "([^"]*)" with status "([^"]*)"$`, tc.theCOSRInGroupShouldHaveCondition)
	sc.Step(`^revision (\d+) should have condition "([^"]*)" with status "([^"]*)" and reason "([^"]*)"$`, tc.revisionShouldHaveConditionWithReason)
	sc.Step(`^revision (\d+) should have observed phase "([^"]*)" with status "([^"]*)"$`, tc.revisionShouldHaveObservedPhase)

	// Phase status assert steps
	sc.Step(`^the COSR should have observed phase "([^"]*)" with status "([^"]*)"$`, tc.theCOSRShouldHaveObservedPhase)
	sc.Step(`^the COSR should have (\d+) observed phases$`, tc.theCOSRShouldHaveObservedPhaseCount)
	sc.Step(`^observed phase "([^"]*)" should have (\d+) incomplete objects$`, tc.observedPhaseShouldHaveIncompleteObjectCount)
	sc.Step(`^observed phase "([^"]*)" should have an incomplete object "([^"]*)"$`, tc.observedPhaseShouldHaveIncompleteObjectNamed)
	sc.Step(`^the COSR should have no observed phases$`, tc.theCOSRShouldHaveNoObservedPhases)
	sc.Step(`^the COSR should have completedAt set$`, tc.theCOSRShouldHaveCompletedAt)
	sc.Step(`^the COSR should not have completedAt set$`, tc.theCOSRShouldNotHaveCompletedAt)
	sc.Step(`^the COSR completedAt should be preserved$`, tc.theCOSRCompletedAtShouldBePreserved)
	sc.Step(`^the COSR completedAt is tracked$`, tc.theCOSRCompletedAtIsTracked)

	// COD assert steps
	sc.Step(`^a COSR should exist with group "([^"]*)" and revision (\d+)$`, tc.aCOSRShouldExistWithGroupAndRevision)
	sc.Step(`^the COSR with group "([^"]*)" and revision (\d+) should have lifecycleState "([^"]*)"$`, tc.cosrShouldHaveLifecycleState)
	sc.Step(`^the COSR with group "([^"]*)" and revision (\d+) should have collisionProtection "([^"]*)"$`, tc.cosrShouldHaveCollisionProtection)
	sc.Step(`^the COSR with group "([^"]*)" and revision (\d+) should have (\d+) phases$`, tc.cosrShouldHavePhaseCount)
	sc.Step(`^the COSR with group "([^"]*)" and revision (\d+) should have (label|annotation) "([^"]*)" with value "([^"]*)"$`, tc.cosrShouldHaveMetadata)
	sc.Step(`^the COSR with group "([^"]*)" and revision (\d+) should not have label "([^"]*)" with value "([^"]*)"$`, tc.cosrShouldNotHaveLabelValue)
	sc.Step(`^the COSR with group "([^"]*)" and revision (\d+) should be named "([^"]*)"$`, tc.cosrShouldBeNamed)
	sc.Step(`^the COSR with group "([^"]*)" and revision (\d+) should have a controller owner reference to COD "([^"]*)"$`, tc.cosrShouldHaveControllerOwnerRef)
	sc.Step(`^the COSR with group "([^"]*)" and revision (\d+) should not exist$`, tc.cosrShouldNotExist)
	sc.Step(`^the COSR with group "([^"]*)" and revision (\d+) should not have an owner reference$`, tc.cosrShouldNotHaveOwnerRef)
	sc.Step(`^the COSR with group "([^"]*)" and revision (\d+) should not have finalizer "([^"]*)"$`, tc.cosrShouldNotHaveFinalizer)
	sc.Step(`^the COSR count for COD "([^"]*)" should be (\d+)$`, tc.cosrCountForCODShouldBe)
	sc.Step(`^the COD "([^"]*)" should be Available$`, tc.theCODShouldBeAvailable)
	sc.Step(`^the COD "([^"]*)" should have condition "([^"]*)" with status "([^"]*)" and reason "([^"]*)"$`, tc.theCODShouldHaveConditionWithReason)
	sc.Step(`^the COD "([^"]*)" should become available without becoming unavailable$`, tc.theCODShouldBecomeAvailableWithoutBecomingUnavailable)
	sc.Step(`^the COD "([^"]*)" should have active revision (\d+)$`, tc.theCODShouldHaveActiveRevision)
	sc.Step(`^the stamped COSR spec for "([^"]*)" revision (\d+) should match the COD template spec$`, tc.stampedCOSRSpecShouldMatchTemplate)
}

func (tc *testContext) theConfigMapExistenceCheck(name, expectation string) error {
	key := types.NamespacedName{Namespace: tc.namespace, Name: name}
	if expectation == "not exist" {
		return tc.pollForObjectAbsence(context.Background(), key, &corev1.ConfigMap{})
	}
	return tc.pollForObject(context.Background(), key, &corev1.ConfigMap{})
}

func (tc *testContext) theConfigMapUIDisTracked(name string) error {
	cm := &corev1.ConfigMap{}
	key := types.NamespacedName{Namespace: tc.namespace, Name: name}
	if err := tc.client.Get(context.Background(), key, cm); err != nil {
		return fmt.Errorf("ConfigMap %q should exist: %w", name, err)
	}
	tc.trackedConfigMapUIDs[name] = cm.UID
	return nil
}

func (tc *testContext) theConfigMapShouldNotHaveBeenRecreated(name string) error {
	cm := &corev1.ConfigMap{}
	key := types.NamespacedName{Namespace: tc.namespace, Name: name}
	if err := tc.client.Get(context.Background(), key, cm); err != nil {
		return fmt.Errorf("ConfigMap %q should exist: %w", name, err)
	}
	if tracked, ok := tc.trackedConfigMapUIDs[name]; ok && cm.UID != tracked {
		return fmt.Errorf("ConfigMap %q was recreated: UID changed from %s to %s", name, tracked, cm.UID)
	}
	return nil
}

func (tc *testContext) aResourceShouldMatch(doc *godog.DocString) error {
	content := strings.ReplaceAll(doc.Content, "${NAMESPACE}", tc.namespace)
	expected := &unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(content), &expected.Object); err != nil {
		return fmt.Errorf("parsing expected YAML: %w", err)
	}
	actual := &unstructured.Unstructured{}
	actual.SetGroupVersionKind(expected.GroupVersionKind())
	key := types.NamespacedName{Name: expected.GetName(), Namespace: expected.GetNamespace()}
	return pollForObjectMatching(tc, actual, key, func() bool {
		return equality.Semantic.DeepDerivative(expected.Object, actual.Object)
	})
}

func (tc *testContext) theCRDShouldExist(name string) error {
	crd := &apiextensionsv1.CustomResourceDefinition{}
	return tc.pollForObject(context.Background(), types.NamespacedName{Name: name + "." + tc.namespace + ".e2e.orb.dev"}, crd)
}

func (tc *testContext) theCRShouldExist(crdName, crName string) error {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(tc.crGVK(crdName))
	return tc.pollForObject(context.Background(), types.NamespacedName{Name: crName}, obj)
}

func (tc *testContext) crGVK(crdName string) schema.GroupVersionKind {
	kind := crdName[:len(crdName)-1]
	return schema.GroupVersionKind{
		Group:   tc.namespace + ".e2e.orb.dev",
		Version: "v1alpha1",
		Kind:    capitalize(kind),
	}
}

func (tc *testContext) theConfigMapOwnerRefCheck(name, haveOrNotHave string) error {
	cm := &corev1.ConfigMap{}
	if err := tc.pollForObject(context.Background(), types.NamespacedName{Namespace: tc.namespace, Name: name}, cm); err != nil {
		return err
	}
	expectOwnerRef := haveOrNotHave == "have"
	if expectOwnerRef && len(cm.OwnerReferences) == 0 {
		return fmt.Errorf("ConfigMap %q has no owner references", name)
	}
	if !expectOwnerRef && len(cm.OwnerReferences) > 0 {
		return fmt.Errorf("ConfigMap %q still has owner references: %v", name, cm.OwnerReferences)
	}
	return nil
}

func (tc *testContext) theConfigMapShouldBeOwnedByCOSR(name, group string, revision uint32) error {
	expectedOwner := tc.cosrName(group, revision)
	cm := &corev1.ConfigMap{}
	return pollForObjectMatching(tc, cm, types.NamespacedName{Namespace: tc.namespace, Name: name}, func() bool {
		ref := metav1.GetControllerOf(cm)
		return ref != nil && ref.Kind == "ClusterObjectSetRevision" && ref.Name == expectedOwner
	})
}

func (tc *testContext) theCOSRShouldNotExist() error {
	name := tc.lastCreatedCOSRName()
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	return tc.pollForObjectAbsence(context.Background(), types.NamespacedName{Name: name}, cosr)
}

func (tc *testContext) theCOSRShouldHaveCondition(condType, status string) error {
	name := tc.lastCreatedCOSRName()
	return tc.pollForCOSRCondition(context.Background(), name, condType, metav1.ConditionStatus(status))
}

func (tc *testContext) theCOSRInGroupShouldHaveCondition(group string, revision uint32, condType, status string) error {
	return tc.pollForCOSRCondition(context.Background(), tc.cosrName(group, revision), condType, metav1.ConditionStatus(status))
}

func (tc *testContext) theCOSRShouldHaveConditionWithReason(condType, status, reason string) error {
	name := tc.lastCreatedCOSRName()
	return tc.pollForConditionWithReasonOn(
		context.Background(),
		&orbv1alpha1.ClusterObjectSetRevision{},
		types.NamespacedName{Name: name},
		cosrConditions, condType, metav1.ConditionStatus(status), reason,
	)
}

func (tc *testContext) revisionShouldHaveConditionWithReason(revision uint32, condType, status, reason string) error {
	for name, cosr := range tc.cosrs {
		if cosr.Spec.Revision == revision {
			return tc.pollForConditionWithReasonOn(
				context.Background(),
				&orbv1alpha1.ClusterObjectSetRevision{},
				types.NamespacedName{Name: name},
				cosrConditions, condType, metav1.ConditionStatus(status), reason,
			)
		}
	}
	return fmt.Errorf("revision %d not found", revision)
}

func (tc *testContext) revisionShouldHaveObservedPhase(revision uint32, phaseName, status string) error {
	for name, cosr := range tc.cosrs {
		if cosr.Spec.Revision == revision {
			obj := &orbv1alpha1.ClusterObjectSetRevision{}
			return pollForObjectMatching(tc, obj, types.NamespacedName{Name: name}, func() bool {
				for _, op := range obj.Status.ObservedPhases {
					if op.Name == phaseName && string(op.Status) == status {
						return true
					}
				}
				return false
			})
		}
	}
	return fmt.Errorf("revision %d not found", revision)
}

func (tc *testContext) codFullName(name string) string {
	return tc.namespace + "-" + name
}

func (tc *testContext) cosrName(group string, revision uint32) string {
	return fmt.Sprintf("%s-%s-%d", tc.namespace, group, revision)
}

func (tc *testContext) getCOSR(ctx context.Context, group string, revision uint32) (*orbv1alpha1.ClusterObjectSetRevision, error) {
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	if err := tc.pollForObject(ctx, types.NamespacedName{Name: tc.cosrName(group, revision)}, cosr); err != nil {
		return nil, fmt.Errorf("COSR %s-%d not found: %w", group, revision, err)
	}
	return cosr, nil
}

func (tc *testContext) aCOSRShouldExistWithGroupAndRevision(group string, revision uint32) error {
	_, err := tc.getCOSR(context.Background(), group, revision)
	return err
}

func (tc *testContext) cosrShouldHaveLifecycleState(group string, revision uint32, state string) error {
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	return pollForObjectMatching(tc, cosr, types.NamespacedName{Name: tc.cosrName(group, revision)}, func() bool {
		actual := string(cosr.Spec.LifecycleState)
		if actual == "" {
			actual = "Active"
		}
		return actual == state
	})
}

func (tc *testContext) cosrShouldHaveCollisionProtection(group string, revision uint32, cp string) error {
	cosr, err := tc.getCOSR(context.Background(), group, revision)
	if err != nil {
		return err
	}
	if cosr.Spec.CollisionProtection == nil {
		return fmt.Errorf("COSR %s-%d collisionProtection is nil, want %q", group, revision, cp)
	}
	if string(*cosr.Spec.CollisionProtection) != cp {
		return fmt.Errorf("COSR %s-%d collisionProtection: got %q, want %q", group, revision, *cosr.Spec.CollisionProtection, cp)
	}
	return nil
}

func (tc *testContext) cosrShouldHavePhaseCount(group string, revision uint32, count int) error {
	cosr, err := tc.getCOSR(context.Background(), group, revision)
	if err != nil {
		return err
	}
	if len(cosr.Spec.Phases) != count {
		return fmt.Errorf("COSR %s-%d phases: got %d, want %d", group, revision, len(cosr.Spec.Phases), count)
	}
	return nil
}

func (tc *testContext) cosrShouldHaveMetadata(group string, revision uint32, kind, key, value string) error {
	cosr, err := tc.getCOSR(context.Background(), group, revision)
	if err != nil {
		return err
	}
	m := cosr.Labels
	if kind == "annotation" {
		m = cosr.Annotations
	}
	if got := m[key]; got != value {
		return fmt.Errorf("COSR %s-%d %s %q: got %q, want %q", group, revision, kind, key, got, value)
	}
	return nil
}

func (tc *testContext) cosrShouldNotHaveLabelValue(group string, revision uint32, key, value string) error {
	cosr, err := tc.getCOSR(context.Background(), group, revision)
	if err != nil {
		return err
	}
	if got := cosr.Labels[key]; got == value {
		return fmt.Errorf("COSR %s-%d label %q: got %q, should not equal %q", group, revision, key, got, value)
	}
	return nil
}

func (tc *testContext) cosrShouldBeNamed(group string, revision uint32, expectedName string) error {
	cosr, err := tc.getCOSR(context.Background(), group, revision)
	if err != nil {
		return err
	}
	expected := tc.codFullName(expectedName)
	if cosr.Name != expected {
		return fmt.Errorf("COSR %s-%d name: got %q, want %q", group, revision, cosr.Name, expected)
	}
	return nil
}

func (tc *testContext) cosrShouldHaveControllerOwnerRef(group string, revision uint32, codName string) error {
	fullCODName := tc.codFullName(codName)
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	return pollForObjectMatching(tc, cosr, types.NamespacedName{Name: tc.cosrName(group, revision)}, func() bool {
		for _, ref := range cosr.OwnerReferences {
			if ref.Kind == "ClusterObjectDeployment" && ref.Name == fullCODName && ref.Controller != nil && *ref.Controller {
				return true
			}
		}
		return false
	})
}

func (tc *testContext) cosrShouldNotExist(group string, revision uint32) error {
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	return tc.pollForObjectAbsence(context.Background(), types.NamespacedName{Name: tc.cosrName(group, revision)}, cosr)
}

func (tc *testContext) cosrShouldNotHaveOwnerRef(group string, revision uint32) error {
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	return pollForObjectMatching(tc, cosr, types.NamespacedName{Name: tc.cosrName(group, revision)}, func() bool {
		return len(cosr.OwnerReferences) == 0
	})
}

func (tc *testContext) cosrShouldNotHaveFinalizer(group string, revision uint32, finalizer string) error {
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	return pollForObjectMatching(tc, cosr, types.NamespacedName{Name: tc.cosrName(group, revision)}, func() bool {
		for _, f := range cosr.Finalizers {
			if f == finalizer {
				return false
			}
		}
		return true
	})
}

func (tc *testContext) cosrCountForCODShouldBe(codName string, count int) error {
	fullCODName := tc.codFullName(codName)
	var list orbv1alpha1.ClusterObjectSetRevisionList
	if err := tc.client.List(context.Background(), &list); err != nil {
		return err
	}
	actual := 0
	for _, cosr := range list.Items {
		if cosr.Spec.Group == fullCODName {
			actual++
		}
	}
	if actual != count {
		return fmt.Errorf("COSR count for COD %q: got %d, want %d", codName, actual, count)
	}
	return nil
}

func normalizeViaJSON(v any) (any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (tc *testContext) stampedCOSRSpecShouldMatchTemplate(group string, revision uint32) error {
	cosr, err := tc.getCOSR(context.Background(), group, revision)
	if err != nil {
		return err
	}
	expected := tc.tmpl.build()
	actual := cosr.Spec.ClusterObjectDeploymentTemplateSpec

	// Normalize both through JSON so that runtime.RawExtension.Object vs .Raw differences are eliminated.
	expectedNorm, err := normalizeViaJSON(expected)
	if err != nil {
		return fmt.Errorf("normalizing expected: %w", err)
	}
	actualNorm, err := normalizeViaJSON(actual)
	if err != nil {
		return fmt.Errorf("normalizing actual: %w", err)
	}

	if !equality.Semantic.DeepEqual(expectedNorm, actualNorm) {
		return fmt.Errorf("COSR spec does not match COD template spec:\n%s", cmp.Diff(expectedNorm, actualNorm))
	}
	return nil
}

func (tc *testContext) theCODShouldBecomeAvailableWithoutBecomingUnavailable(codName string) error {
	fullCODName := tc.codFullName(codName)
	cod := &orbv1alpha1.ClusterObjectDeployment{}
	key := types.NamespacedName{Name: fullCODName}
	var sawUnavailable bool
	err := wait.PollUntilContextTimeout(context.Background(), pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		if err := tc.client.Get(ctx, key, cod); err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		for _, c := range cod.Status.Conditions {
			if c.Type != orbv1alpha1.ConditionTypeAvailable || c.ObservedGeneration != cod.Generation {
				continue
			}
			if c.Status == metav1.ConditionFalse {
				sawUnavailable = true
			}
			if c.Status == metav1.ConditionTrue && c.Reason == orbv1alpha1.ReasonAvailable {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		return err
	}
	if sawUnavailable {
		return fmt.Errorf("COD %q became unavailable during rollout", codName)
	}
	return nil
}

func (tc *testContext) theCODShouldHaveActiveRevision(codName string, revision uint32) error {
	fullCODName := tc.codFullName(codName)
	expectedCOSRName := fmt.Sprintf("%s-%d", fullCODName, revision)
	cod := &orbv1alpha1.ClusterObjectDeployment{}
	return pollForObjectMatching(tc, cod, types.NamespacedName{Name: fullCODName}, func() bool {
		for _, rs := range cod.Status.ActiveRevisions {
			if rs.Name == expectedCOSRName {
				return true
			}
		}
		return false
	})
}

func (tc *testContext) theCOSRShouldHaveObservedPhase(phaseName, status string) error {
	name := tc.lastCreatedCOSRName()
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	return pollForObjectMatching(tc, cosr, types.NamespacedName{Name: name}, func() bool {
		for _, op := range cosr.Status.ObservedPhases {
			if op.Name == phaseName && string(op.Status) == status {
				return true
			}
		}
		return false
	})
}

func (tc *testContext) theCOSRShouldHaveObservedPhaseCount(count int) error {
	name := tc.lastCreatedCOSRName()
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	return pollForObjectMatching(tc, cosr, types.NamespacedName{Name: name}, func() bool {
		return len(cosr.Status.ObservedPhases) == count
	})
}

func (tc *testContext) observedPhaseShouldHaveIncompleteObjectCount(phaseName string, count int) error {
	name := tc.lastCreatedCOSRName()
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	return pollForObjectMatching(tc, cosr, types.NamespacedName{Name: name}, func() bool {
		for _, op := range cosr.Status.ObservedPhases {
			if op.Name == phaseName {
				return len(op.IncompleteObjects) == count
			}
		}
		return false
	})
}

func (tc *testContext) observedPhaseShouldHaveIncompleteObjectNamed(phaseName, objectName string) error {
	name := tc.lastCreatedCOSRName()
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	return pollForObjectMatching(tc, cosr, types.NamespacedName{Name: name}, func() bool {
		for _, op := range cosr.Status.ObservedPhases {
			if op.Name == phaseName {
				for _, obj := range op.IncompleteObjects {
					if obj.Name == objectName {
						return true
					}
				}
			}
		}
		return false
	})
}

func (tc *testContext) theCOSRShouldHaveCompletedAt() error {
	name := tc.lastCreatedCOSRName()
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	return pollForObjectMatching(tc, cosr, types.NamespacedName{Name: name}, func() bool {
		return cosr.Status.CompletedAt != nil
	})
}

// Single Get, not poll: preceding poll-based assertions guarantee the controller
// has reconciled. If completedAt is set at this point, that's a real bug.
func (tc *testContext) theCOSRShouldNotHaveCompletedAt() error {
	name := tc.lastCreatedCOSRName()
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	if err := tc.client.Get(context.Background(), types.NamespacedName{Name: name}, cosr); err != nil {
		return err
	}
	if cosr.Status.CompletedAt != nil {
		return fmt.Errorf("completedAt is set: %v", cosr.Status.CompletedAt)
	}
	return nil
}

func (tc *testContext) theCOSRShouldHaveNoObservedPhases() error {
	name := tc.lastCreatedCOSRName()
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	return pollForObjectMatching(tc, cosr, types.NamespacedName{Name: name}, func() bool {
		return len(cosr.Status.ObservedPhases) == 0
	})
}

func (tc *testContext) theCOSRCompletedAtIsTracked() error {
	name := tc.lastCreatedCOSRName()
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	if err := tc.client.Get(context.Background(), types.NamespacedName{Name: name}, cosr); err != nil {
		return err
	}
	if cosr.Status.CompletedAt == nil {
		return fmt.Errorf("completedAt is not set")
	}
	tc.trackedCompletedAt = cosr.Status.CompletedAt
	return nil
}

// Single Get, not poll: preceding poll-based assertions guarantee the controller
// has reconciled. completedAt and conditions are set in the same status update,
// so if the condition has changed, completedAt is already in its final state.
func (tc *testContext) theCOSRCompletedAtShouldBePreserved() error {
	if tc.trackedCompletedAt == nil {
		return fmt.Errorf("no tracked completedAt")
	}
	name := tc.lastCreatedCOSRName()
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	if err := tc.client.Get(context.Background(), types.NamespacedName{Name: name}, cosr); err != nil {
		return err
	}
	if cosr.Status.CompletedAt == nil {
		return fmt.Errorf("completedAt is nil")
	}
	if !cosr.Status.CompletedAt.Equal(tc.trackedCompletedAt) {
		return fmt.Errorf("completedAt changed from %s to %s", tc.trackedCompletedAt, cosr.Status.CompletedAt)
	}
	return nil
}

func (tc *testContext) theCODShouldBeAvailable(codName string) error {
	return tc.theCODShouldHaveConditionWithReason(codName, "Available", "True", "Available")
}

func (tc *testContext) theCODShouldHaveConditionWithReason(codName, condType, status, reason string) error {
	return tc.pollForCODConditionWithReason(context.Background(), tc.codFullName(codName), condType, metav1.ConditionStatus(status), reason)
}
