package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cucumber/godog"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func registerActionSteps(sc *godog.ScenarioContext, tc *testContext) {
	sc.Step(`^the COS is created( and becomes Available)?$`, tc.theCOSIsCreated)
	sc.Step(`^the ConfigMap "([^"]*)" is deleted$`, tc.theConfigMapIsDeleted)
	sc.Step(`^the COS lifecycleState is set to "([^"]*)"$`, tc.theCOSLifecycleStateIsSetTo)
	sc.Step(`^setting the COS lifecycleState to "([^"]*)" should fail$`, tc.settingCOSLifecycleStateShouldFail)
	sc.Step(`^revision (\d+) is archived$`, tc.revisionIsArchived)
	sc.Step(`^creating the COS should fail$`, tc.creatingTheCOSShouldFail)
	sc.Step(`^creating a COS with zero phases should fail$`, tc.creatingCOSWithZeroPhasesShouldFail)
	sc.Step(`^creating a COS with a phase with zero objects should fail$`, tc.creatingCOSWithZeroObjectsShouldFail)
	sc.Step(`^updating the COS (group|revision|phases|collisionProtection) should fail$`, tc.updatingCOSFieldShouldFail)
	sc.Step(`^creating a COS with revision 0 should fail$`, tc.creatingCOSWithRevisionZeroShouldFail)
	sc.Step(`^creating a COS with unset lifecycleState should fail$`, tc.creatingCOSWithUnsetLifecycleStateShouldFail)
	sc.Step(`^creating a COS with unknown lifecycleState should fail$`, tc.creatingCOSWithUnknownLifecycleStateShouldFail)
	sc.Step(`^creating a COS with a group name of exactly 52 characters should succeed$`, tc.creatingCOSWithExact52CharGroupShouldSucceed)
	sc.Step(`^creating a COS with a group name longer than 52 characters should fail$`, tc.creatingCOSWithLongGroupShouldFail)
	sc.Step(`^the COS is deleted with cascade (foreground|background|orphan)$`, tc.theCOSIsDeletedWithCascade)
	sc.Step(`^the CRD "([^"]*)" is deleted$`, tc.theCRDIsDeleted)
	sc.Step(`^the gate on ConfigMap "([^"]*)" is (opened|closed)$`, tc.theConfigMapGateIsSetTo)
	sc.Step(`^a resource is patched with:$`, tc.aResourceIsPatched)

	sc.Step(`^an object is created with:$`, tc.anObjectIsCreatedWith)
	sc.Step(`^a finalizer "([^"]*)" is added to ConfigMap "([^"]*)"$`, tc.aFinalizerIsAddedToConfigMap)
	sc.Step(`^the finalizer "([^"]*)" is removed from ConfigMap "([^"]*)"$`, tc.theFinalizerIsRemovedFromConfigMap)
	sc.Step(`^the COS is deleted$`, tc.theCOSIsDeleted)
	sc.Step(`^the COS with group "([^"]*)" and revision (\d+) lifecycleState is set to "([^"]*)"$`, tc.theCOSInGroupLifecycleStateIsSetTo)
	sc.Step(`^the COS with group "([^"]*)" and revision (\d+) is deleted$`, tc.theCOSInGroupIsDeleted)

	// ClusterObjectSlice action steps
	sc.Step(`^the ClusterObjectSlice "([^"]*)" is deleted$`, tc.theSliceIsDeleted)
	sc.Step(`^the ClusterObjectSlice "([^"]*)" is recreated with a ConfigMap "([^"]*)" with data key "([^"]*)" value "([^"]*)"$`, tc.theSliceIsRecreatedWithDifferentContent)

	// COD action steps
	sc.Step(`^the COD is created$`, tc.theCODIsCreated)
	sc.Step(`^the COD template spec is updated with a ConfigMap "([^"]*)" in phase "([^"]*)"$`, tc.theCODTemplateSpecIsUpdated)
	sc.Step(`^the COD template spec is updated with a gated ConfigMap "([^"]*)" in phase "([^"]*)"$`, tc.theCODTemplateSpecIsUpdatedWithGatedConfigMap)
	sc.Step(`^the COD template label "([^"]*)" is updated to "([^"]*)"$`, tc.theCODTemplateLabelIsUpdated)
	sc.Step(`^the COD "([^"]*)" is deleted$`, tc.theCODIsDeleted)
	sc.Step(`^the COD "([^"]*)" is deleted with cascade (orphan)$`, tc.theCODIsDeletedWithCascade)
	sc.Step(`^the COD "([^"]*)" label "([^"]*)" is set to "([^"]*)"$`, tc.theCODLabelIsSetTo)
	sc.Step(`^the COD "([^"]*)" revisionHistoryLimit is set to (\d+)$`, tc.theCOSevisionHistoryLimitIsSetTo)
	sc.Step(`^creating a COD with a name of exactly 52 characters should succeed$`, tc.creatingCODWithExact52CharNameShouldSucceed)
	sc.Step(`^creating a COD with a name longer than 52 characters should fail$`, tc.creatingCODWithLongNameShouldFail)
}

