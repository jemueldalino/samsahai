package activepromotion

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/agoda-com/samsahai/internal"
	"github.com/agoda-com/samsahai/internal/samsahai/exporter"
	s2hv1beta1 "github.com/agoda-com/samsahai/pkg/apis/env/v1beta1"
)

func (c *controller) createActivePromotionHistory(ctx context.Context, atpComp *s2hv1beta1.ActivePromotion) (string, error) {
	defaultLabels := internal.GetDefaultLabels(atpComp.Name)
	if err := c.deleteActivePromotionHistoryIfOutOfRange(ctx, atpComp.Name, defaultLabels); err != nil {
		return "", err
	}

	now := metav1.Now()
	atpLabels := internal.GetDefaultLabels(atpComp.Name)
	atpLabels["namespace"] = atpComp.Status.TargetNamespace

	history := &s2hv1beta1.ActivePromotionHistory{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s",
				atpComp.Name,
				now.Format("20060102-150405")),
			Labels: atpLabels,
		},
		Spec: s2hv1beta1.ActivePromotionHistorySpec{
			TeamName: atpComp.Name,
			ActivePromotion: &s2hv1beta1.ActivePromotion{
				Spec:   atpComp.Spec,
				Status: atpComp.Status,
			},
			IsSuccess: atpComp.IsActivePromotionSuccess(),
			CreatedAt: &now,
		},
	}

	// TODO: store values file

	if err := c.client.Create(ctx, history); err != nil && !k8serrors.IsNotFound(err) {
		return "", errors.Wrapf(err, "cannot create activepromotionhistory of %s", atpComp.Name)
	}

	return history.Name, nil
}

func (c *controller) updateActivePromotionHistory(ctx context.Context, histName string, atpComp *s2hv1beta1.ActivePromotion) error {
	atpHist := &s2hv1beta1.ActivePromotionHistory{}
	if err := c.client.Get(ctx, types.NamespacedName{Name: histName}, atpHist); err != nil {
		return err
	}

	atpHist.Spec.ActivePromotion = &s2hv1beta1.ActivePromotion{
		Spec:   atpComp.Spec,
		Status: atpComp.Status,
	}

	if err := c.client.Update(ctx, atpHist); err != nil {
		return errors.Wrapf(err, "cannot update activepromotionhistory %s", histName)
	}

	atpHisList := s2hv1beta1.ActivePromotionHistoryList{Items: []s2hv1beta1.ActivePromotionHistory{*atpHist}}
	exporter.SetActivePromotionHistoriesMetric(&atpHisList)

	return nil
}

func (c *controller) deleteActivePromotionHistoryIfOutOfRange(ctx context.Context, teamName string, selectors map[string]string) error {
	atpHists := s2hv1beta1.ActivePromotionHistoryList{}
	listOpt := client.ListOptions{LabelSelector: labels.SelectorFromSet(selectors)}
	if err := c.client.List(ctx, &listOpt, &atpHists); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}

		return errors.Wrapf(err, "cannot list activepromotionhistories with selectors %+v", selectors)
	}

	maxHistoriesPerTeam := c.configs.ActivePromotion.MaxHistories

	// get configuration
	configMgr, err := c.getTeamConfiguration(teamName)
	if err != nil {
		return err
	}
	if cfg := configMgr.Get(); cfg.ActivePromotion != nil && cfg.ActivePromotion.MaxHistories != 0 {
		maxHistoriesPerTeam = cfg.ActivePromotion.MaxHistories
	}

	// +1 for current active promotion history
	if len(atpHists.Items)+1 > maxHistoriesPerTeam {
		atpHists.SortDESC()

		// -2 for current active promotion history
		for i := len(atpHists.Items) - 1; i > maxHistoriesPerTeam-2; i-- {
			if err := c.client.Delete(ctx, &atpHists.Items[i]); err != nil {
				if k8serrors.IsNotFound(err) {
					return nil
				}

				return errors.Wrapf(err, "cannot delete activepromotionhistory %s", atpHists.Items[i].Name)
			}
		}
	}
	return nil
}