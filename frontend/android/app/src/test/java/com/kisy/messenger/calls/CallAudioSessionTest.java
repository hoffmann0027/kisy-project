package com.kisy.messenger.calls;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertNull;
import static org.junit.Assert.assertTrue;

import java.util.ArrayList;
import java.util.List;
import org.junit.Before;
import org.junit.Test;

/**
 * The rules the call plugin delegates to. The one that matters most: whatever
 * state a call ends in, the proximity lock is released — a leaked lock leaves
 * the phone's screen dark and unresponsive after the call.
 */
public class CallAudioSessionTest {

    /** A phone with a proximity sensor and no hardware behind it. */
    static final class FakePhone implements CallAudioSession.Platform {
        boolean inCallMode;
        boolean wired;
        boolean bluetooth;
        boolean lockHeld;
        int acquires;
        int releases;
        CallAudioSession.Route routed;
        final List<String> published = new ArrayList<>();

        @Override public void enterCallMode() { inCallMode = true; }
        @Override public void leaveCallMode() { inCallMode = false; }
        @Override public boolean hasWiredHeadset() { return wired; }
        @Override public boolean hasBluetoothHeadset() { return bluetooth; }
        @Override public void routeTo(CallAudioSession.Route route) { routed = route; }

        @Override
        public void acquireProximityLock() {
            // A real WakeLock acquired twice with reference counting off is
            // still one lock; counting acquires catches a session that asks
            // again and again for no reason.
            lockHeld = true;
            acquires++;
        }

        @Override
        public void releaseProximityLock() {
            lockHeld = false;
            releases++;
        }

        @Override public boolean isProximityLockHeld() { return lockHeld; }
        int silenced;
        @Override public void silenceRinger() { silenced++; }
        @Override public void publish(CallAudioSession.Route route, boolean speaker) { published.add(route.wire + ":" + speaker); }
    }

    private FakePhone phone;
    private CallAudioSession session;

    @Before
    public void setUp() {
        phone = new FakePhone();
        session = new CallAudioSession(phone);
    }

    @org.junit.After
    public void tearDown() {
        session.end(); // the in-call flag is process-wide; leave it clear
    }

    // --- the ringer during a call ------------------------------------------
    //
    // The server sends the call push even when the app is open, and it can
    // land after the call was answered in the app. A ringtone started then
    // plays through the whole conversation — and in MODE_IN_COMMUNICATION
    // Android turns it into its in-call notification tone: a short double
    // beep, over and over.

    @Test
    public void afterConnectedTheRingerIsStopped() {
        session.start(false);
        assertTrue("the ringer must be silenced when the call has media", phone.silenced >= 1);
    }

    @Test
    public void everyRouteChangeSilencesTheRingerAgain() {
        session.start(false);
        int afterStart = phone.silenced;
        session.setSpeaker(true);
        assertTrue(phone.silenced > afterStart);
        int afterSpeaker = phone.silenced;
        phone.wired = true;
        session.devicesChanged();
        assertTrue("a ringer restarted mid-call is stopped on the next route change", phone.silenced > afterSpeaker);
    }

    @Test
    public void aCallPushDoesNotRingWhileACallIsInProgress() {
        assertFalse(CallAudioSession.callInProgress());
        session.start(false);
        assertTrue("a late call push must not start the ringer during a call", CallAudioSession.callInProgress());
        session.end();
        assertFalse("the next call must ring again", CallAudioSession.callInProgress());
    }

    @Test
    public void voiceCallStartsOnTheEarpieceWithTheScreenBlankingAtTheEar() {
        session.start(false);
        assertTrue(phone.inCallMode);
        assertEquals(CallAudioSession.Route.EARPIECE, phone.routed);
        assertTrue(phone.lockHeld);
        assertEquals("earpiece:false", phone.published.get(phone.published.size() - 1));
    }

