package api

import (
	"context"
	"time"

	"github.com/mephistofox/fxtunnel/internal/server/database"
	"github.com/mephistofox/fxtunnel/internal/server/payment"
)

const (
	// reconcileInterval is how often the reconciler sweeps pending payments.
	reconcileInterval = 2 * time.Minute
	// reconcileMinAge leaves payments younger than this to the webhook, so the
	// webhook and the reconciler rarely act on the same payment at once.
	reconcileMinAge = 3 * time.Minute
	// reconcileMaxAge bounds the lookback. The effective useful range is
	// 3min–1h, since the hourly stale cleanup marks older pending payments
	// failed (and this query only returns pending); the extra margin just
	// tolerates a delayed cleanup without unbounding per-sweep lookups.
	reconcileMaxAge = 3 * time.Hour
)

// StartPaymentReconciler runs a background loop that reconciles pending YooKassa
// payments against the provider API and activates any that succeeded but whose
// webhook never arrived (YooKassa webhook delivery is best-effort and has been
// observed to drop notifications). This makes subscription activation
// independent of webhook delivery. It is a no-op when YooKassa is not
// configured, and blocks until ctx is done.
func (s *Server) StartPaymentReconciler(ctx context.Context) {
	if s.paymentProviders == nil || !s.paymentProviders.Has("yookassa") {
		return
	}
	s.log.Info().Dur("interval", reconcileInterval).Msg("Payment reconciler started")

	ticker := time.NewTicker(reconcileInterval)
	defer ticker.Stop()

	s.reconcilePendingPayments(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.reconcilePendingPayments(ctx)
		}
	}
}

// reconcilePendingPayments queries YooKassa for each stale-but-pending payment
// and applies the real outcome: succeeded → activate (idempotent, shared with
// the webhook), canceled → mark failed.
func (s *Server) reconcilePendingPayments(ctx context.Context) {
	prov, err := s.paymentProviders.Get("yookassa")
	if err != nil {
		s.log.Warn().Err(err).Msg("Reconciler: yookassa provider unavailable")
		return
	}
	yk, ok := prov.(*payment.YooKassa)
	if !ok {
		s.log.Warn().Msg("Reconciler: yookassa provider has unexpected type")
		return
	}

	now := time.Now()
	pending, err := s.db.Payments.ListPendingByProviderInWindow("yookassa", now.Add(-reconcileMaxAge), now.Add(-reconcileMinAge))
	if err != nil {
		s.log.Error().Err(err).Msg("Reconciler: failed to list pending payments")
		return
	}

	for _, pmt := range pending {
		select {
		case <-ctx.Done():
			return
		default:
		}

		pid := providerPaymentID(pmt)
		if pid == "" {
			continue
		}

		yooPayment, err := yk.GetPayment(pid)
		if err != nil {
			s.log.Warn().Err(err).
				Int64("invoice_id", pmt.InvoiceID).
				Str("payment_id", pid).
				Msg("Reconciler: GetPayment failed")
			continue
		}

		switch yooPayment.Status {
		case "succeeded":
			_, _, metaPlanID := parseYooKassaMetadata(yooPayment.Metadata)
			res, err := s.applySucceededPayment(pmt, yooPayment, metaPlanID, "reconciler")
			if err != nil {
				s.log.Error().Err(err).
					Int64("invoice_id", pmt.InvoiceID).
					Int64("user_id", pmt.UserID).
					Msg("Reconciler: failed to apply succeeded payment")
				continue
			}
			if res == paymentApplied {
				s.log.Warn().
					Int64("invoice_id", pmt.InvoiceID).
					Int64("user_id", pmt.UserID).
					Str("payment_id", pid).
					Msg("Reconciler: activated a paid payment whose webhook was missed")
			}
		case "canceled":
			pmt.Status = database.PaymentStatusFailed
			if err := s.db.Payments.Update(pmt); err != nil {
				s.log.Error().Err(err).
					Int64("invoice_id", pmt.InvoiceID).
					Msg("Reconciler: failed to mark canceled payment failed")
			}
		}
	}
}