func (tc *testContext) theCOSIsCreated(andBecomesAvailable string) error {
	if err := tc.createCOS(context.Background()); err != nil {
		return err
	}
	if andBecomesAvailable != "" {
		return tc.pollForCOSCondition(context.Background(), tc.lastCreatedCOSName(), "Available", metav1.ConditionTrue)
	}
	return nil
}

func (tc *testContext) theConfigMapIsDeleted(name string) error {
	return deleteObject[corev1.ConfigMap](tc, types.NamespacedName{Namespace: tc.namespace, Name: name})
}

func (tc *testContext) theCOSIsDeletedWithCascade(cascade string) error {
	policy := cascadePolicy(cascade)
	return deleteObject[orbv1alpha1.ClusterObjectSet](tc, types.NamespacedName{Name: tc.lastCreatedCOSName()}, &client.DeleteOptions{
		PropagationPolicy: &policy,
	})
}

func cascadePolicy(cascade string) metav1.DeletionPropagation {
	policies := map[string]metav1.DeletionPropagation{
		"foreground": metav1.DeletePropagationForeground,
		"background": metav1.DeletePropagationBackground,
		"orphan":     metav1.DeletePropagationOrphan,
	}
	return policies[cascade]
}

func (tc *testContext) theCRDIsDeleted(name string) error {
	return deleteObject[apiextensionsv1.CustomResourceDefinition](tc, types.NamespacedName{Name: name + "." + tc.namespace + ".e2e.orb.dev"})
}

func (tc *testContext) theCOSLifecycleStateIsSetTo(state string) error {
	name := tc.lastCreatedCOSName()
	cos := &orbv1alpha1.ClusterObjectSet{}
	if err := tc.client.Get(context.Background(), types.NamespacedName{Name: name}, cos); err != nil {
		return err
	}
	cos.Spec.LifecycleState = orbv1alpha1.LifecycleState(state)
	return tc.client.Update(context.Background(), cos)
}

func (tc *testContext) settingCOSLifecycleStateShouldFail(state string) error {
	return expectError(tc.theCOSLifecycleStateIsSetTo(state), fmt.Sprintf("setting lifecycleState to %q", state))
}

func (tc *testContext) creatingTheCOSShouldFail() error {
	return expectError(tc.createCOS(context.Background()), "COS creation")
}

func (tc *testContext) creatingCOSWithZeroPhasesShouldFail() error {
	tc.resetCOSBuilder("zero-phases", 1)
	return expectError(tc.createCOS(context.Background()), "COS with zero phases")
}

func (tc *testContext) creatingCOSWithZeroObjectsShouldFail() error {
	tc.resetCOSBuilder("zero-objects", 1)
	tc.addPhase("empty")
	return expectError(tc.createCOS(context.Background()), "COS with zero objects in phase")
}

