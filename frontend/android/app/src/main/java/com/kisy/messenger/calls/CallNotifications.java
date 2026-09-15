package com.kisy.messenger.calls;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.Context;
import android.content.Intent;
import android.os.Build;
import android.util.Log;
import androidx.core.app.NotificationCompat;
import androidx.core.app.NotificationManagerCompat;
import androidx.core.app.Person;
import com.kisy.messenger.R;

/**
 * The incoming-call notification — the part of a call that has to work with
 * the app not running at all.
 *
 * A data-only push wakes this process for a few seconds. In that window the
 * only way to take over the screen is a full-screen intent: Android launches
 * the attached activity itself, over the lock screen, because a background
 * process is no longer allowed to start an activity on its own.
 *
 * When the full-screen intent is refused — Android 14 grants it only to apps
 * the Play Store recognises as calling apps, and a sideloaded build has to be
 * allowed by hand — the same notification still arrives as a heads-up banner
 * with Accept and Decline on it, and CallRinger still rings. The call is then
 * answerable, just without the takeover screen.
 */
public final class CallNotifications {

    private static final String TAG = "KisyCall";

    public static final String CHANNEL_ID = "kisy_calls";
    private static final int NOTIFICATION_ID = 4711;

    /** Matches CallRinger: the notification outlives the server's 45s ring, then goes. */
    private static final long TIMEOUT_MS = 50_000L;

    private CallNotifications() {}

