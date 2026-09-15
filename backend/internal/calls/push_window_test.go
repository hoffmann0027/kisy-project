package calls

import "testing"

// A call is written off as missed when RingTimeout fires. If Firebase were
// still allowed to deliver the invite at that point, the callee's phone could
// start ringing for a call the server had already buried — and answering it
// would fail. The gap is the callee's time to react, so it has to be real.
func TestRingTimeoutOutlivesPush(t *testing.T) {
	if RingTimeout <= CallPushTTL {
		t.Fatalf("RingTimeout (%s) must exceed CallPushTTL (%s): the invite would expire as missed while still in flight",
			RingTimeout, CallPushTTL)
	}
	if gap := RingTimeout - CallPushTTL; gap < 10*1e9 {
		t.Fatalf("only %s between the last possible delivery and the missed mark; that is not enough to answer", gap)
	}
}
