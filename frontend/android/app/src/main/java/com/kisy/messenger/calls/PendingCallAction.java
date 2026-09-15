package com.kisy.messenger.calls;

import android.content.Context;
import android.content.SharedPreferences;

/**
 * What the user decided on the ringing screen, kept until the web layer can
 * act on it.
 *
 * The decision and the code that can carry it out live in two different
 * worlds: Accept and Decline are native buttons that work with the app killed,
 * while answering a call means WebRTC inside the WebView, which only exists
 * once the app is running. So the tap is recorded here and replayed by
 * KisyCallPlugin.getPendingAction() as soon as JavaScript asks.
 *
 * Disk, not a static field: the process that showed the ringing screen can be
 * killed before MainActivity starts, and an in-memory decision would die with
 * it.
 */
public final class PendingCallAction {

    public static final String ACCEPT = "accept";
    public static final String REJECT = "reject";

    private static final String PREFS = "kisy.calls";
    private static final String KEY_ACTION = "pendingAction";
    private static final String KEY_CALL_ID = "pendingCallId";
    private static final String KEY_AT = "pendingAt";

    /**
     * How long a recorded decision stays worth replaying. A call nobody picked
     * up is over in under a minute (the server rings for 45 seconds), so a
     * decision found hours later belongs to a call that no longer exists —
     * replaying it would reject whatever call happens to be ringing now.
     */
    private static final long MAX_AGE_MS = 5 * 60 * 1000L;

    private PendingCallAction() {}

    private static SharedPreferences prefs(Context ctx) {
        return ctx.getApplicationContext().getSharedPreferences(PREFS, Context.MODE_PRIVATE);
    }

    public static void put(Context ctx, String action, String callId) {
        prefs(ctx)
            .edit()
            .putString(KEY_ACTION, action)
            .putString(KEY_CALL_ID, callId)
            .putLong(KEY_AT, System.currentTimeMillis())
            .apply();
    }

    /** Returns {action, callId} and forgets it, or null when there is nothing (or nothing fresh). */
    public static String[] take(Context ctx) {
        SharedPreferences p = prefs(ctx);
        String action = p.getString(KEY_ACTION, null);
        String callId = p.getString(KEY_CALL_ID, null);
        long at = p.getLong(KEY_AT, 0L);
        clear(ctx);
        if (action == null || callId == null) return null;
        if (System.currentTimeMillis() - at > MAX_AGE_MS) return null;
        return new String[] { action, callId };
    }

    public static void clear(Context ctx) {
        prefs(ctx).edit().remove(KEY_ACTION).remove(KEY_CALL_ID).remove(KEY_AT).apply();
    }
}
