package com.kisy.messenger.calls;

import android.Manifest;
import android.content.Intent;
import android.net.Uri;
import android.os.Build;
import android.provider.Settings;
import androidx.core.app.NotificationManagerCompat;
import com.getcapacitor.JSObject;
import com.getcapacitor.PermissionState;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;
import com.getcapacitor.annotation.Permission;
import com.getcapacitor.annotation.PermissionCallback;

/**
 * The bridge between the native call layer and the call code in the WebView.
 *
 * Only the web layer can actually answer a call — it owns the session, the
 * WebSocket and WebRTC — but it does not exist while the app is killed, which
 * is exactly when a phone has to ring. So the native side rings, records what
 * the user tapped, and this plugin hands that decision over as soon as
 * JavaScript is alive to receive it.
 *
 * It also does the two things a web page cannot: route a call's sound to the
 * earpiece, loudspeaker or headset (with the screen blanking at the ear), and
 * report and request the permissions a call depends on.
 */
@CapacitorPlugin(
    name = "KisyCall",
    permissions = {
        @Permission(alias = KisyCallPlugin.MICROPHONE, strings = { Manifest.permission.RECORD_AUDIO }),
        @Permission(alias = KisyCallPlugin.NOTIFICATIONS, strings = { Manifest.permission.POST_NOTIFICATIONS }),
    }
)
public class KisyCallPlugin extends Plugin {

    static final String MICROPHONE = "microphone";
    static final String NOTIFICATIONS = "notifications";
    static final String FULL_SCREEN_INTENT = "fullScreenIntent";

    /** The instance with a live bridge, or null while the app is not running. */
    private static KisyCallPlugin live;

    /** Earpiece, loudspeaker, headset and the proximity lock of the call in progress. */
    private CallAudioSession audio;

    @Override
    public void load() {
        live = this;
        AndroidCallAudio platform = new AndroidCallAudio(getContext(), (route, speakerRequested) -> {
            JSObject data = new JSObject();
            data.put("route", route.wire);
            data.put("speaker", speakerRequested);
            notifyListeners("audioRoute", data);
        });
        audio = new CallAudioSession(platform);
        platform.attach(audio);
    }

    @Override
    protected void handleOnDestroy() {
        // The activity is going away, possibly mid-call. The proximity lock must
        // not outlive it: a leaked one leaves the screen dark and deaf to touch.
        if (audio != null) audio.end();
        if (live == this) live = null;
        super.handleOnDestroy();
    }

    /**
     * Tells a running app what was tapped, so it can act without waiting to be
     * asked. A killed app hears nothing here — it picks the same decision up
     * from getPendingAction() when it starts.
     */
    public static void announce(String action, String callId) {
        KisyCallPlugin p = live;
        if (p == null) return;
        JSObject data = new JSObject();
        data.put("action", action);
        data.put("callId", callId);
        p.notifyListeners("callAction", data);
    }

    /**
     * The decision taken on the ringing screen, or nulls when there was none.
     * Reading it consumes it: a decision replayed twice would reject a second,
     * unrelated call.
     */
    @PluginMethod
    public void getPendingAction(PluginCall call) {
        String[] pending = PendingCallAction.take(getContext());
        JSObject res = new JSObject();
        res.put("action", pending == null ? null : pending[0]);
        res.put("callId", pending == null ? null : pending[1]);
        call.resolve(res);
    }

    /** Silences the phone and clears the call notification (the web UI has taken over). */
    @PluginMethod
    public void stopRinging(PluginCall call) {
        CallNotifications.dismiss(getContext());
        call.resolve();
    }

    /**
     * Whether this build may take over the screen for a call. False on
     * Android 14+ until the user allows it by hand, and then a call on a
     * locked phone is a banner plus a ringtone rather than a call screen.
     */
    @PluginMethod
    public void canUseFullScreenIntent(PluginCall call) {
        JSObject res = new JSObject();
        res.put("granted", CallNotifications.canUseFullScreenIntent(getContext()));
        call.resolve(res);
    }