    @Test
    public void endingTheCallReleasesTheProximityLock() {
        session.start(false);
        session.end();
        assertFalse("screen left dark after the call", phone.lockHeld);
        assertFalse(phone.inCallMode);
        assertFalse(session.isActive());
        assertNull(session.route());
    }

    @Test
    public void endingReleasesTheLockEvenIfTheSessionThinksItIsOver() {
        // The lock is held but the session is not active — the state a crash
        // between two calls, or a second end() from a racing teardown, leaves.
        phone.lockHeld = true;
        session.end();
        assertFalse(phone.lockHeld);
    }

    @Test
    public void endingTwiceIsHarmless() {
        session.start(false);
        session.end();
        session.end();
        assertFalse(phone.lockHeld);
        assertEquals(1, phone.releases);
    }

    @Test
    public void loudspeakerReleasesTheLockAndEarpieceTakesItBack() {
        session.start(false);
        session.setSpeaker(true);
        assertEquals(CallAudioSession.Route.SPEAKER, phone.routed);
        assertFalse("screen must not blank on the loudspeaker", phone.lockHeld);

        session.setSpeaker(false);
        assertEquals(CallAudioSession.Route.EARPIECE, phone.routed);
        assertTrue(phone.lockHeld);
        assertEquals(2, phone.acquires);
    }

    @Test
    public void endingOnTheLoudspeakerAfterEarpieceStillLeavesNoLock() {
        session.start(false);
        session.setSpeaker(true);
        session.setSpeaker(false);
        session.end();
        assertFalse(phone.lockHeld);
    }

    @Test
    public void videoCallIsOnTheLoudspeakerWithoutTheProximityLock() {
        session.start(true);
        assertEquals(CallAudioSession.Route.SPEAKER, phone.routed);
        assertFalse(phone.lockHeld);

        // Even switched to the earpiece, a video call keeps its screen: the
        // picture is the point.
        session.setSpeaker(false);
        assertEquals(CallAudioSession.Route.EARPIECE, phone.routed);
        assertFalse(phone.lockHeld);
        assertEquals(0, phone.acquires);
    }

    @Test
    public void aHeadsetTakesTheSoundAndTheLockGoes() {
        session.start(false);
        phone.wired = true;
        session.devicesChanged();
        assertEquals(CallAudioSession.Route.WIRED, phone.routed);
        assertFalse(phone.lockHeld);

        phone.wired = false;
        session.devicesChanged();
        assertEquals("unplugged: back to the earpiece, not the loudspeaker", CallAudioSession.Route.EARPIECE, phone.routed);
        assertTrue(phone.lockHeld);
    }

    @Test
    public void headsetWinsOverAnEarlierLoudspeakerChoice() {
        session.start(false);
        session.setSpeaker(true);
        phone.bluetooth = true;
        session.devicesChanged();
        assertEquals(CallAudioSession.Route.BLUETOOTH, phone.routed);
        assertEquals("bluetooth:true", phone.published.get(phone.published.size() - 1));

        phone.bluetooth = false;
        session.devicesChanged();
        assertEquals("the choice made before the headset is kept", CallAudioSession.Route.SPEAKER, phone.routed);
    }

    @Test
    public void wiredIsPreferredOverBluetooth() {
        phone.wired = true;
        phone.bluetooth = true;
        session.start(false);
        assertEquals(CallAudioSession.Route.WIRED, phone.routed);
    }

    @Test
    public void nothingHappensOutsideACall() {
        session.setSpeaker(true);
        session.devicesChanged();
        assertNull(phone.routed);
        assertFalse(phone.lockHeld);
        assertFalse(phone.inCallMode);
    }

    @Test
    public void startingAgainDoesNotStackLocksOrReenterCallMode() {
        session.start(false);
        phone.inCallMode = false; // would be set back only by a second enterCallMode
        session.start(false);
        assertFalse("call mode entered twice would overwrite the mode saved for restore", phone.inCallMode);
        assertEquals(1, phone.acquires);
        session.end();
        assertFalse(phone.lockHeld);
    }
}
