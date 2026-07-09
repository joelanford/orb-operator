package e2e

import (
	"context"
	"fmt"

	"github.com/cucumber/godog"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func registerSetupSteps(sc *godog.ScenarioContext, tc *testContext) {
	sc.Step(`^a COS named "([^"]*)" with group "([^"]*)" and revision (\d+)$`, tc.aCOSNamedWithGroupAndRevision)
	sc.Step(`^a COS with group "([^"]*)" and revision (\d+)$`, tc.aCOSWithGroupAndRevision)
	sc.Step(`^(?:a|the) phase "([^"]*)" (?:with|has) a ConfigMap "([^"]*)"$`, tc.aPhaseWithConfigMap)
	sc.Step(`^(?:a|the) phase "([^"]*)" (?:with|has) a gated ConfigMap "([^"]*)"$`, tc.aPhaseWithGatedConfigMap)
	sc.Step(`^the phase "([^"]*)" also has a ConfigMap "([^"]*)"$`, tc.phaseAlsoHasConfigMap)
	sc.Step(`^a phase "([^"]*)" with a CRD "([^"]*)"$`, tc.aPhaseWithCRD)
	sc.Step(`^the phase "([^"]*)" also has a CRD "([^"]*)"$`, tc.phaseAlsoHasCRD)
	sc.Step(`^a phase "([^"]*)" with a CRD "([^"]*)" with assertion conditionEqual type "([^"]*)" status "([^"]*)"$`, tc.aPhaseWithCRDConditionEqual)
	sc.Step(`^a phase "([^"]*)" with a "([^"]*)" named "([^"]*)"$`, tc.aPhaseWithCR)
	sc.Step(`^the phase "([^"]*)" also has a "([^"]*)" named "([^"]*)"$`, tc.phaseAlsoHasCR)
	sc.Step(`^a phase "([^"]*)" with a ConfigMap "([^"]*)" with assertion fieldValue path "([^"]*)" value "([^"]*)"$`, tc.aPhaseWithConfigMapFieldValue)
	sc.Step(`^a phase "([^"]*)" with a ConfigMap "([^"]*)" with assertion celExpression "([^"]*)"$`, tc.aPhaseWithConfigMapCEL)
	sc.Step(`^a phase "([^"]*)" with a ConfigMap "([^"]*)" with assertion celExpression "([^"]*)" message "([^"]*)"$`, tc.aPhaseWithConfigMapCELMessage)
	sc.Step(`^(?:a|the) phase "([^"]*)" (?:with|has) a ConfigMap "([^"]*)" with data:$`, tc.aPhaseWithConfigMapDataTable)
	sc.Step(`^a phase "([^"]*)" with a ConfigMap "([^"]*)" with data key "([^"]*)" value "([^"]*)"$`, tc.aPhaseWithConfigMapData)
	sc.Step(`^the phase "([^"]*)" also has a ConfigMap "([^"]*)" with data:$`, tc.phaseAlsoHasConfigMapDataTable)
	sc.Step(`^the last object has assertion conditionEqual type "([^"]*)" status "([^"]*)"$`, tc.lastObjectHasConditionEqualAssertion)
	sc.Step(`^the last object has assertion fieldsEqual fieldA "([^"]*)" fieldB "([^"]*)"$`, tc.lastObjectHasFieldsEqualAssertion)
	sc.Step(`^the last object has assertion fieldValue path "([^"]*)" value "([^"]*)"$`, tc.lastObjectHasFieldValueAssertion)
	sc.Step(`^the last object has assertion celExpression "([^"]*)"$`, tc.lastObjectHasCELAssertion)
	sc.Step(`^the COS collisionProtection is "([^"]*)"$`, tc.theCOSCollisionProtectionIs)
	sc.Step(`^the phase "([^"]*)" collisionProtection is "([^"]*)"$`, tc.thePhaseCollisionProtectionIs)
	sc.Step(`^the last object collisionProtection is "([^"]*)"$`, tc.theLastObjectCollisionProtectionIs)
	sc.Step(`^a standalone ConfigMap "([^"]*)" exists$`, tc.aStandaloneConfigMapExists)

	sc.Step(`^a phase "([^"]*)" with an unregistered resource type$`, tc.aPhaseWithUnregisteredResourceType)
	sc.Step(`^ConfigMap(?:\s+"([^"]*)")? operations are blocked$`, tc.configMapOpsAreBlocked)

	sc.Step(`^an available COS with group "([^"]*)" and revision (\d+)$`, tc.anAvailableCOS)

	// COD setup steps
	sc.Step(`^a COD named "([^"]*)"$`, tc.aCODNamed)
	sc.Step(`^a COD named "([^"]*)" with revisionHistoryLimit (\d+)$`, tc.aCODNamedWithRevisionHistoryLimit)
	sc.Step(`^an available COD named "([^"]*)"$`, tc.anAvailableCOD)
	sc.Step(`^the COD template has (label|annotation) "([^"]*)" with value "([^"]*)"$`, tc.theCODTemplateHasMetadata)
	sc.Step(`^the COD has progressDeadlineMinutes (\d+)$`, tc.theCODHasProgressDeadlineMinutes)
}

func (tc *testContext) aCOSNamedWithGroupAndRevision(name, group string, revision uint32) {
	tc.resetCOSBuilder(group, revision)
	tc.cos.nameOverride = name
}

func (tc *testContext) anAvailableCOS(group string, revision uint32) error {
	tc.resetCOSBuilder(group, revision)
	tc.addPhase("install")
	tc.addConfigMapToPhase("cm-"+group, false)
	if err := tc.createCOS(context.Background()); err != nil {
		return err
	}
	return tc.pollForCOSCondition(context.Background(), tc.lastCreatedCOSName(), "Available", metav1.ConditionTrue)
}

func (tc *testContext) aCOSWithGroupAndRevision(group string, revision uint32) {
	tc.resetCOSBuilder(group, revision)
}

func (tc *testContext) addConfigMapToPhase(name string, gated bool) {
	assertion := openByDefaultGateAssertion
	if gated {
		assertion = closedByDefaultGateAssertion
	}
	phase := tc.currentPhase()
	phase.Objects = append(phase.Objects, orbv1alpha1.PhaseObject{
		Object:     runtime.RawExtension{Object: newConfigMap(name, tc.namespace)},
		Assertions: []orbv1alpha1.Assertion{assertion},
	})
}

func (tc *testContext) aPhaseWithConfigMap(phaseName, cmName string) {
	tc.addPhase(phaseName)
	tc.addConfigMapToPhase(cmName, false)
}

func (tc *testContext) aPhaseWithGatedConfigMap(phaseName, cmName string) {
	tc.addPhase(phaseName)
	tc.addConfigMapToPhase(cmName, true)
}

func (tc *testContext) phaseAlsoHasConfigMap(_, cmName string) {
	tc.addConfigMapToPhase(cmName, false)
}

func (tc *testContext) aPhaseWithCRDConditionEqual(phaseName, crdName, condType, condStatus string) {
	tc.addPhase(phaseName)
	crd := newCRD(crdName, tc.namespace)
	tc.crds = append(tc.crds, crd.Name)
	tc.addObjectWithAssertions(crd, []orbv1alpha1.Assertion{{
		ConditionEqual: &orbv1alpha1.ConditionEqualAssertion{
			Type:   condType,
			Status: condStatus,
		},
	}})
}

func (tc *testContext) aPhaseWithCRD(phaseName, crdName string) {
	tc.addPhase(phaseName)
	crd := newCRD(crdName, tc.namespace)
	tc.crds = append(tc.crds, crd.Name)
	tc.addObjectToPhase(crd)
}

func (tc *testContext) phaseAlsoHasCRD(_, crdName string) {
	crd := newCRD(crdName, tc.namespace)
	tc.crds = append(tc.crds, crd.Name)
	tc.addObjectToPhase(crd)
}

func (tc *testContext) aPhaseWithCR(phaseName, crdName, crName string) {
	tc.addPhase(phaseName)
	tc.addObjectToPhase(newCR(crdName, crName, tc.namespace))
}

func (tc *testContext) phaseAlsoHasCR(_, crdName, crName string) {
	tc.addObjectToPhase(newCR(crdName, crName, tc.namespace))
}

func (tc *testContext) aPhaseWithConfigMapFieldValue(phaseName, cmName, path, value string) {
	tc.addPhase(phaseName)
	tc.addObjectWithAssertions(newConfigMap(cmName, tc.namespace), []orbv1alpha1.Assertion{{
		FieldValue: &orbv1alpha1.FieldValueAssertion{
			FieldPath: path,
			Value:     value,
		},
	}})
}

func (tc *testContext) aPhaseWithConfigMapCEL(phaseName, cmName, expr string) {
	tc.addPhase(phaseName)
	tc.addObjectWithAssertions(newConfigMap(cmName, tc.namespace), []orbv1alpha1.Assertion{{
		CELExpression: &orbv1alpha1.CELExpressionAssertion{
			Expression: expr,
		},
	}})
}

func (tc *testContext) aPhaseWithConfigMapCELMessage(phaseName, cmName, expr, message string) {
	tc.addPhase(phaseName)
	tc.addObjectWithAssertions(newConfigMap(cmName, tc.namespace), []orbv1alpha1.Assertion{{
		CELExpression: &orbv1alpha1.CELExpressionAssertion{
			Expression: expr,
			Message:    message,
		},
	}})
}

func (tc *testContext) aPhaseWithConfigMapDataTable(phaseName, cmName string, table *godog.Table) {
	tc.addPhase(phaseName)
	tc.addObjectToPhase(newConfigMapWithData(cmName, tc.namespace, tableToMap(table)))
}

func (tc *testContext) aPhaseWithConfigMapData(phaseName, cmName, key, value string) {
	tc.addPhase(phaseName)
	tc.addObjectToPhase(newConfigMapWithData(cmName, tc.namespace, map[string]string{key: value}))
}

func (tc *testContext) phaseAlsoHasConfigMapDataTable(_, cmName string, table *godog.Table) {
	tc.addObjectToPhase(newConfigMapWithData(cmName, tc.namespace, tableToMap(table)))
}

func (tc *testContext) lastObjectHasConditionEqualAssertion(condType, condStatus string) {
	obj := tc.lastObject()
	obj.Assertions = append(obj.Assertions, orbv1alpha1.Assertion{
		ConditionEqual: &orbv1alpha1.ConditionEqualAssertion{
			Type:   condType,
			Status: condStatus,
		},
	})
}

func (tc *testContext) lastObjectHasFieldsEqualAssertion(fieldA, fieldB string) {
	obj := tc.lastObject()
	obj.Assertions = append(obj.Assertions, orbv1alpha1.Assertion{
		FieldsEqual: &orbv1alpha1.FieldsEqualAssertion{
			FieldA: fieldA,
			FieldB: fieldB,
		},
	})
}

func (tc *testContext) lastObjectHasFieldValueAssertion(path, value string) {
	obj := tc.lastObject()
	obj.Assertions = append(obj.Assertions, orbv1alpha1.Assertion{
		FieldValue: &orbv1alpha1.FieldValueAssertion{
			FieldPath: path,
			Value:     value,
		},
	})
}

func (tc *testContext) lastObjectHasCELAssertion(expr string) {
	obj := tc.lastObject()
	obj.Assertions = append(obj.Assertions, orbv1alpha1.Assertion{
		CELExpression: &orbv1alpha1.CELExpressionAssertion{
			Expression: expr,
		},
	})
}

func (tc *testContext) theCOSCollisionProtectionIs(cp string) {
	v := orbv1alpha1.CollisionProtection(cp)
	tc.tmpl.collisionProtection = &v
}

func (tc *testContext) thePhaseCollisionProtectionIs(phaseName, cp string) {
	v := orbv1alpha1.CollisionProtection(cp)
	for i := range tc.tmpl.phases {
		if tc.tmpl.phases[i].Name == phaseName {
			tc.tmpl.phases[i].CollisionProtection = &v
			return
		}
	}
}

func (tc *testContext) theLastObjectCollisionProtectionIs(cp string) {
	v := orbv1alpha1.CollisionProtection(cp)
	tc.lastObject().CollisionProtection = &v
}

func (tc *testContext) aStandaloneConfigMapExists(name string) error {
	return tc.client.Create(context.Background(), newConfigMap(name, tc.namespace))
}

func (tc *testContext) aCODNamed(name string) {
	tc.resetCODBuilder(name)
}

func (tc *testContext) anAvailableCOD(name string) error {
	tc.resetCODBuilder(name)
	tc.addPhase("install")
	tc.addConfigMapToPhase("cm-"+name, false)
	if err := tc.createCOD(context.Background()); err != nil {
		return err
	}
	return tc.theCODShouldBeAvailable(name)
}

func (tc *testContext) aCODNamedWithRevisionHistoryLimit(name string, limit int32) {
	tc.resetCODBuilder(name)
	tc.cod.revisionHistoryLimit = &limit
}

func (tc *testContext) theCODTemplateHasMetadata(kind, key, value string) {
	if kind == "label" {
		if tc.cod.labels == nil {
			tc.cod.labels = make(map[string]string)
		}
		tc.cod.labels[key] = value
	} else {
		if tc.cod.annotations == nil {
			tc.cod.annotations = make(map[string]string)
		}
		tc.cod.annotations[key] = value
	}
}

func (tc *testContext) theCODHasProgressDeadlineMinutes(minutes int32) {
	tc.cod.progressDeadlineMinutes = &minutes
}

func newConfigMap(name, namespace string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func newConfigMapWithData(name, namespace string, data map[string]string) *corev1.ConfigMap {
	cm := newConfigMap(name, namespace)
	cm.Data = data
	return cm
}

var openByDefaultGateAssertion = orbv1alpha1.Assertion{
	CELExpression: &orbv1alpha1.CELExpressionAssertion{
		Expression: "!has(self.data) || !has(self.data.gate) || self.data.gate != 'closed'",
	},
}

var closedByDefaultGateAssertion = orbv1alpha1.Assertion{
	CELExpression: &orbv1alpha1.CELExpressionAssertion{
		Expression: "has(self.data) && has(self.data.gate) && self.data.gate == 'open'",
	},
}

func newGatedConfigMapPhaseObject(name, namespace string, gated bool) orbv1alpha1.PhaseObject {
	assertion := openByDefaultGateAssertion
	if gated {
		assertion = closedByDefaultGateAssertion
	}
	return orbv1alpha1.PhaseObject{
		Object:     runtime.RawExtension{Object: newConfigMap(name, namespace)},
		Assertions: []orbv1alpha1.Assertion{assertion},
	}
}

func newCRD(name, namespace string) *apiextensionsv1.CustomResourceDefinition {
	group := namespace + ".e2e.orb.dev"
	return &apiextensionsv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name + "." + group,
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: group,
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:   name,
				Singular: name[:len(name)-1],
				Kind:     capitalize(name[:len(name)-1]),
			},
			Scope: apiextensionsv1.ClusterScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{
				Name:    "v1alpha1",
				Served:  true,
				Storage: true,
				Schema: &apiextensionsv1.CustomResourceValidation{
					OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
						Type:                   "object",
						XPreserveUnknownFields: boolPtr(true),
					},
				},
			}},
		},
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return string(s[0]-32) + s[1:]
}