    private static void ensureChannel(Context ctx) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return;
        NotificationManager nm = ctx.getSystemService(NotificationManager.class);
        if (nm == null || nm.getNotificationChannel(CHANNEL_ID) != null) return;
        NotificationChannel ch = new NotificationChannel(
            CHANNEL_ID,
            ctx.getString(R.string.call_channel_name),
            NotificationManager.IMPORTANCE_HIGH
        );
        ch.setDescription(ctx.getString(R.string.call_channel_description));
        ch.setLockscreenVisibility(Notification.VISIBILITY_PUBLIC);
        // Silent on purpose: CallRinger owns the sound, because a channel plays
        // its tone once and a ringing phone has to keep ringing.
        ch.setSound(null, null);
        ch.enableVibration(false);
        ch.setShowBadge(false);
        nm.createNotificationChannel(ch);
    }

    /** Immutable is mandatory from Android 12 and harmless before it. */
    private static int flags(int extra) {
        return extra | PendingIntent.FLAG_IMMUTABLE;
    }

    /** Rings the phone: notification, full-screen intent and sound. */
    public static void ring(Context ctx, String callId, String callerName) {
        ensureChannel(ctx);

        String name = callerName == null || callerName.isEmpty() ? ctx.getString(R.string.call_unknown_caller) : callerName;

        PendingIntent screen = PendingIntent.getActivity(
            ctx,
            0,
            IncomingCallActivity.intent(ctx, IncomingCallActivity.ACTION_RING, callId, name),
            flags(PendingIntent.FLAG_UPDATE_CURRENT)
        );
        PendingIntent accept = PendingIntent.getActivity(
            ctx,
            1,
            IncomingCallActivity.intent(ctx, IncomingCallActivity.ACTION_ACCEPT, callId, name),
            flags(PendingIntent.FLAG_UPDATE_CURRENT)
        );
        // Declining goes to a receiver, not an activity: refusing a call should
        // not drag the whole app onto the screen.
        Intent declineIntent = new Intent(ctx, CallActionReceiver.class)
            .setAction(CallActionReceiver.ACTION_DECLINE)
            .putExtra(CallActionReceiver.EXTRA_CALL_ID, callId);
        PendingIntent decline = PendingIntent.getBroadcast(ctx, 2, declineIntent, flags(PendingIntent.FLAG_UPDATE_CURRENT));

        // CallStyle, not a plain notification with two buttons: from Android 12
        // this is how the system is told "this is a ringing call", and it is
        // what earns the call treatment — the heads-up banner that outranks
        // everything else, the lock-screen presentation, and on several OEM
        // skins the right to appear at all. A generic notification with an
        // "Ответить" button is, to the system, just a notification.
        Person caller = new Person.Builder().setName(name).setImportant(true).build();
        NotificationCompat.Builder b = new NotificationCompat.Builder(ctx, CHANNEL_ID)
            .setSmallIcon(android.R.drawable.sym_call_incoming)
            .setContentTitle(name)
            .setContentText(ctx.getString(R.string.call_incoming))
            .setStyle(NotificationCompat.CallStyle.forIncomingCall(caller, decline, accept))
            .setPriority(NotificationCompat.PRIORITY_MAX)
            .setCategory(NotificationCompat.CATEGORY_CALL)
            .setVisibility(NotificationCompat.VISIBILITY_PUBLIC)
            .setOngoing(true)
            .setAutoCancel(false)
            // Deliberately NOT setSilent(true): a silenced notification is
            // denied its heads-up banner, so the phone rang from CallRinger
            // while the call itself stayed hidden in the shade. The channel is
            // what keeps it quiet (no sound, no vibration) — the ringing is
            // CallRinger's job.
            .setTimeoutAfter(TIMEOUT_MS)
            .setContentIntent(screen)
            .setFullScreenIntent(screen, true);

        try {
            NotificationManagerCompat.from(ctx).notify(NOTIFICATION_ID, b.build());
        } catch (SecurityException e) {
            // POST_NOTIFICATIONS denied. Nothing can be shown, but the phone
            // still rings and the app still finds the call when it opens.
            Log.w(TAG, "notify refused: " + e.getMessage());
        }
        CallRinger.start(ctx, callId);
        Log.i(TAG, "incoming call " + callId + " from " + name);
        logDelivery(ctx);
    }

    /** Takes the call off the screen and silences the phone. */
    public static void dismiss(Context ctx) {
        NotificationManagerCompat.from(ctx).cancel(NOTIFICATION_ID);
        CallRinger.stop();
        // A cancelled call must not leave a live "Ответить" on the lock screen.
        IncomingCallActivity.closeIfShowing();
    }

    /**
     * Reports everything that decides whether a call is actually shown.
     *
     * When a phone rings but no call appears, the cause is almost never in this
     * code — it is a permission the system never granted, a channel the user or
     * the OEM quietly demoted, or a skin that keeps heads-up notifications for
     * itself. None of that is visible from the app's behaviour, so it is all
     * written down at the moment a call arrives.
     */
    private static void logDelivery(Context ctx) {
        NotificationManagerCompat compat = NotificationManagerCompat.from(ctx);
        StringBuilder s = new StringBuilder("delivery: notificationsEnabled=")
            .append(compat.areNotificationsEnabled())
            .append(" fullScreenIntent=")
            .append(canUseFullScreenIntent(ctx));

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            NotificationManager nm = ctx.getSystemService(NotificationManager.class);
            NotificationChannel ch = nm == null ? null : nm.getNotificationChannel(CHANNEL_ID);
            if (ch == null) {
                s.append(" channel=MISSING");
            } else {
                // importance below IMPORTANCE_HIGH (4) means no heads-up, and
                // it can be lowered after the channel was created.
                s.append(" channelImportance=").append(ch.getImportance()).append("/4");
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
                    s.append(" channelBlocked=").append(ch.getImportance() == NotificationManager.IMPORTANCE_NONE);
                }
            }
            if (nm != null) {
                s.append(" dndFilter=").append(nm.getCurrentInterruptionFilter());
            }
        }
        Log.i(TAG, s.toString());
    }

    /**
     * Whether Android will let this build take over the screen. False means
     * the call arrives as a banner instead — worth logging, since "the phone
     * buzzed but no call screen appeared" has exactly one cause.
     */
    public static boolean canUseFullScreenIntent(Context ctx) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.UPSIDE_DOWN_CAKE) return true;
        NotificationManager nm = ctx.getSystemService(NotificationManager.class);
        return nm != null && nm.canUseFullScreenIntent();
    }
}