func (tc *testContext) updatingCOSFieldShouldFail(field string) error {
	name := tc.lastCreatedCOSName()
	cos := &orbv1alpha1.ClusterObjectSet{}
	if err := tc.client.Get(context.Background(), types.NamespacedName{Name: name}, cos); err != nil {
		return err
	}
	switch field {
	case "group":
		cos.Spec.Group = cos.Spec.Group + "-changed"
	case "revision":
		cos.Spec.Revision = cos.Spec.Revision + 1
	case "phases":
		cos.Spec.Phases = append(cos.Spec.Phases, orbv1alpha1.Phase{Name: "injected"})
	case "collisionProtection":
		cp := orbv1alpha1.CollisionProtectionNone
		cos.Spec.CollisionProtection = &cp
	}
	return expectError(tc.client.Update(context.Background(), cos), fmt.Sprintf("updating %s", field))
}

func (tc *testContext) creatingCOSWithRevisionZeroShouldFail() error {
	tc.resetCOSBuilder("rev-zero", 0)
	tc.addPhase("install")
	tc.addObjectToPhase(newConfigMap("cm-rev-zero", tc.namespace))
	return expectError(tc.createCOS(context.Background()), "COS with revision 0")
}

func (tc *testContext) creatingCOSWithUnsetLifecycleStateShouldFail() error {
	tc.resetCOSBuilder("lcs-unset", 1)
	tc.addPhase("install")
	tc.addObjectToPhase(newConfigMap("cm-lcs-unset", tc.namespace))
	cos := tc.buildCOS()
	cos.Spec.LifecycleState = ""
	return expectError(tc.client.Create(context.Background(), cos), "COS with unset lifecycleState")
}

func (tc *testContext) creatingCOSWithUnknownLifecycleStateShouldFail() error {
	tc.resetCOSBuilder("lcs-unknown", 1)
	tc.addPhase("install")
	tc.addObjectToPhase(newConfigMap("cm-lcs-unknown", tc.namespace))
	cos := tc.buildCOS()
	cos.Spec.LifecycleState = "Unknown"
	return expectError(tc.client.Create(context.Background(), cos), "COS with unknown lifecycleState")
}

func (tc *testContext) cosGroupOfLength(n int) string {
	prefix := tc.namespace + "-"
	pad := n - len(prefix)
	if pad < 0 {
		pad = 0
	}
	return prefix + strings.Repeat("a", pad)
}

func (tc *testContext) creatingCOSWithExact52CharGroupShouldSucceed() error {
	tc.resetCOSBuilder("x", 1)
	tc.cos.group = tc.cosGroupOfLength(52)
	tc.addPhase("install")
	tc.addObjectToPhase(newConfigMap("cm-exact-group", tc.namespace))
	return tc.createCOS(context.Background())
}

func (tc *testContext) creatingCOSWithLongGroupShouldFail() error {
	tc.resetCOSBuilder("x", 1)
	tc.cos.group = tc.cosGroupOfLength(53)
	tc.addPhase("install")
	tc.addObjectToPhase(newConfigMap("cm-long-group", tc.namespace))
	return expectError(tc.createCOS(context.Background()), "COS with group longer than 52 characters")
}

func (tc *testContext) theConfigMapGateIsSetTo(name, state string) error {
	value := "open"
	if state == "closed" {
		value = "closed"
	}
	return pollMutateUpdate[corev1.ConfigMap](tc, types.NamespacedName{Namespace: tc.namespace, Name: name}, func(cm *corev1.ConfigMap) {
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		cm.Data["gate"] = value
	})
}

func (tc *testContext) aResourceIsPatched(doc *godog.DocString) error {
	content := strings.ReplaceAll(doc.Content, "${NAMESPACE}", tc.namespace)
	obj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(content), &obj.Object); err != nil {
		return fmt.Errorf("parsing patch YAML: %w", err)
	}
	return tc.client.Apply(context.Background(), client.ApplyConfigurationFromUnstructured(obj), client.FieldOwner("e2e-test"), client.ForceOwnership)
}

func (tc *testContext) anObjectIsCreatedWith(doc *godog.DocString) error {
	content := strings.ReplaceAll(doc.Content, "${NAMESPACE}", tc.namespace)
	obj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(content), &obj.Object); err != nil {
		return fmt.Errorf("parsing YAML: %w", err)
	}
	if err := tc.client.Create(context.Background(), obj); err != nil {
		return err
	}
	tc.createdObjects = append(tc.createdObjects, metav1.PartialObjectMetadata{
		TypeMeta: metav1.TypeMeta{
			APIVersion: obj.GetAPIVersion(),
			Kind:       obj.GetKind(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
		},
	})
	return nil
}

