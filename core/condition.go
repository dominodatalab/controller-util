package core

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConditionObject interface {
	GetConditions() *[]metav1.Condition
}

type ConditionHelper struct {
	obj     client.Object
	pending map[string]metav1.Condition
}

func NewConditionHelper(obj client.Object) *ConditionHelper {
	return &ConditionHelper{
		obj:     obj,
		pending: map[string]metav1.Condition{},
	}
}

func (h *ConditionHelper) Flush() error {
	// NOTE: what do we do if obj does not adhere to interface, assuming they have not conditions?
	condObj, ok := h.obj.(ConditionObject)
	if !ok {
		return nil
	}

	for _, cond := range h.pending {
		SetStatusCondition(condObj.GetConditions(), cond)
	}

	h.pending = map[string]metav1.Condition{}
	return nil
}

func (h *ConditionHelper) SetCondition(cond metav1.Condition) {
	if cond.ObservedGeneration == 0 {
		cond.ObservedGeneration = h.obj.GetGeneration()
	}
	h.pending[cond.Type] = cond
}

func (h *ConditionHelper) Set(
	conditionType string, status metav1.ConditionStatus, reason, message string,
) {
	h.SetCondition(metav1.Condition{
		Type:    conditionType,
		Status:  status,
		Reason:  reason,
		Message: message,
	})
}

func (h *ConditionHelper) Setf(
	conditionType string, status metav1.ConditionStatus, reason, message string, args ...interface{},
) {
	h.Set(conditionType, status, reason, fmt.Sprintf(message, args...))
}

func (h *ConditionHelper) SetFalse(conditionType string, reason, message string) {
	h.Set(conditionType, metav1.ConditionFalse, reason, message)
}

func (h *ConditionHelper) SetTrue(conditionType string, reason, message string) {
	h.Set(conditionType, metav1.ConditionTrue, reason, message)
}

func (h *ConditionHelper) SetUnknown(conditionType string, reason, message string) {
	h.Set(conditionType, metav1.ConditionUnknown, reason, message)
}

func (h *ConditionHelper) SetfUnknown(conditionType string, reason, message string, args ...interface{}) {
	h.Setf(conditionType, metav1.ConditionUnknown, reason, message, args...)
}

func SetStatusCondition(conditions *[]metav1.Condition, newCondition metav1.Condition) {
	// Find the condition index if it exists
	index, exists := FindStatusCondition(*conditions, newCondition.Type)

	if !exists {
		// Add new condition
		if newCondition.LastTransitionTime.IsZero() {
			newCondition.LastTransitionTime = metav1.NewTime(time.Now())
		}
		*conditions = append(*conditions, newCondition)
		return
	}

	// Update existing condition
	existing := (*conditions)[index]

	// Create updated condition with all fields properly set
	updatedCondition := metav1.Condition{
		Type:               existing.Type,
		Status:             newCondition.Status,
		Reason:             newCondition.Reason,
		Message:            newCondition.Message,
		LastTransitionTime: existing.LastTransitionTime,
		ObservedGeneration: newCondition.ObservedGeneration,
	}

	// Update LastTransitionTime if status changes
	if existing.Status != newCondition.Status {
		if !newCondition.LastTransitionTime.IsZero() {
			updatedCondition.LastTransitionTime = newCondition.LastTransitionTime
		} else {
			updatedCondition.LastTransitionTime = metav1.NewTime(time.Now())
		}
	}

	// Preserve ObservedGeneration if not set in new condition
	if newCondition.ObservedGeneration == 0 {
		updatedCondition.ObservedGeneration = existing.ObservedGeneration
	}

	// Replace the condition in the slice
	(*conditions)[index] = updatedCondition
}

func FindStatusCondition(conditions []metav1.Condition, conditionType string) (int, bool) {
	for i, cond := range conditions {
		if cond.Type == conditionType {
			return i, true
		}
	}
	return -1, false
}
