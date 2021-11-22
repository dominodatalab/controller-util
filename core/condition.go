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

type conditionHelper struct {
	obj     client.Object
	pending map[string]metav1.Condition
}

func NewConditionHelper(obj client.Object) *conditionHelper {
	return &conditionHelper{
		obj:     obj,
		pending: map[string]metav1.Condition{},
	}
}

func (h *conditionHelper) Flush() error {
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

func (h *conditionHelper) SetCondition(cond metav1.Condition) {
	if cond.ObservedGeneration == 0 {
		cond.ObservedGeneration = h.obj.GetGeneration()
	}
	h.pending[cond.Type] = cond
}

func (h *conditionHelper) Set(conditionType string, status metav1.ConditionStatus, reason, message string) {
	h.SetCondition(metav1.Condition{
		Type:    conditionType,
		Status:  status,
		Reason:  reason,
		Message: message,
	})
}

func (h *conditionHelper) Setf(conditionType string, status metav1.ConditionStatus, reason, message string, args ...interface{}) {
	h.Set(conditionType, status, reason, fmt.Sprintf(message, args...))
}

func (h *conditionHelper) SetFalse(conditionType string, reason, message string) {
	h.Set(conditionType, metav1.ConditionFalse, reason, message)
}

func (h *conditionHelper) SetTrue(conditionType string, reason, message string) {
	h.Set(conditionType, metav1.ConditionTrue, reason, message)
}

func (h *conditionHelper) SetUnknown(conditionType string, reason, message string) {
	h.Set(conditionType, metav1.ConditionUnknown, reason, message)
}

func (h *conditionHelper) SetfUnknown(conditionType string, reason, message string, args ...interface{}) {
	h.Setf(conditionType, metav1.ConditionUnknown, reason, message, args...)
}

func SetStatusCondition(conditions *[]metav1.Condition, newCondition metav1.Condition) {
	existing := FindStatusCondition(*conditions, newCondition.Type)

	if existing == nil {
		if newCondition.LastTransitionTime.IsZero() {
			newCondition.LastTransitionTime = metav1.NewTime(time.Now())
		}

		*conditions = append(*conditions, newCondition)
		return
	}

	if existing.Status != newCondition.Status {
		existing.Status = newCondition.Status

		if !newCondition.LastTransitionTime.IsZero() {
			existing.LastTransitionTime = newCondition.LastTransitionTime
		} else {
			existing.LastTransitionTime = metav1.NewTime(time.Now())
		}
	}

	existing.Reason = newCondition.Reason
	existing.Message = newCondition.Message
}

func FindStatusCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for _, cond := range conditions {
		if cond.Type == conditionType {
			return &cond
		}
	}

	return nil
}