func (tc *testContext) aFinalizerIsAddedToConfigMap(finalizer, name string) error {
	return pollMutateUpdate[corev1.ConfigMap](tc, types.NamespacedName{Namespace: tc.namespace, Name: name}, func(cm *corev1.ConfigMap) {
		for _, f := range cm.Finalizers {
			if f == finalizer {
				return
			}
		}
		cm.Finalizers = append(cm.Finalizers, finalizer)
	})
}

func (tc *testContext) theFinalizerIsRemovedFromConfigMap(finalizer, name string) error {
	return pollMutateUpdate[corev1.ConfigMap](tc, types.NamespacedName{Namespace: tc.namespace, Name: name}, func(cm *corev1.ConfigMap) {
		var filtered []string
		for _, f := range cm.Finalizers {
			if f != finalizer {
				filtered = append(filtered, f)
			}
		}
		cm.Finalizers = filtered
	})
}

func (tc *testContext) theCOSIsDeleted() error {
	return deleteObject[orbv1alpha1.ClusterObjectSet](tc, types.NamespacedName{Name: tc.lastCreatedCOSName()})
}

func (tc *testContext) theCOSInGroupLifecycleStateIsSetTo(group string, revision uint32, state string) error {
	return pollMutateUpdate[orbv1alpha1.ClusterObjectSet](tc, types.NamespacedName{Name: tc.cosName(group, revision)}, func(cos *orbv1alpha1.ClusterObjectSet) {
		cos.Spec.LifecycleState = orbv1alpha1.LifecycleState(state)
	})
}

func (tc *testContext) theCOSInGroupIsDeleted(group string, revision uint32) error {
	return deleteObject[orbv1alpha1.ClusterObjectSet](tc, types.NamespacedName{Name: tc.cosName(group, revision)})
}

func (tc *testContext) revisionIsArchived(revision uint32) error {
	for name, cos := range tc.coss {
		if cos.Spec.Revision == revision {
			latest := &orbv1alpha1.ClusterObjectSet{}
			if err := tc.client.Get(context.Background(), types.NamespacedName{Name: name}, latest); err != nil {
				return err
			}
			latest.Spec.LifecycleState = orbv1alpha1.LifecycleStateArchived
			return tc.client.Update(context.Background(), latest)
		}
	}
	return fmt.Errorf("revision %d not found", revision)
}

func (tc *testContext) theCODIsCreated() error {
	return tc.createCOD(context.Background())
}

func (tc *testContext) theCODTemplateSpecIsUpdated(cmName, phaseName string) error {
	return pollMutateUpdate[orbv1alpha1.ClusterObjectDeployment](tc, types.NamespacedName{Name: tc.lastCreatedCODName()}, func(cod *orbv1alpha1.ClusterObjectDeployment) {
		cod.Spec.Template.Spec = orbv1alpha1.ClusterObjectDeploymentTemplateSpec{
			Phases: []orbv1alpha1.Phase{{
				Name:    phaseName,
				Objects: []orbv1alpha1.PhaseObject{newGatedConfigMapPhaseObject(cmName, tc.namespace, false)},
			}},
		}
	})
}

func (tc *testContext) theCODTemplateSpecIsUpdatedWithGatedConfigMap(cmName, phaseName string) error {
	return pollMutateUpdate[orbv1alpha1.ClusterObjectDeployment](tc, types.NamespacedName{Name: tc.lastCreatedCODName()}, func(cod *orbv1alpha1.ClusterObjectDeployment) {
		cod.Spec.Template.Spec = orbv1alpha1.ClusterObjectDeploymentTemplateSpec{
			Phases: []orbv1alpha1.Phase{{
				Name:    phaseName,
				Objects: []orbv1alpha1.PhaseObject{newGatedConfigMapPhaseObject(cmName, tc.namespace, true)},
			}},
		}
	})
}

