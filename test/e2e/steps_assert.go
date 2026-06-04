package e2e

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cucumber/godog"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func registerAssertSteps(sc *godog.ScenarioContext, tc *testContext) {
	sc.Step(`^the ConfigMap "([^"]*)" should exist$`, tc.theConfigMapShouldExist)
	sc.Step(`^the ConfigMap "([^"]*)" should not exist$`, tc.theConfigMapShouldNotExist)
	sc.Step(`^the ConfigMap "([^"]*)" should be recreated$`, tc.theConfigMapShouldBeRecreated)
	sc.Step(`^the ConfigMap "([^"]*)" UID is tracked$`, tc.theConfigMapUIDisTracked)
	sc.Step(`^the ConfigMap "([^"]*)" should not have been deleted and recreated$`, tc.theConfigMapShouldNotHaveBeenRecreated)
	sc.Step(`^the ConfigMap "([^"]*)" should have data key "([^"]*)" with value "([^"]*)"$`, tc.theConfigMapShouldHaveDataKeyValue)
	sc.Step(`^the ConfigMap "([^"]*)" should not have data key "([^"]*)"$`, tc.theConfigMapShouldNotHaveDataKey)
	sc.Step(`^the CRD "([^"]*)" should exist$`, tc.theCRDShouldExist)
	sc.Step(`^the "([^"]*)" named "([^"]*)" should exist$`, tc.theCRShouldExist)
	sc.Step(`^the ConfigMap "([^"]*)" should have an owner reference$`, tc.theConfigMapShouldHaveOwnerRef)
	sc.Step(`^the ConfigMap "([^"]*)" should not have an owner reference$`, tc.theConfigMapShouldNotHaveOwnerRef)
	sc.Step(`^the COSR should not exist$`, tc.theCOSRShouldNotExist)
	sc.Step(`^the COSR should have condition "([^"]*)" with status "([^"]*)"$`, tc.theCOSRShouldHaveCondition)
	sc.Step(`^the COSR should have condition "([^"]*)" with status "([^"]*)" and reason "([^"]*)"$`, tc.theCOSRShouldHaveConditionWithReason)
	sc.Step(`^the COSR in group "([^"]*)" revision (\d+) should have condition "([^"]*)" with status "([^"]*)"$`, tc.theCOSRInGroupShouldHaveCondition)
	sc.Step(`^revision (\d+) should have condition "([^"]*)" with status "([^"]*)" and reason "([^"]*)"$`, tc.revisionShouldHaveConditionWithReason)

	// COS assert steps
	sc.Step(`^a COSR should exist with group "([^"]*)" and revision (\d+)$`, tc.aCOSRShouldExistWithGroupAndRevision)
	sc.Step(`^the COSR with group "([^"]*)" and revision (\d+) should have lifecycleState "([^"]*)"$`, tc.cosrShouldHaveLifecycleState)
	sc.Step(`^the COSR with group "([^"]*)" and revision (\d+) should have collisionProtection "([^"]*)"$`, tc.cosrShouldHaveCollisionProtection)
	sc.Step(`^the COSR with group "([^"]*)" and revision (\d+) should have (\d+) phases$`, tc.cosrShouldHavePhaseCount)
	sc.Step(`^the COSR with group "([^"]*)" and revision (\d+) should have label "([^"]*)" with value "([^"]*)"$`, tc.cosrShouldHaveLabel)
	sc.Step(`^the COSR with group "([^"]*)" and revision (\d+) should have annotation "([^"]*)" with value "([^"]*)"$`, tc.cosrShouldHaveAnnotation)
	sc.Step(`^the COSR with group "([^"]*)" and revision (\d+) should have a controller owner reference to COS "([^"]*)"$`, tc.cosrShouldHaveControllerOwnerRef)
	sc.Step(`^the COSR with group "([^"]*)" and revision (\d+) should not exist$`, tc.cosrShouldNotExist)
	sc.Step(`^the COSR with group "([^"]*)" and revision (\d+) should not have an owner reference$`, tc.cosrShouldNotHaveOwnerRef)
	sc.Step(`^the COSR count for COS "([^"]*)" should be (\d+)$`, tc.cosrCountForCOSShouldBe)
	sc.Step(`^the COS "([^"]*)" should have condition "([^"]*)" with status "([^"]*)" and reason "([^"]*)"$`, tc.theCOSShouldHaveConditionWithReason)
	sc.Step(`^the COS "([^"]*)" should become available without becoming unavailable$`, tc.theCOSShouldBecomeAvailableWithoutBecomingUnavailable)
	sc.Step(`^the COS "([^"]*)" should have active revision (\d+)$`, tc.theCOSShouldHaveActiveRevision)
	sc.Step(`^the stamped COSR spec for "([^"]*)" revision (\d+) should match the COS template spec$`, tc.stampedCOSRSpecShouldMatchTemplate)
}

