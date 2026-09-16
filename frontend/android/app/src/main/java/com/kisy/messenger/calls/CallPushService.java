package com.kisy.messenger.calls;

import androidx.annotation.NonNull;
import android.util.Log;
import com.capacitorjs.plugins.pushnotifications.MessagingService;
import com.google.firebase.messaging.RemoteMessage;
import java.util.Map;

/**
 * Receives Firebase messages before the push plugin does.
 *
 * Extends the plugin's own service and still calls through to it, so ordinary
 * notifications and the pushNotificationReceived event behave exactly as
 * before; the only thing added is that a call push rings the phone natively
 * instead of waiting for JavaScript that, in a killed app, never runs.
 *
 * Replaces the plugin's service in the manifest — Firebase delivers to a single
 * service, and two declarations would make which one wins a matter of merge
 * order.
 */
public class CallPushService extends MessagingService {

    private static final String TAG = "KisyCall";

    @Override
    public void onMessageReceived(@NonNull RemoteMessage remoteMessage) {
        Map<String, String> data = remoteMessage.getData();
        String type = data.get("type");

        if ("call_invite".equals(type)) {
            String callId = data.get("callId");
            Log.i(TAG, "push call_invite " + callId);
            if (CallAudioSession.callInProgress()) {
                // The server sends this push even to an app that is open, and
                // Firebase may deliver it after the call was already answered
                // there. Ringing now would play through the conversation — as
                // Android's in-call beep, since the audio is in call mode. A
                // genuinely new caller is answered "busy" by the server anyway.
                Log.i(TAG, "push call_invite during a call — not ringing");
            } else {
                CallNotifications.ring(this, callId, data.get("callerName"));
            }
        } else if ("call_cancel".equals(type)) {
            String callId = data.get("callId");
            Log.i(TAG, "push call_cancel " + callId);
            // Only if it is the call being rung: a cancel for an older call
            // must not silence the one ringing now.
            if (callId == null || callId.equals(CallRinger.current())) {
                CallNotifications.dismiss(this);
            }
        }

        super.onMessageReceived(remoteMessage);
    }
}
