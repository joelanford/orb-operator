package e2e

import (
	"context"
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
	sc.Step(`^the COSR is created( and becomes Available)?$`, tc.theCOSRIsCreated)
	sc.Step(`^the ConfigMap "([^"]*)" is deleted$`, tc.theConfigMapIsDeleted)
	sc.Step(`^the COSR lifecycleState is set to "([^"]*)"$`, tc.theCOSRLifecycleStateIsSetTo)
	sc.Step(`^setting the COSR lifecycleState to "([^"]*)" should fail$`, tc.settingCOSRLifecycleStateShouldFail)
	sc.Step(`^revision (\d+) is archived$`, tc.revisionIsArchived)
	sc.Step(`^creating the COSR should fail$`, tc.creatingTheCOSRShouldFail)
	sc.Step(`^creating a COSR with zero phases should fail$`, tc.creatingCOSRWithZeroPhasesShouldFail)
	sc.Step(`^creating a COSR with a phase with zero objects should fail$`, tc.creatingCOSRWithZeroObjectsShouldFail)
	sc.Step(`^updating the COSR (group|revision|phases|collisionProtection) should fail$`, tc.updatingCOSRFieldShouldFail)
	sc.Step(`^creating a COSR with revision 0 should fail$`, tc.creatingCOSRWithRevisionZeroShouldFail)
	sc.Step(`^creating a COSR with unset lifecycleState should fail$`, tc.creatingCOSRWithUnsetLifecycleStateShouldFail)
	sc.Step(`^creating a COSR with unknown lifecycleState should fail$`, tc.creatingCOSRWithUnknownLifecycleStateShouldFail)
	sc.Step(`^creating a COSR with a group name of exactly 52 characters should succeed$`, tc.creatingCOSRWithExact52CharGroupShouldSucceed)
	sc.Step(`^creating a COSR with a group name longer than 52 characters should fail$`, tc.creatingCOSRWithLongGroupShouldFail)
	sc.Step(`^the COSR is deleted with cascade (foreground|background|orphan)$`, tc.theCOSRIsDeletedWithCascade)
	sc.Step(`^the CRD "([^"]*)" is deleted$`, tc.theCRDIsDeleted)
	sc.Step(`^the gate on ConfigMap "([^"]*)" is (opened|closed)$`, tc.theConfigMapGateIsSetTo)
	sc.Step(`^a resource is patched with:$`, tc.aResourceIsPatched)

	sc.Step(`^the COSR with group "([^"]*)" and revision (\d+) lifecycleState is set to "([^"]*)"$`, tc.theCOSRInGroupLifecycleStateIsSetTo)
	sc.Step(`^the COSR with group "([^"]*)" and revision (\d+) is deleted$`, tc.theCOSRInGroupIsDeleted)

	// COS action steps
	sc.Step(`^the COS is created$`, tc.theCOSIsCreated)
	sc.Step(`^the COS template spec is updated with a ConfigMap "([^"]*)" in phase "([^"]*)"$`, tc.theCOSTemplateSpecIsUpdated)
	sc.Step(`^the COS template spec is updated with a gated ConfigMap "([^"]*)" in phase "([^"]*)"$`, tc.theCOSTemplateSpecIsUpdatedWithGatedConfigMap)
	sc.Step(`^the COS template label "([^"]*)" is updated to "([^"]*)"$`, tc.theCOSTemplateLabelIsUpdated)
	sc.Step(`^the COS "([^"]*)" is deleted$`, tc.theCOSIsDeleted)
	sc.Step(`^the COS "([^"]*)" is deleted with cascade (orphan)$`, tc.theCOSIsDeletedWithCascade)
	sc.Step(`^the COS "([^"]*)" label "([^"]*)" is set to "([^"]*)"$`, tc.theCOSLabelIsSetTo)
	sc.Step(`^the COS "([^"]*)" revisionHistoryLimit is set to (\d+)$`, tc.theCOSRevisionHistoryLimitIsSetTo)
	sc.Step(`^creating a COS with a name of exactly 52 characters should succeed$`, tc.creatingCOSWithExact52CharNameShouldSucceed)
	sc.Step(`^creating a COS with a name longer than 52 characters should fail$`, tc.creatingCOSWithLongNameShouldFail)
}

func (tc *testContext) theCOSRIsCreated(andBecomesAvailable string) error {
	if err := tc.createCOSR(context.Background()); err != nil {
		return err
	}
	if andBecomesAvailable != "" {
		return tc.pollForCOSRCondition(context.Background(), tc.lastCreatedCOSRName(), "Available", metav1.ConditionTrue)
	}
	return nil
}

func (tc *testContext) theConfigMapIsDeleted(name string) error {
	return deleteObject[corev1.ConfigMap](tc, types.NamespacedName{Namespace: tc.namespace, Name: name})
}

