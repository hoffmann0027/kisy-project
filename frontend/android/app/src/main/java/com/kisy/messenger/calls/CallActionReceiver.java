package com.kisy.messenger.calls;

import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.util.Log;

/** Handles Decline from the notification, without opening the app. */
public class CallActionReceiver extends BroadcastReceiver {

    private static final String TAG = "KisyCall";

    public static final String ACTION_DECLINE = "com.kisy.messenger.CALL_DECLINE";
    public static final String EXTRA_CALL_ID = "callId";

    @Override
    public void onReceive(Context ctx, Intent intent) {
        if (!ACTION_DECLINE.equals(intent.getAction())) return;
        String callId = intent.getStringExtra(EXTRA_CALL_ID);
        Log.i(TAG, "declined " + callId + " from the notification");

        CallNotifications.dismiss(ctx);
        // The reject itself needs a signed-in session, which only exists inside
        // the WebView; recorded here and sent the moment the app is opened.
        PendingCallAction.put(ctx, PendingCallAction.REJECT, callId);
        // A running app can send it right now instead of waiting to be opened.
        KisyCallPlugin.announce(PendingCallAction.REJECT, callId);
    }
}