func (tc *testContext) theConfigMapShouldExist(name string) error {
	cm := &corev1.ConfigMap{}
	return tc.pollForObject(context.Background(), types.NamespacedName{Namespace: tc.namespace, Name: name}, cm)
}

func (tc *testContext) theConfigMapShouldNotExist(name string) error {
	cm := &corev1.ConfigMap{}
	return tc.pollForObjectAbsence(context.Background(), types.NamespacedName{Namespace: tc.namespace, Name: name}, cm)
}

func (tc *testContext) theConfigMapShouldBeRecreated(name string) error {
	return tc.pollForObject(context.Background(), types.NamespacedName{Namespace: tc.namespace, Name: name}, &corev1.ConfigMap{})
}

func (tc *testContext) theConfigMapUIDisTracked(name string) error {
	cm := &corev1.ConfigMap{}
	key := types.NamespacedName{Namespace: tc.namespace, Name: name}
	if err := tc.client.Get(context.Background(), key, cm); err != nil {
		return fmt.Errorf("ConfigMap %q should exist: %w", name, err)
	}
	tc.trackedUIDs[name] = cm.UID
	return nil
}

func (tc *testContext) theConfigMapShouldNotHaveBeenRecreated(name string) error {
	cm := &corev1.ConfigMap{}
	key := types.NamespacedName{Namespace: tc.namespace, Name: name}
	if err := tc.client.Get(context.Background(), key, cm); err != nil {
		return fmt.Errorf("ConfigMap %q should exist: %w", name, err)
	}
	if tracked, ok := tc.trackedUIDs[name]; ok && cm.UID != tracked {
		return fmt.Errorf("ConfigMap %q was recreated: UID changed from %s to %s", name, tracked, cm.UID)
	}
	return nil
}

func (tc *testContext) theConfigMapShouldHaveDataKeyValue(name, key, value string) error {
	cm := &corev1.ConfigMap{}
	if err := tc.pollForObject(context.Background(), types.NamespacedName{Namespace: tc.namespace, Name: name}, cm); err != nil {
		return err
	}
	if got := cm.Data[key]; got != value {
		return fmt.Errorf("ConfigMap %q data key %q: got %q, want %q", name, key, got, value)
	}
	return nil
}

func (tc *testContext) theConfigMapShouldNotHaveDataKey(name, key string) error {
	cm := &corev1.ConfigMap{}
	if err := tc.pollForObject(context.Background(), types.NamespacedName{Namespace: tc.namespace, Name: name}, cm); err != nil {
		return err
	}
	if _, ok := cm.Data[key]; ok {
		return fmt.Errorf("ConfigMap %q should not have data key %q, but it does", name, key)
	}
	return nil
}

func (tc *testContext) theCRDShouldExist(name string) error {
	crd := &apiextensionsv1.CustomResourceDefinition{}
	return tc.pollForObject(context.Background(), types.NamespacedName{Name: name + ".e2e.orb.dev"}, crd)
}

func (tc *testContext) theCRShouldExist(crdName, crName string) error {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(crGVK(crdName))
	return tc.pollForObject(context.Background(), types.NamespacedName{Name: crName}, obj)
}

func crGVK(crdName string) schema.GroupVersionKind {
	kind := crdName[:len(crdName)-1]
	return schema.GroupVersionKind{
		Group:   "e2e.orb.dev",
		Version: "v1alpha1",
		Kind:    string(kind[0]-32) + kind[1:],
	}
}

func (tc *testContext) theConfigMapShouldHaveOwnerRef(name string) error {
	cm := &corev1.ConfigMap{}
	if err := tc.pollForObject(context.Background(), types.NamespacedName{Namespace: tc.namespace, Name: name}, cm); err != nil {
		return err
	}
	if len(cm.OwnerReferences) == 0 {
		return fmt.Errorf("ConfigMap %q has no owner references", name)
	}
	return nil
}