    /** Opens the system screen where that permission is granted. */
    @PluginMethod
    public void openFullScreenIntentSettings(PluginCall call) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.UPSIDE_DOWN_CAKE) {
            call.resolve();
            return;
        }
        Intent intent = new Intent(Settings.ACTION_MANAGE_APP_USE_FULL_SCREEN_INTENT)
            .setData(Uri.parse("package:" + getContext().getPackageName()))
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
        getContext().startActivity(intent);
        call.resolve();
    }

    // --- call audio --------------------------------------------------------

    /**
     * The call has media: route it for a voice call (earpiece, the screen
     * blanks at the ear) or a video call (loudspeaker, the screen stays on).
     * Called again once the call connects, because the WebView chooses its own
     * route when the remote audio starts, and it chooses the loudspeaker.
     */
    @PluginMethod
    public void startCallAudio(PluginCall call) {
        audio.start(Boolean.TRUE.equals(call.getBoolean("video", false)));
        resolveRoute(call);
    }

    /** The loudspeaker button. A connected headset still takes precedence. */
    @PluginMethod
    public void setSpeaker(PluginCall call) {
        audio.setSpeaker(Boolean.TRUE.equals(call.getBoolean("on", false)));
        resolveRoute(call);
    }

    /** The call is over: normal audio mode, and the screen is released. */
    @PluginMethod
    public void stopCallAudio(PluginCall call) {
        audio.end();
        call.resolve();
    }

    private void resolveRoute(PluginCall call) {
        CallAudioSession.Route route = audio.route();
        JSObject res = new JSObject();
        res.put("route", route == null ? null : route.wire);
        call.resolve(res);
    }

    // --- permissions -------------------------------------------------------
    //
    // One vocabulary for all three, Capacitor's own: "granted", "prompt",
    // "prompt-with-rationale" and "denied". Here "denied" always means the
    // system will not ask again and only Settings can change it. Capacitor
    // records that state itself after a request answered with "don't ask
    // again", which is what lets the onboarding stop asking in circles.

    @PluginMethod
    public void checkAppPermissions(PluginCall call) {
        JSObject res = new JSObject();
        res.put(NOTIFICATIONS, notificationState());
        res.put(MICROPHONE, getPermissionState(MICROPHONE).toString());
        res.put(FULL_SCREEN_INTENT, fullScreenIntentState());
        call.resolve(res);
    }

    /** Shows the system dialog for one permission and resolves with its new state. */
    @PluginMethod
    public void requestAppPermission(PluginCall call) {
        String name = call.getString("name", "");
        boolean hasDialog = MICROPHONE.equals(name)
            || (NOTIFICATIONS.equals(name) && Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU);
        if (!hasDialog) {
            // Nothing to show: notifications before Android 13 are a setting,
            // and so are full-screen intents from Android 14.
            resolveState(call, NOTIFICATIONS.equals(name) ? notificationState() : fullScreenIntentState());
            return;
        }
        if (getPermissionState(name) == PermissionState.GRANTED) {
            resolveState(call, stateOf(name));
            return;
        }
        requestPermissionForAlias(name, call, "permissionAnswered");
    }

    @PermissionCallback
    private void permissionAnswered(PluginCall call) {
        resolveState(call, stateOf(call.getString("name", "")));
    }

    /** Opens the settings page where the given permission is changed by hand. */
    @PluginMethod
    public void openPermissionSettings(PluginCall call) {
        String name = call.getString("name", "");
        String pkg = getContext().getPackageName();
        Intent intent;
        if (FULL_SCREEN_INTENT.equals(name) && Build.VERSION.SDK_INT >= Build.VERSION_CODES.UPSIDE_DOWN_CAKE) {
            intent = new Intent(Settings.ACTION_MANAGE_APP_USE_FULL_SCREEN_INTENT).setData(Uri.parse("package:" + pkg));
        } else if (NOTIFICATIONS.equals(name) && Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            intent = new Intent(Settings.ACTION_APP_NOTIFICATION_SETTINGS).putExtra(Settings.EXTRA_APP_PACKAGE, pkg);
        } else {
            intent = new Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS).setData(Uri.parse("package:" + pkg));
        }
        getContext().startActivity(intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK));
        call.resolve();
    }

    private String stateOf(String name) {
        return NOTIFICATIONS.equals(name) ? notificationState() : getPermissionState(name).toString();
    }

    private void resolveState(PluginCall call, String state) {
        JSObject res = new JSObject();
        res.put("state", state);
        call.resolve(res);
    }

    private String notificationState() {
        boolean enabled = NotificationManagerCompat.from(getContext()).areNotificationsEnabled();
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) {
            // Nothing to ask before Android 13: on unless turned off in Settings.
            return enabled ? PermissionState.GRANTED.toString() : PermissionState.DENIED.toString();
        }
        PermissionState state = getPermissionState(NOTIFICATIONS);
        // The permission can be granted while the app's notifications are
        // switched off in Settings; that is a "no" only Settings can undo.
        if (state == PermissionState.GRANTED && !enabled) return PermissionState.DENIED.toString();
        return state.toString();
    }

    private String fullScreenIntentState() {
        return CallNotifications.canUseFullScreenIntent(getContext())
            ? PermissionState.GRANTED.toString()
            : PermissionState.DENIED.toString();
    }
}
