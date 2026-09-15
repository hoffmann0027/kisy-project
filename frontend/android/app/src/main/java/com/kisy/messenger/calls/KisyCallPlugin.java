package com.kisy.messenger.calls;

import android.content.Intent;
import android.net.Uri;
import android.os.Build;
import android.provider.Settings;
import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

/**
 * The bridge between the native ringing screen and the call code in the WebView.
 *
 * Only the web layer can actually answer a call — it owns the session, the
 * WebSocket and WebRTC — but it does not exist while the app is killed, which
 * is exactly when a phone has to ring. So the native side rings, records what
 * the user tapped, and this plugin hands that decision over as soon as
 * JavaScript is alive to receive it.
 */
@CapacitorPlugin(name = "KisyCall")
public class KisyCallPlugin extends Plugin {

    /** The instance with a live bridge, or null while the app is not running. */
    private static KisyCallPlugin live;

    @Override
    public void load() {
        live = this;
    }

    @Override
    protected void handleOnDestroy() {
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
}