func (tc *testContext) theCODTemplateLabelIsUpdated(key, value string) error {
	return pollMutateUpdate[orbv1alpha1.ClusterObjectDeployment](tc, types.NamespacedName{Name: tc.lastCreatedCODName()}, func(cod *orbv1alpha1.ClusterObjectDeployment) {
		if cod.Spec.Template.Metadata.Labels == nil {
			cod.Spec.Template.Metadata.Labels = make(map[string]string)
		}
		cod.Spec.Template.Metadata.Labels[key] = value
	})
}

func (tc *testContext) theCODLabelIsSetTo(codName, key, value string) error {
	return pollMutateUpdate[orbv1alpha1.ClusterObjectDeployment](tc, types.NamespacedName{Name: tc.codFullName(codName)}, func(cod *orbv1alpha1.ClusterObjectDeployment) {
		if cod.Labels == nil {
			cod.Labels = make(map[string]string)
		}
		cod.Labels[key] = value
	})
}

func (tc *testContext) theCOSevisionHistoryLimitIsSetTo(codName string, limit int32) error {
	return pollMutateUpdate[orbv1alpha1.ClusterObjectDeployment](tc, types.NamespacedName{Name: tc.codFullName(codName)}, func(cod *orbv1alpha1.ClusterObjectDeployment) {
		cod.Spec.RevisionHistoryLimit = &limit
	})
}

func (tc *testContext) codNameOfLength(n int) string {
	prefix := tc.namespace + "-"
	pad := n - len(prefix)
	if pad < 0 {
		pad = 0
	}
	return prefix + strings.Repeat("b", pad)
}

func (tc *testContext) creatingCODWithExact52CharNameShouldSucceed() error {
	tc.resetCODBuilder("x")
	tc.cod.name = tc.codNameOfLength(52)
	tc.addPhase("install")
	tc.addObjectToPhase(newConfigMap("cm-exact-name", tc.namespace))
	return tc.createCOD(context.Background())
}

func (tc *testContext) creatingCODWithLongNameShouldFail() error {
	tc.resetCODBuilder("x")
	tc.cod.name = tc.codNameOfLength(53)
	tc.addPhase("install")
	tc.addObjectToPhase(newConfigMap("cm-long-name", tc.namespace))
	return expectError(tc.createCOD(context.Background()), "COD with name longer than 52 characters")
}

func (tc *testContext) theCODIsDeleted(name string) error {
	return deleteObject[orbv1alpha1.ClusterObjectDeployment](tc, types.NamespacedName{Name: tc.codFullName(name)})
}

func (tc *testContext) theCODIsDeletedWithCascade(name, cascade string) error {
	policy := cascadePolicy(cascade)
	return deleteObject[orbv1alpha1.ClusterObjectDeployment](tc, types.NamespacedName{Name: tc.codFullName(name)}, &client.DeleteOptions{
		PropagationPolicy: &policy,
	})
}

func (tc *testContext) theSliceIsDeleted(sliceName string) error {
	fullName := tc.namespace + "-" + sliceName
	if err := deleteObject[orbv1alpha1.ClusterObjectSlice](tc, types.NamespacedName{Name: fullName}); err != nil {
		return err
	}
	return tc.pollForObjectAbsence(context.Background(), types.NamespacedName{Name: fullName}, &orbv1alpha1.ClusterObjectSlice{})
}

func (tc *testContext) theSliceIsRecreatedWithDifferentContent(sliceName, cmName, key, value string) error {
	fullName := tc.namespace + "-" + sliceName
	cm := newConfigMapWithData(cmName, tc.namespace, map[string]string{key: value})
	raw, err := json.Marshal(cm)
	if err != nil {
		return fmt.Errorf("marshalling ConfigMap: %w", err)
	}
	return tc.createSlice(context.Background(), fullName, []orbv1alpha1.SliceObject{{
		ObjectKey: orbv1alpha1.ObjectKey{
			APIVersion: "v1",
			Kind:       "ConfigMap",
			Name:       cmName,
			Namespace:  tc.namespace,
		},
		Content: raw,
	}})
}
