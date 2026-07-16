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
	sc.Step(`^the ConfigMap "([^"]*)" should have a controller owner reference to COS with group "([^"]*)" and revision (\d+)$`, tc.theConfigMapShouldBeOwnedByCOS)
	sc.Step(`^the COS should not exist$`, tc.theCOSShouldNotExist)
	sc.Step(`^the COS should have condition "([^"]*)" with status "([^"]*)"$`, tc.theCOSShouldHaveCondition)
	sc.Step(`^the COS should have condition "([^"]*)" with status "([^"]*)" and reason "([^"]*)"(?: and message containing "([^"]*)")?$`, tc.theCOSShouldHaveConditionWithReason)
	sc.Step(`^the COS in group "([^"]*)" revision (\d+) should have condition "([^"]*)" with status "([^"]*)"$`, tc.theCOSInGroupShouldHaveCondition)
	sc.Step(`^revision (\d+) should have condition "([^"]*)" with status "([^"]*)" and reason "([^"]*)"$`, tc.revisionShouldHaveConditionWithReason)
	sc.Step(`^revision (\d+) should have observed phase "([^"]*)" with status "([^"]*)"$`, tc.revisionShouldHaveObservedPhase)

	// Phase status assert steps
	sc.Step(`^the COS should have observed phase "([^"]*)" with status "([^"]*)"(?: and message "([^"]*)")?$`, tc.theCOSShouldHaveObservedPhase)
	sc.Step(`^the COS should have (\d+) observed phases$`, tc.theCOSShouldHaveObservedPhaseCount)
	sc.Step(`^observed phase "([^"]*)" should have object counts total:(\d+)/present:(\d+)/synced:(\d+)/available:(\d+)$`, tc.observedPhaseShouldHaveObjectCounts)
	sc.Step(`^observed phase "([^"]*)" should have object details for "([^"]*)"$`, tc.observedPhaseShouldHaveObjectDetailsFor)
	sc.Step(`^the COS should have object counts total:(\d+)/present:(\d+)/synced:(\d+)/available:(\d+)$`, tc.theCOSShouldHaveObjectCounts)
	sc.Step(`^the COS should have no observed phases$`, tc.theCOSShouldHaveNoObservedPhases)
	sc.Step(`^the COS should have completedAt set$`, tc.theCOSShouldHaveCompletedAt)
	sc.Step(`^the COS should not have completedAt set$`, tc.theCOSShouldNotHaveCompletedAt)
	sc.Step(`^the COS completedAt should be preserved$`, tc.theCOSCompletedAtShouldBePreserved)
	sc.Step(`^the COS completedAt is tracked$`, tc.theCOSCompletedAtIsTracked)

	// COD assert steps
	sc.Step(`^a COS should exist with group "([^"]*)" and revision (\d+)$`, tc.aCOSShouldExistWithGroupAndRevision)
	sc.Step(`^the COS with group "([^"]*)" and revision (\d+) should have lifecycleState "([^"]*)"$`, tc.cosShouldHaveLifecycleState)
	sc.Step(`^the COS with group "([^"]*)" and revision (\d+) should have collisionProtection "([^"]*)"$`, tc.cosShouldHaveCollisionProtection)
	sc.Step(`^the COS with group "([^"]*)" and revision (\d+) should have (\d+) phases$`, tc.cosShouldHavePhaseCount)
	sc.Step(`^the COS with group "([^"]*)" and revision (\d+) should have (label|annotation) "([^"]*)" with value "([^"]*)"$`, tc.cosShouldHaveMetadata)
	sc.Step(`^the COS with group "([^"]*)" and revision (\d+) should not have label "([^"]*)" with value "([^"]*)"$`, tc.cosShouldNotHaveLabelValue)
	sc.Step(`^the COS with group "([^"]*)" and revision (\d+) should be named "([^"]*)"$`, tc.cosShouldBeNamed)
	sc.Step(`^the COS with group "([^"]*)" and revision (\d+) should have a controller owner reference to COD "([^"]*)"$`, tc.cosShouldHaveControllerOwnerRef)
	sc.Step(`^the COS with group "([^"]*)" and revision (\d+) should not exist$`, tc.cosShouldNotExist)
	sc.Step(`^the COS with group "([^"]*)" and revision (\d+) should not have an owner reference$`, tc.cosShouldNotHaveOwnerRef)
	sc.Step(`^the COS with group "([^"]*)" and revision (\d+) should not have finalizer "([^"]*)"$`, tc.cosShouldNotHaveFinalizer)
	sc.Step(`^the COS count for COD "([^"]*)" should be (\d+)$`, tc.cosCountForCODShouldBe)
	sc.Step(`^the COD "([^"]*)" should be Available$`, tc.theCODShouldBeAvailable)
	sc.Step(`^the COD "([^"]*)" should have condition "([^"]*)" with status "([^"]*)" and reason "([^"]*)"$`, tc.theCODShouldHaveConditionWithReason)
	sc.Step(`^the COD "([^"]*)" should have object counts total:(\d+)/present:(\d+)/synced:(\d+)/available:(\d+)$`, tc.theCODShouldHaveObjectCounts)
	sc.Step(`^the COD "([^"]*)" should become available without becoming unavailable$`, tc.theCODShouldBecomeAvailableWithoutBecomingUnavailable)
	sc.Step(`^the COD "([^"]*)" should have active revision (\d+)$`, tc.theCODShouldHaveActiveRevision)
	sc.Step(`^the stamped COS spec for "([^"]*)" revision (\d+) should match the COD template spec$`, tc.stampedCOSSpecShouldMatchTemplate)
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