func (tc *testContext) theConfigMapShouldNotHaveOwnerRef(name string) error {
	cm := &corev1.ConfigMap{}
	if err := tc.pollForObject(context.Background(), types.NamespacedName{Namespace: tc.namespace, Name: name}, cm); err != nil {
		return err
	}
	if len(cm.OwnerReferences) > 0 {
		return fmt.Errorf("ConfigMap %q still has owner references: %v", name, cm.OwnerReferences)
	}
	return nil
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
	name := fmt.Sprintf("%s-%s-%d", tc.namespace, group, revision)
	return tc.pollForCOSRCondition(context.Background(), name, condType, metav1.ConditionStatus(status))
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
	name := tc.cosrName(group, revision)
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	return wait.PollUntilContextTimeout(context.Background(), pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		if err := tc.client.Get(ctx, types.NamespacedName{Name: name}, cosr); err != nil {
			return false, nil
		}
		actual := string(cosr.Spec.LifecycleState)
		if actual == "" {
			actual = "Active"
		}
		return actual == state, nil
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

func (tc *testContext) cosrShouldHaveLabel(group string, revision uint32, key, value string) error {
	cosr, err := tc.getCOSR(context.Background(), group, revision)
	if err != nil {
		return err
	}
	if got := cosr.Labels[key]; got != value {
		return fmt.Errorf("COSR %s-%d label %q: got %q, want %q", group, revision, key, got, value)
	}
	return nil
}

func (tc *testContext) cosrShouldHaveAnnotation(group string, revision uint32, key, value string) error {
	cosr, err := tc.getCOSR(context.Background(), group, revision)
	if err != nil {
		return err
	}
	if got := cosr.Annotations[key]; got != value {
		return fmt.Errorf("COSR %s-%d annotation %q: got %q, want %q", group, revision, key, got, value)
	}
	return nil
}

func (tc *testContext) cosrShouldHaveControllerOwnerRef(group string, revision uint32, cosName string) error {
	fullCOSName := tc.namespace + "-" + cosName
	name := tc.cosrName(group, revision)
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	return wait.PollUntilContextTimeout(context.Background(), pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		if err := tc.client.Get(ctx, types.NamespacedName{Name: name}, cosr); err != nil {
			return false, nil
		}
		for _, ref := range cosr.OwnerReferences {
			if ref.Kind == "ClusterObjectSet" && ref.Name == fullCOSName && ref.Controller != nil && *ref.Controller {
				return true, nil
			}
		}
		return false, nil
	})
}

func (tc *testContext) cosrShouldNotExist(group string, revision uint32) error {
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	return tc.pollForObjectAbsence(context.Background(), types.NamespacedName{Name: tc.cosrName(group, revision)}, cosr)
}

func (tc *testContext) cosrShouldNotHaveOwnerRef(group string, revision uint32) error {
	ctx := context.Background()
	name := tc.cosrName(group, revision)
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		if err := tc.client.Get(ctx, types.NamespacedName{Name: name}, cosr); err != nil {
			return false, nil
		}
		return len(cosr.OwnerReferences) == 0, nil
	})
}

func (tc *testContext) cosrCountForCOSShouldBe(cosName string, count int) error {
	fullCOSName := tc.namespace + "-" + cosName
	var list orbv1alpha1.ClusterObjectSetRevisionList
	if err := tc.client.List(context.Background(), &list); err != nil {
		return err
	}
	actual := 0
	for _, cosr := range list.Items {
		if cosr.Spec.Group == fullCOSName {
			actual++
		}
	}
	if actual != count {
		return fmt.Errorf("COSR count for COS %q: got %d, want %d", cosName, actual, count)
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
	actual := cosr.Spec.ClusterObjectSetTemplateSpec

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
		return fmt.Errorf("COSR spec does not match COS template spec:\n%s", cmp.Diff(expectedNorm, actualNorm))
	}
	return nil
}

func (tc *testContext) theCOSShouldBecomeAvailableWithoutBecomingUnavailable(cosName string) error {
	fullCOSName := tc.namespace + "-" + cosName
	cos := &orbv1alpha1.ClusterObjectSet{}
	key := types.NamespacedName{Name: fullCOSName}
	var sawUnavailable bool
	err := wait.PollUntilContextTimeout(context.Background(), pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		if err := tc.client.Get(ctx, key, cos); err != nil {
			return false, nil
		}
		for _, c := range cos.Status.Conditions {
			if c.Type != orbv1alpha1.ConditionTypeAvailable || c.ObservedGeneration != cos.Generation {
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
		return fmt.Errorf("COS %q became unavailable during rollout", cosName)
	}
	return nil
}

func (tc *testContext) theCOSShouldHaveActiveRevision(cosName string, revision uint32) error {
	fullCOSName := tc.namespace + "-" + cosName
	expectedCOSRName := fmt.Sprintf("%s-%d", fullCOSName, revision)
	cos := &orbv1alpha1.ClusterObjectSet{}
	key := types.NamespacedName{Name: fullCOSName}
	return wait.PollUntilContextTimeout(context.Background(), pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		if err := tc.client.Get(ctx, key, cos); err != nil {
			return false, nil
		}
		for _, rs := range cos.Status.ActiveRevisions {
			if rs.Name == expectedCOSRName {
				return true, nil
			}
		}
		return false, nil
	})
}

func (tc *testContext) theCOSShouldHaveConditionWithReason(cosName, condType, status, reason string) error {
	fullCOSName := tc.namespace + "-" + cosName
	return tc.pollForCOSConditionWithReason(context.Background(), fullCOSName, condType, metav1.ConditionStatus(status), reason)
}
