package com.kisy.messenger.calls;

/**
 * Where a call's sound goes, and whether the screen blanks at the ear.
 *
 * The decisions live here, apart from AudioManager and PowerManager, for one
 * reason: the rule that matters most — the proximity lock is released when the
 * call ends, whatever state it ended in — must be testable without a phone. A
 * lock that outlives its call leaves the screen dark and deaf to touch until
 * the app is killed, and nothing on the screen can tell the user why.
 *
 * The routing rule, in order:
 *   1. a connected headset takes the sound — wired first, then Bluetooth;
 *   2. otherwise the loudspeaker if it was asked for;
 *   3. otherwise the earpiece.
 * A voice call starts on the earpiece, a video call on the loudspeaker. The
 * proximity lock is held only while the sound is on the earpiece of a voice
 * call: that is the one situation in which the phone is against a face. On the
 * loudspeaker or a headset it is in a hand or on a table, and a screen that
 * went dark whenever a finger passed the sensor would be a fault.
 */
public final class CallAudioSession {

    public enum Route {
        EARPIECE("earpiece"),
        SPEAKER("speaker"),
        WIRED("wired"),
        BLUETOOTH("bluetooth");

        public final String wire;

        Route(String wire) {
            this.wire = wire;
        }
    }

    /** Everything that touches the device, so the rules above can run in a unit test. */
    public interface Platform {
        /** MODE_IN_COMMUNICATION, remembering the mode to restore. */
        void enterCallMode();

        /** Restores the mode from before the call and drops any forced route. */
        void leaveCallMode();

        boolean hasWiredHeadset();

        boolean hasBluetoothHeadset();

        void routeTo(Route route);

        void acquireProximityLock();

        void releaseProximityLock();

        boolean isProximityLockHeld();

        /** Reports the route now in effect to the web layer. */
        void publish(Route route, boolean speakerRequested);
    }

    private final Platform platform;
    private boolean active;
    private boolean video;
    private boolean speakerRequested;
    private Route route;

    public CallAudioSession(Platform platform) {
        this.platform = platform;
    }

    /** The call has media. Idempotent: starting again re-applies the defaults for the kind of call. */
    public synchronized void start(boolean isVideo) {
        if (!active) platform.enterCallMode();
        active = true;
        video = isVideo;
        speakerRequested = isVideo;
        apply();
    }

    /** The loudspeaker button. Ignored outside a call. */
    public synchronized void setSpeaker(boolean on) {
        if (!active) return;
        speakerRequested = on;
        apply();
    }

    /** A headset came or went; the route follows it. */
    public synchronized void devicesChanged() {
        if (active) apply();
    }

    /**
     * The call is over. Always releases the proximity lock — even when the
     * session never started, or was already ended — because the cost of a
     * leaked lock is a dead screen, and the cost of a spare release is nothing.
     */
    public synchronized void end() {
        if (platform.isProximityLockHeld()) platform.releaseProximityLock();
        if (!active) return;
        active = false;
        video = false;
        speakerRequested = false;
        route = null;
        platform.leaveCallMode();
    }

    public synchronized boolean isActive() {
        return active;
    }

    public synchronized Route route() {
        return route;
    }

    private void apply() {
        Route next;
        if (platform.hasWiredHeadset()) next = Route.WIRED;
        else if (platform.hasBluetoothHeadset()) next = Route.BLUETOOTH;
        else if (speakerRequested) next = Route.SPEAKER;
        else next = Route.EARPIECE;

        route = next;
        platform.routeTo(next);

        boolean atTheEar = !video && next == Route.EARPIECE;
        if (atTheEar && !platform.isProximityLockHeld()) platform.acquireProximityLock();
        if (!atTheEar && platform.isProximityLockHeld()) platform.releaseProximityLock();

        platform.publish(next, speakerRequested);
    }
}
