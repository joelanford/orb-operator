package e2e

import (
	"context"
	"fmt"

	"github.com/cucumber/godog"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

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
	return tc.pollForCondition(context.Background(), name, condType, metav1.ConditionStatus(status))
}

func (tc *testContext) theCOSRInGroupShouldHaveCondition(group string, revision uint32, condType, status string) error {
	name := fmt.Sprintf("%s-%s-%d", tc.namespace, group, revision)
	return tc.pollForCondition(context.Background(), name, condType, metav1.ConditionStatus(status))
}

func (tc *testContext) theCOSRShouldHaveConditionWithReason(condType, status, reason string) error {
	name := tc.lastCreatedCOSRName()
	return tc.pollForConditionWithReason(context.Background(), name, condType, metav1.ConditionStatus(status), reason)
}

func (tc *testContext) revisionShouldHaveConditionWithReason(revision uint32, condType, status, reason string) error {
	for name, cosr := range tc.cosrs {
		if cosr.Spec.Revision == revision {
			return tc.pollForConditionWithReason(context.Background(), name, condType, metav1.ConditionStatus(status), reason)
		}
	}
	return fmt.Errorf("revision %d not found", revision)
}

func (tc *testContext) pollForConditionWithReason(ctx context.Context, name, condType string, status metav1.ConditionStatus, reason string) error {
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	if err := tc.pollForCondition(ctx, name, condType, status); err != nil {
		return err
	}
	if err := tc.client.Get(ctx, types.NamespacedName{Name: name}, cosr); err != nil {
		return err
	}
	for _, c := range cosr.Status.Conditions {
		if c.Type == condType && c.Status == status && c.Reason == reason {
			return nil
		}
	}
	return fmt.Errorf("COSR %q: condition %q with status %q and reason %q not found", name, condType, status, reason)
}