func boolPtr(b bool) *bool {
	return &b
}

func newCR(crdName, crName, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": namespace + ".e2e.orb.dev/v1alpha1",
			"kind":       capitalize(crdName[:len(crdName)-1]),
			"metadata": map[string]interface{}{
				"name": crName,
			},
		},
	}
}

func (tc *testContext) aPhaseWithUnregisteredResourceType(phaseName string) {
	tc.addPhase(phaseName)
	tc.addObjectToPhase(&unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "totally.fake.example.com/v1",
			"kind":       "Nonexistent",
			"metadata": map[string]interface{}{
				"name": "fake-resource",
			},
		},
	})
}

func (tc *testContext) configMapOpsAreBlocked(cmName string) error {
	ctx := context.Background()

	vapName := tc.namespace + "-block-cm"
	if cmName != "" {
		vapName += "-" + cmName
	}

	writeExpr := "request.dryRun"
	deleteExpr := "request.dryRun"
	if cmName != "" {
		writeExpr += " || !object.metadata.name.startsWith('" + cmName + "')"
		deleteExpr += " || !oldObject.metadata.name.startsWith('" + cmName + "')"
	}

	vap := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "admissionregistration.k8s.io/v1",
		"kind":       "ValidatingAdmissionPolicy",
		"metadata":   map[string]interface{}{"name": vapName},
		"spec": map[string]interface{}{
			"failurePolicy": "Fail",
			"matchConstraints": map[string]interface{}{
				"resourceRules": []interface{}{map[string]interface{}{
					"apiGroups":   []interface{}{""},
					"apiVersions": []interface{}{"v1"},
					"operations":  []interface{}{"CREATE", "UPDATE", "DELETE"},
					"resources":   []interface{}{"configmaps"},
				}},
				"namespaceSelector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"kubernetes.io/metadata.name": tc.namespace,
					},
				},
			},
			"validations": []interface{}{
				map[string]interface{}{
					"expression": "request.operation == 'DELETE' || " + writeExpr,
					"message":    "e2e: configmap write blocked",
				},
				map[string]interface{}{
					"expression": "request.operation != 'DELETE' || " + deleteExpr,
					"message":    "e2e: configmap delete blocked",
				},
			},
		},
	}}
	if err := tc.client.Create(ctx, vap); err != nil {
		return fmt.Errorf("creating VAP: %w", err)
	}

	vapb := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "admissionregistration.k8s.io/v1",
		"kind":       "ValidatingAdmissionPolicyBinding",
		"metadata":   map[string]interface{}{"name": vapName},
		"spec": map[string]interface{}{
			"policyName":        vapName,
			"validationActions": []interface{}{"Deny"},
		},
	}}
	if err := tc.client.Create(ctx, vapb); err != nil {
		return fmt.Errorf("creating VAPB: %w", err)
	}

	tc.createdObjects = append(tc.createdObjects,
		metav1.PartialObjectMetadata{
			TypeMeta:   metav1.TypeMeta{APIVersion: "admissionregistration.k8s.io/v1", Kind: "ValidatingAdmissionPolicy"},
			ObjectMeta: metav1.ObjectMeta{Name: vapName},
		},
		metav1.PartialObjectMetadata{
			TypeMeta:   metav1.TypeMeta{APIVersion: "admissionregistration.k8s.io/v1", Kind: "ValidatingAdmissionPolicyBinding"},
			ObjectMeta: metav1.ObjectMeta{Name: vapName},
		},
	)

	canaryName := "canary-vap-probe"
	if cmName != "" {
		canaryName = cmName + "-canary"
	}
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		canary := newConfigMap(canaryName, tc.namespace)
		if err := tc.client.Create(ctx, canary); err != nil {
			return true, nil
		}
		_ = tc.client.Delete(ctx, canary)
		return false, nil
	})
}

func tableToMap(table *godog.Table) map[string]string {
	data := make(map[string]string)
	for _, row := range table.Rows[1:] {
		data[row.Cells[0].Value] = row.Cells[1].Value
	}
	return data
}