func (tc *testContext) theConfigMapShouldBeOwnedByCOS(name, group string, revision uint32) error {
	expectedOwner := tc.cosName(group, revision)
	cm := &corev1.ConfigMap{}
	return pollForObjectMatching(tc, cm, types.NamespacedName{Namespace: tc.namespace, Name: name}, func() bool {
		ref := metav1.GetControllerOf(cm)
		return ref != nil && ref.Kind == "ClusterObjectSet" && ref.Name == expectedOwner
	})
}

func (tc *testContext) theCOSShouldNotExist() error {
	name := tc.lastCreatedCOSName()
	cos := &orbv1alpha1.ClusterObjectSet{}
	return tc.pollForObjectAbsence(context.Background(), types.NamespacedName{Name: name}, cos)
}

func (tc *testContext) theCOSShouldHaveCondition(condType, status string) error {
	name := tc.lastCreatedCOSName()
	return tc.pollForCOSCondition(context.Background(), name, condType, metav1.ConditionStatus(status))
}

func (tc *testContext) theCOSInGroupShouldHaveCondition(group string, revision uint32, condType, status string) error {
	return tc.pollForCOSCondition(context.Background(), tc.cosName(group, revision), condType, metav1.ConditionStatus(status))
}

func (tc *testContext) theCOSShouldHaveConditionWithReason(condType, status, reason, messageSubstring string) error {
	name := tc.lastCreatedCOSName()
	return tc.pollForConditionWithReasonMessageOn(
		context.Background(),
		&orbv1alpha1.ClusterObjectSet{},
		types.NamespacedName{Name: name},
		cosConditions, condType, metav1.ConditionStatus(status), reason, messageSubstring,
	)
}

func (tc *testContext) revisionShouldHaveConditionWithReason(revision uint32, condType, status, reason string) error {
	for name, cos := range tc.coss {
		if cos.Spec.Revision == revision {
			return tc.pollForConditionWithReasonOn(
				context.Background(),
				&orbv1alpha1.ClusterObjectSet{},
				types.NamespacedName{Name: name},
				cosConditions, condType, metav1.ConditionStatus(status), reason,
			)
		}
	}
	return fmt.Errorf("revision %d not found", revision)
}