func (tc *testContext) theCOSRIsDeletedWithCascade(cascade string) error {
	policy := cascadePolicy(cascade)
	return deleteObject[orbv1alpha1.ClusterObjectSetRevision](tc, types.NamespacedName{Name: tc.lastCreatedCOSRName()}, &client.DeleteOptions{
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
	return deleteObject[apiextensionsv1.CustomResourceDefinition](tc, types.NamespacedName{Name: name + ".e2e.orb.dev"})
}

func (tc *testContext) theCOSRLifecycleStateIsSetTo(state string) error {
	name := tc.lastCreatedCOSRName()
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	if err := tc.client.Get(context.Background(), types.NamespacedName{Name: name}, cosr); err != nil {
		return err
	}
	cosr.Spec.LifecycleState = orbv1alpha1.LifecycleState(state)
	return tc.client.Update(context.Background(), cosr)
}

func (tc *testContext) settingCOSRLifecycleStateShouldFail(state string) error {
	return expectError(tc.theCOSRLifecycleStateIsSetTo(state), fmt.Sprintf("setting lifecycleState to %q", state))
}

func (tc *testContext) creatingTheCOSRShouldFail() error {
	return expectError(tc.createCOSR(context.Background()), "COSR creation")
}

func (tc *testContext) creatingCOSRWithZeroPhasesShouldFail() error {
	tc.resetCOSRBuilder("zero-phases", 1)
	return expectError(tc.createCOSR(context.Background()), "COSR with zero phases")
}

func (tc *testContext) creatingCOSRWithZeroObjectsShouldFail() error {
	tc.resetCOSRBuilder("zero-objects", 1)
	tc.addPhase("empty")
	return expectError(tc.createCOSR(context.Background()), "COSR with zero objects in phase")
}

func (tc *testContext) updatingCOSRFieldShouldFail(field string) error {
	name := tc.lastCreatedCOSRName()
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	if err := tc.client.Get(context.Background(), types.NamespacedName{Name: name}, cosr); err != nil {
		return err
	}
	switch field {
	case "group":
		cosr.Spec.Group = cosr.Spec.Group + "-changed"
	case "revision":
		cosr.Spec.Revision = cosr.Spec.Revision + 1
	case "phases":
		cosr.Spec.Phases = append(cosr.Spec.Phases, orbv1alpha1.Phase{Name: "injected"})
	case "collisionProtection":
		cp := orbv1alpha1.CollisionProtectionNone
		cosr.Spec.CollisionProtection = &cp
	}
	return expectError(tc.client.Update(context.Background(), cosr), fmt.Sprintf("updating %s", field))
}

func (tc *testContext) creatingCOSRWithRevisionZeroShouldFail() error {
	tc.resetCOSRBuilder("rev-zero", 0)
	tc.addPhase("install")
	tc.addObjectToPhase(newConfigMap("cm-rev-zero", tc.namespace))
	return expectError(tc.createCOSR(context.Background()), "COSR with revision 0")
}

func (tc *testContext) creatingCOSRWithUnsetLifecycleStateShouldFail() error {
	tc.resetCOSRBuilder("lcs-unset", 1)
	tc.addPhase("install")
	tc.addObjectToPhase(newConfigMap("cm-lcs-unset", tc.namespace))
	cosr := tc.buildCOSR()
	cosr.Spec.LifecycleState = ""
	return expectError(tc.client.Create(context.Background(), cosr), "COSR with unset lifecycleState")
}

func (tc *testContext) creatingCOSRWithUnknownLifecycleStateShouldFail() error {
	tc.resetCOSRBuilder("lcs-unknown", 1)
	tc.addPhase("install")
	tc.addObjectToPhase(newConfigMap("cm-lcs-unknown", tc.namespace))
	cosr := tc.buildCOSR()
	cosr.Spec.LifecycleState = "Unknown"
	return expectError(tc.client.Create(context.Background(), cosr), "COSR with unknown lifecycleState")
}

func (tc *testContext) cosrGroupOfLength(n int) string {
	prefix := tc.namespace + "-"
	pad := n - len(prefix)
	if pad < 0 {
		pad = 0
	}
	return prefix + strings.Repeat("a", pad)
}

func (tc *testContext) creatingCOSRWithExact52CharGroupShouldSucceed() error {
	tc.resetCOSRBuilder("x", 1)
	tc.cosr.group = tc.cosrGroupOfLength(52)
	tc.addPhase("install")
	tc.addObjectToPhase(newConfigMap("cm-exact-group", tc.namespace))
	return tc.createCOSR(context.Background())
}

func (tc *testContext) creatingCOSRWithLongGroupShouldFail() error {
	tc.resetCOSRBuilder("x", 1)
	tc.cosr.group = tc.cosrGroupOfLength(53)
	tc.addPhase("install")
	tc.addObjectToPhase(newConfigMap("cm-long-group", tc.namespace))
	return expectError(tc.createCOSR(context.Background()), "COSR with group longer than 52 characters")
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

func (tc *testContext) theCOSRInGroupLifecycleStateIsSetTo(group string, revision uint32, state string) error {
	return pollMutateUpdate[orbv1alpha1.ClusterObjectSetRevision](tc, types.NamespacedName{Name: tc.cosrName(group, revision)}, func(cosr *orbv1alpha1.ClusterObjectSetRevision) {
		cosr.Spec.LifecycleState = orbv1alpha1.LifecycleState(state)
	})
}

func (tc *testContext) theCOSRInGroupIsDeleted(group string, revision uint32) error {
	return deleteObject[orbv1alpha1.ClusterObjectSetRevision](tc, types.NamespacedName{Name: tc.cosrName(group, revision)})
}

func (tc *testContext) revisionIsArchived(revision uint32) error {
	for name, cosr := range tc.cosrs {
		if cosr.Spec.Revision == revision {
			latest := &orbv1alpha1.ClusterObjectSetRevision{}
			if err := tc.client.Get(context.Background(), types.NamespacedName{Name: name}, latest); err != nil {
				return err
			}
			latest.Spec.LifecycleState = orbv1alpha1.LifecycleStateArchived
			return tc.client.Update(context.Background(), latest)
		}
	}
	return fmt.Errorf("revision %d not found", revision)
}

func (tc *testContext) theCOSIsCreated() error {
	return tc.createCOS(context.Background())
}

func (tc *testContext) theCOSTemplateSpecIsUpdated(cmName, phaseName string) error {
	return pollMutateUpdate[orbv1alpha1.ClusterObjectSet](tc, types.NamespacedName{Name: tc.lastCreatedCOSName()}, func(cos *orbv1alpha1.ClusterObjectSet) {
		cos.Spec.Template.Spec = orbv1alpha1.ClusterObjectSetTemplateSpec{
			Phases: []orbv1alpha1.Phase{{
				Name:    phaseName,
				Objects: []orbv1alpha1.PhaseObject{newGatedConfigMapPhaseObject(cmName, tc.namespace, false)},
			}},
		}
	})
}

func (tc *testContext) theCOSTemplateSpecIsUpdatedWithGatedConfigMap(cmName, phaseName string) error {
	return pollMutateUpdate[orbv1alpha1.ClusterObjectSet](tc, types.NamespacedName{Name: tc.lastCreatedCOSName()}, func(cos *orbv1alpha1.ClusterObjectSet) {
		cos.Spec.Template.Spec = orbv1alpha1.ClusterObjectSetTemplateSpec{
			Phases: []orbv1alpha1.Phase{{
				Name:    phaseName,
				Objects: []orbv1alpha1.PhaseObject{newGatedConfigMapPhaseObject(cmName, tc.namespace, true)},
			}},
		}
	})
}

func (tc *testContext) theCOSTemplateLabelIsUpdated(key, value string) error {
	return pollMutateUpdate[orbv1alpha1.ClusterObjectSet](tc, types.NamespacedName{Name: tc.lastCreatedCOSName()}, func(cos *orbv1alpha1.ClusterObjectSet) {
		if cos.Spec.Template.Metadata.Labels == nil {
			cos.Spec.Template.Metadata.Labels = make(map[string]string)
		}
		cos.Spec.Template.Metadata.Labels[key] = value
	})
}

func (tc *testContext) theCOSLabelIsSetTo(cosName, key, value string) error {
	return pollMutateUpdate[orbv1alpha1.ClusterObjectSet](tc, types.NamespacedName{Name: tc.cosFullName(cosName)}, func(cos *orbv1alpha1.ClusterObjectSet) {
		if cos.Labels == nil {
			cos.Labels = make(map[string]string)
		}
		cos.Labels[key] = value
	})
}

func (tc *testContext) theCOSRevisionHistoryLimitIsSetTo(cosName string, limit int32) error {
	return pollMutateUpdate[orbv1alpha1.ClusterObjectSet](tc, types.NamespacedName{Name: tc.cosFullName(cosName)}, func(cos *orbv1alpha1.ClusterObjectSet) {
		cos.Spec.RevisionHistoryLimit = &limit
	})
}

func (tc *testContext) cosNameOfLength(n int) string {
	prefix := tc.namespace + "-"
	pad := n - len(prefix)
	if pad < 0 {
		pad = 0
	}
	return prefix + strings.Repeat("b", pad)
}

func (tc *testContext) creatingCOSWithExact52CharNameShouldSucceed() error {
	tc.resetCOSBuilder("x")
	tc.cos.name = tc.cosNameOfLength(52)
	tc.addPhase("install")
	tc.addObjectToPhase(newConfigMap("cm-exact-name", tc.namespace))
	return tc.createCOS(context.Background())
}

func (tc *testContext) creatingCOSWithLongNameShouldFail() error {
	tc.resetCOSBuilder("x")
	tc.cos.name = tc.cosNameOfLength(53)
	tc.addPhase("install")
	tc.addObjectToPhase(newConfigMap("cm-long-name", tc.namespace))
	return expectError(tc.createCOS(context.Background()), "COS with name longer than 52 characters")
}

func (tc *testContext) theCOSIsDeleted(name string) error {
	return deleteObject[orbv1alpha1.ClusterObjectSet](tc, types.NamespacedName{Name: tc.cosFullName(name)})
}

func (tc *testContext) theCOSIsDeletedWithCascade(name, cascade string) error {
	policy := cascadePolicy(cascade)
	return deleteObject[orbv1alpha1.ClusterObjectSet](tc, types.NamespacedName{Name: tc.cosFullName(name)}, &client.DeleteOptions{
		PropagationPolicy: &policy,
	})
}
