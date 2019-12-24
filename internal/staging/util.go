package staging

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/agoda-com/samsahai/internal"
	s2hv1beta1 "github.com/agoda-com/samsahai/pkg/apis/env/v1beta1"
)

func (c *controller) getDeployConfiguration(queue *s2hv1beta1.Queue) *internal.ConfigDeploy {
	cfg := c.getConfiguration()
	var deployConfig *internal.ConfigDeploy

	if cfg.Staging != nil {
		deployConfig = cfg.Staging.Deployment
	}

	if queue.IsActivePromotionQueue() {
		if cfg.ActivePromotion != nil {
			deployConfig = cfg.ActivePromotion.Deployment
		} else {
			deployConfig = &internal.ConfigDeploy{}
		}
	}

	return deployConfig
}

func (c *controller) getTestConfiguration(queue *s2hv1beta1.Queue) *internal.ConfigTestRunner {
	deployConfig := c.getDeployConfiguration(queue)
	if deployConfig == nil {
		return nil
	}

	return deployConfig.TestRunner
}

func (c *controller) clearCurrentQueue() {
	c.mtQueue.Lock()
	defer c.mtQueue.Unlock()
	c.currentQueue = nil
}

func (c *controller) getCurrentQueue() *s2hv1beta1.Queue {
	c.mtQueue.Lock()
	defer c.mtQueue.Unlock()
	return c.currentQueue
}

func (c *controller) updateQueue(queue *s2hv1beta1.Queue) error {
	if err := c.client.Update(context.TODO(), queue); err != nil {
		return errors.Wrap(err, "updating queue error")
	}
	return nil
}

func (c *controller) deleteQueue(q *s2hv1beta1.Queue) error {
	isDeploySuccess, isTestSuccess, isReverify := q.IsDeploySuccess(), q.IsTestSuccess(), q.IsReverify()

	if isDeploySuccess && isTestSuccess && !isReverify {
		// success deploy and test without reverify state
		// delete queue
		if err := c.client.Delete(context.TODO(), q); err != nil {
			logger.Error(err, "deleting queue error")
			return err
		}
	} else if isReverify {
		// reverify
		// TODO: fix me, 24 hours hard-code
		if err := c.queueCtrl.SetRetryQueue(q, 0, time.Now().Add(24*time.Hour)); err != nil {
			logger.Error(err, "cannot set retry queue")
			return err
		}
	} else {
		// Testing or deploying failed
		// Retry this component
		q.Spec.NoOfRetry++

		maxNoOfRetry := 0
		configMgr := c.getConfigManager()
		if cfg := configMgr.Get(); cfg != nil && cfg.Staging != nil {
			maxNoOfRetry = cfg.Staging.MaxRetry
		}

		if q.Spec.NoOfRetry > maxNoOfRetry {
			// Retry reached maximum retry limit, we need to verify that is our system still ok?
			if err := c.queueCtrl.SetReverifyQueueAtFirst(q); err != nil {
				logger.Error(err, "cannot set reverify queue")
				return err
			}
		} else {
			if err := c.queueCtrl.SetRetryQueue(q, q.Spec.NoOfRetry, time.Now()); err != nil {
				logger.Error(err, "cannot set retry queue")
				return err
			}
		}
	}

	c.clearCurrentQueue()

	return nil
}

func (c *controller) updateQueueWithState(q *s2hv1beta1.Queue, state s2hv1beta1.QueueState) error {
	q.SetState(state)
	logger.Debug(fmt.Sprintf("queue %s/%s update to state: %s", q.GetNamespace(), q.GetName(), q.Status.State))
	return c.updateQueue(q)
}

func (c *controller) genReleaseName(comp *internal.Component) string {
	return genReleaseName(c.teamName, c.namespace, comp.Name)
}

func genReleaseName(teamName, namespace, compName string) string {
	return teamName + "-" + namespace + "-" + compName
}