func (tc *testContext) revisionShouldHaveObservedPhase(revision uint32, phaseName, status string) error {
	for name, cos := range tc.coss {
		if cos.Spec.Revision == revision {
			obj := &orbv1alpha1.ClusterObjectSet{}
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

func (tc *testContext) cosName(group string, revision uint32) string {
	return fmt.Sprintf("%s-%s-%d", tc.namespace, group, revision)
}

func (tc *testContext) getCOS(ctx context.Context, group string, revision uint32) (*orbv1alpha1.ClusterObjectSet, error) {
	cos := &orbv1alpha1.ClusterObjectSet{}
	if err := tc.pollForObject(ctx, types.NamespacedName{Name: tc.cosName(group, revision)}, cos); err != nil {
		return nil, fmt.Errorf("COS %s-%d not found: %w", group, revision, err)
	}
	return cos, nil
}

func (tc *testContext) aCOSShouldExistWithGroupAndRevision(group string, revision uint32) error {
	_, err := tc.getCOS(context.Background(), group, revision)
	return err
}

func (tc *testContext) cosShouldHaveLifecycleState(group string, revision uint32, state string) error {
	cos := &orbv1alpha1.ClusterObjectSet{}
	return pollForObjectMatching(tc, cos, types.NamespacedName{Name: tc.cosName(group, revision)}, func() bool {
		actual := string(cos.Spec.LifecycleState)
		if actual == "" {
			actual = "Active"
		}
		return actual == state
	})
}

func (tc *testContext) cosShouldHaveCollisionProtection(group string, revision uint32, cp string) error {
	cos, err := tc.getCOS(context.Background(), group, revision)
	if err != nil {
		return err
	}
	if cos.Spec.CollisionProtection == nil {
		return fmt.Errorf("COS %s-%d collisionProtection is nil, want %q", group, revision, cp)
	}
	if string(*cos.Spec.CollisionProtection) != cp {
		return fmt.Errorf("COS %s-%d collisionProtection: got %q, want %q", group, revision, *cos.Spec.CollisionProtection, cp)
	}
	return nil
}

func (tc *testContext) cosShouldHavePhaseCount(group string, revision uint32, count int) error {
	cos, err := tc.getCOS(context.Background(), group, revision)
	if err != nil {
		return err
	}
	if len(cos.Spec.Phases) != count {
		return fmt.Errorf("COS %s-%d phases: got %d, want %d", group, revision, len(cos.Spec.Phases), count)
	}
	return nil
}

func (tc *testContext) cosShouldHaveMetadata(group string, revision uint32, kind, key, value string) error {
	cos, err := tc.getCOS(context.Background(), group, revision)
	if err != nil {
		return err
	}
	m := cos.Labels
	if kind == "annotation" {
		m = cos.Annotations
	}
	if got := m[key]; got != value {
		return fmt.Errorf("COS %s-%d %s %q: got %q, want %q", group, revision, kind, key, got, value)
	}
	return nil
}

func (tc *testContext) cosShouldNotHaveLabelValue(group string, revision uint32, key, value string) error {
	cos, err := tc.getCOS(context.Background(), group, revision)
	if err != nil {
		return err
	}
	if got := cos.Labels[key]; got == value {
		return fmt.Errorf("COS %s-%d label %q: got %q, should not equal %q", group, revision, key, got, value)
	}
	return nil
}

func (tc *testContext) cosShouldBeNamed(group string, revision uint32, expectedName string) error {
	cos, err := tc.getCOS(context.Background(), group, revision)
	if err != nil {
		return err
	}
	expected := tc.codFullName(expectedName)
	if cos.Name != expected {
		return fmt.Errorf("COS %s-%d name: got %q, want %q", group, revision, cos.Name, expected)
	}
	return nil
}

func (tc *testContext) cosShouldHaveControllerOwnerRef(group string, revision uint32, codName string) error {
	fullCODName := tc.codFullName(codName)
	cos := &orbv1alpha1.ClusterObjectSet{}
	return pollForObjectMatching(tc, cos, types.NamespacedName{Name: tc.cosName(group, revision)}, func() bool {
		for _, ref := range cos.OwnerReferences {
			if ref.Kind == "ClusterObjectDeployment" && ref.Name == fullCODName && ref.Controller != nil && *ref.Controller {
				return true
			}
		}
		return false
	})
}

func (tc *testContext) cosShouldNotExist(group string, revision uint32) error {
	cos := &orbv1alpha1.ClusterObjectSet{}
	return tc.pollForObjectAbsence(context.Background(), types.NamespacedName{Name: tc.cosName(group, revision)}, cos)
}

func (tc *testContext) cosShouldNotHaveOwnerRef(group string, revision uint32) error {
	cos := &orbv1alpha1.ClusterObjectSet{}
	return pollForObjectMatching(tc, cos, types.NamespacedName{Name: tc.cosName(group, revision)}, func() bool {
		return len(cos.OwnerReferences) == 0
	})
}

func (tc *testContext) cosShouldNotHaveFinalizer(group string, revision uint32, finalizer string) error {
	cos := &orbv1alpha1.ClusterObjectSet{}
	return pollForObjectMatching(tc, cos, types.NamespacedName{Name: tc.cosName(group, revision)}, func() bool {
		for _, f := range cos.Finalizers {
			if f == finalizer {
				return false
			}
		}
		return true
	})
}

func (tc *testContext) cosCountForCODShouldBe(codName string, count int) error {
	fullCODName := tc.codFullName(codName)
	var list orbv1alpha1.ClusterObjectSetList
	if err := tc.client.List(context.Background(), &list); err != nil {
		return err
	}
	actual := 0
	for _, cos := range list.Items {
		if cos.Spec.Group == fullCODName {
			actual++
		}
	}
	if actual != count {
		return fmt.Errorf("COS count for COD %q: got %d, want %d", codName, actual, count)
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

func (tc *testContext) stampedCOSSpecShouldMatchTemplate(group string, revision uint32) error {
	cos, err := tc.getCOS(context.Background(), group, revision)
	if err != nil {
		return err
	}
	expected := tc.tmpl.build()
	actual := cos.Spec.ClusterObjectDeploymentTemplateSpec

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
		return fmt.Errorf("COS spec does not match COD template spec:\n%s", cmp.Diff(expectedNorm, actualNorm))
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
	expectedCOSName := fmt.Sprintf("%s-%d", fullCODName, revision)
	cod := &orbv1alpha1.ClusterObjectDeployment{}
	return pollForObjectMatching(tc, cod, types.NamespacedName{Name: fullCODName}, func() bool {
		for _, rs := range cod.Status.ActiveRevisions {
			if rs.Name == expectedCOSName {
				return true
			}
		}
		return false
	})
}

func (tc *testContext) theCOSShouldHaveObservedPhase(phaseName, status, message string) error {
	name := tc.lastCreatedCOSName()
	cos := &orbv1alpha1.ClusterObjectSet{}
	return pollForObjectMatching(tc, cos, types.NamespacedName{Name: name}, func() bool {
		for _, op := range cos.Status.ObservedPhases {
			if op.Name == phaseName && string(op.Status) == status {
				return message == "" || op.Message == message
			}
		}
		return false
	})
}

func (tc *testContext) theCOSShouldHaveObservedPhaseCount(count int) error {
	name := tc.lastCreatedCOSName()
	cos := &orbv1alpha1.ClusterObjectSet{}
	return pollForObjectMatching(tc, cos, types.NamespacedName{Name: name}, func() bool {
		return len(cos.Status.ObservedPhases) == count
	})
}

func (tc *testContext) observedPhaseShouldHaveObjectCounts(phaseName string, total, present, synced, available int) error {
	name := tc.lastCreatedCOSName()
	cos := &orbv1alpha1.ClusterObjectSet{}
	return pollForObjectMatching(tc, cos, types.NamespacedName{Name: name}, func() bool {
		for _, op := range cos.Status.ObservedPhases {
			if op.Name == phaseName {
				return op.ObjectCounts.Total == int64(total) &&
					op.ObjectCounts.Present == int64(present) &&
					op.ObjectCounts.Synced == int64(synced) &&
					op.ObjectCounts.Available == int64(available)
			}
		}
		return false
	})
}

func (tc *testContext) theCOSShouldHaveObjectCounts(total, present, synced, available int) error {
	name := tc.lastCreatedCOSName()
	cos := &orbv1alpha1.ClusterObjectSet{}
	return pollForObjectMatching(tc, cos, types.NamespacedName{Name: name}, func() bool {
		return cos.Status.ObjectCounts != nil &&
			cos.Status.ObjectCounts.Total == int64(total) &&
			cos.Status.ObjectCounts.Present == int64(present) &&
			cos.Status.ObjectCounts.Synced == int64(synced) &&
			cos.Status.ObjectCounts.Available == int64(available)
	})
}

func (tc *testContext) theCODShouldHaveObjectCounts(codName string, total, present, synced, available int) error {
	cod := &orbv1alpha1.ClusterObjectDeployment{}
	return pollForObjectMatching(tc, cod, types.NamespacedName{Name: tc.codFullName(codName)}, func() bool {
		return cod.Status.ObjectCounts != nil &&
			cod.Status.ObjectCounts.Total == int64(total) &&
			cod.Status.ObjectCounts.Present == int64(present) &&
			cod.Status.ObjectCounts.Synced == int64(synced) &&
			cod.Status.ObjectCounts.Available == int64(available)
	})
}

func (tc *testContext) observedPhaseShouldHaveObjectDetailsFor(phaseName, objectNames string) error {
	expected := make(map[string]struct{})
	for _, n := range strings.Split(objectNames, `","`) {
		expected[strings.Trim(n, `"`)] = struct{}{}
	}
	name := tc.lastCreatedCOSName()
	cos := &orbv1alpha1.ClusterObjectSet{}
	return pollForObjectMatching(tc, cos, types.NamespacedName{Name: name}, func() bool {
		for _, op := range cos.Status.ObservedPhases {
			if op.Name != phaseName {
				continue
			}
			if len(op.ObjectDetails) != len(expected) {
				return false
			}
			for _, obj := range op.ObjectDetails {
				if _, ok := expected[obj.Name]; !ok {
					return false
				}
			}
			return true
		}
		return false
	})
}

func (tc *testContext) theCOSShouldHaveCompletedAt() error {
	name := tc.lastCreatedCOSName()
	cos := &orbv1alpha1.ClusterObjectSet{}
	return pollForObjectMatching(tc, cos, types.NamespacedName{Name: name}, func() bool {
		return cos.Status.CompletedAt != nil
	})
}

// Single Get, not poll: preceding poll-based assertions guarantee the controller
// has reconciled. If completedAt is set at this point, that's a real bug.
func (tc *testContext) theCOSShouldNotHaveCompletedAt() error {
	name := tc.lastCreatedCOSName()
	cos := &orbv1alpha1.ClusterObjectSet{}
	if err := tc.client.Get(context.Background(), types.NamespacedName{Name: name}, cos); err != nil {
		return err
	}
	if cos.Status.CompletedAt != nil {
		return fmt.Errorf("completedAt is set: %v", cos.Status.CompletedAt)
	}
	return nil
}

func (tc *testContext) theCOSShouldHaveNoObservedPhases() error {
	name := tc.lastCreatedCOSName()
	cos := &orbv1alpha1.ClusterObjectSet{}
	return pollForObjectMatching(tc, cos, types.NamespacedName{Name: name}, func() bool {
		return len(cos.Status.ObservedPhases) == 0
	})
}

func (tc *testContext) theCOSCompletedAtIsTracked() error {
	name := tc.lastCreatedCOSName()
	cos := &orbv1alpha1.ClusterObjectSet{}
	if err := tc.client.Get(context.Background(), types.NamespacedName{Name: name}, cos); err != nil {
		return err
	}
	if cos.Status.CompletedAt == nil {
		return fmt.Errorf("completedAt is not set")
	}
	tc.trackedCompletedAt = cos.Status.CompletedAt
	return nil
}

// Single Get, not poll: preceding poll-based assertions guarantee the controller
// has reconciled. completedAt and conditions are set in the same status update,
// so if the condition has changed, completedAt is already in its final state.
func (tc *testContext) theCOSCompletedAtShouldBePreserved() error {
	if tc.trackedCompletedAt == nil {
		return fmt.Errorf("no tracked completedAt")
	}
	name := tc.lastCreatedCOSName()
	cos := &orbv1alpha1.ClusterObjectSet{}
	if err := tc.client.Get(context.Background(), types.NamespacedName{Name: name}, cos); err != nil {
		return err
	}
	if cos.Status.CompletedAt == nil {
		return fmt.Errorf("completedAt is nil")
	}
	if !cos.Status.CompletedAt.Equal(tc.trackedCompletedAt) {
		return fmt.Errorf("completedAt changed from %s to %s", tc.trackedCompletedAt, cos.Status.CompletedAt)
	}
	return nil
}

func (tc *testContext) theCODShouldBeAvailable(codName string) error {
	return tc.theCODShouldHaveConditionWithReason(codName, "Available", "True", "Available")
}

func (tc *testContext) theCODShouldHaveConditionWithReason(codName, condType, status, reason string) error {
	return tc.pollForCODConditionWithReason(context.Background(), tc.codFullName(codName), condType, metav1.ConditionStatus(status), reason)
}
