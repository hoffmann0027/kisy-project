package com.kisy.messenger.calls;

import android.app.Activity;
import android.app.KeyguardManager;
import android.content.Context;
import android.content.Intent;
import android.os.Build;
import android.os.Bundle;
import android.util.Log;
import android.view.WindowManager;
import android.widget.TextView;
import com.kisy.messenger.MainActivity;
import com.kisy.messenger.R;

/**
 * The screen that appears when the phone rings — over the lock screen, with
 * the app not running.
 *
 * It is deliberately native and deliberately dumb: no WebView, no session, no
 * network. Starting Capacitor takes seconds the caller does not have, and on a
 * locked phone it would be showing a chat list behind the lock anyway. All
 * this screen does is show who is calling and record the answer; the call
 * itself is picked up by the web layer through KisyCallPlugin once MainActivity
 * is up (see useCall.resumePending).
 */
public class IncomingCallActivity extends Activity {

    private static final String TAG = "KisyCall";

    public static final String ACTION_RING = "ring";
    public static final String ACTION_ACCEPT = "accept";

    private static final String EXTRA_ACTION = "action";
    private static final String EXTRA_CALL_ID = "callId";
    private static final String EXTRA_CALLER = "callerName";

    public static Intent intent(Context ctx, String action, String callId, String callerName) {
        return new Intent(ctx, IncomingCallActivity.class)
            .setAction(action) // distinct actions, or PendingIntents would collapse into one
            .putExtra(EXTRA_ACTION, action)
            .putExtra(EXTRA_CALL_ID, callId)
            .putExtra(EXTRA_CALLER, callerName)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_CLEAR_TOP);
    }

    /**
     * The call screen currently on display, so a call that ends elsewhere can
     * take it down. Without this, a cancelled call leaves a dead "Ответить"
     * sitting on top of the lock screen until the user dismisses it by hand.
     */
    private static volatile IncomingCallActivity showing;

    /** Closes the ringing screen if one is up. Safe to call from any thread. */
    static void closeIfShowing() {
        IncomingCallActivity a = showing;
        if (a == null) return;
        showing = null;
        a.runOnUiThread(a::finish);
    }

    private String callId;

    @Override
    protected void onCreate(Bundle saved) {
        super.onCreate(saved);
        showOverLockScreen();

        callId = getIntent().getStringExtra(EXTRA_CALL_ID);
        String action = getIntent().getStringExtra(EXTRA_ACTION);
        String caller = getIntent().getStringExtra(EXTRA_CALLER);

        // Accept was tapped on the notification itself: there is nothing to
        // show, go straight to the app.
        if (ACTION_ACCEPT.equals(action)) {
            accept();
            return;
        }

        showing = this;
        setContentView(R.layout.activity_incoming_call);
        TextView name = findViewById(R.id.call_caller);
        name.setText(caller == null || caller.isEmpty() ? getString(R.string.call_unknown_caller) : caller);
        findViewById(R.id.call_accept).setOnClickListener(v -> accept());
        findViewById(R.id.call_decline).setOnClickListener(v -> decline());
    }

    @Override
    protected void onNewIntent(Intent intent) {
        super.onNewIntent(intent);
        setIntent(intent);
        String action = intent.getStringExtra(EXTRA_ACTION);
        callId = intent.getStringExtra(EXTRA_CALL_ID);
        if (ACTION_ACCEPT.equals(action)) accept();
    }

    /** Wakes the screen and draws on top of the keyguard (API 27+ API, flags below that). */
    private void showOverLockScreen() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O_MR1) {
            setShowWhenLocked(true);
            setTurnScreenOn(true);
        } else {
            getWindow()
                .addFlags(
                    WindowManager.LayoutParams.FLAG_SHOW_WHEN_LOCKED |
                    WindowManager.LayoutParams.FLAG_TURN_SCREEN_ON |
                    WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON
                );
        }
        getWindow().addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);
    }

    private void accept() {
        Log.i(TAG, "accepted " + callId + " from the call screen");
        showing = null; // this screen is leaving on its own terms
        CallNotifications.dismiss(this);
        PendingCallAction.put(this, PendingCallAction.ACCEPT, callId);
        KisyCallPlugin.announce(PendingCallAction.ACCEPT, callId);

        // The user has to get past the lock screen anyway to talk; asking now
        // means the app opens to the call instead of to a locked black screen.
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O_MR1) {
            KeyguardManager km = getSystemService(KeyguardManager.class);
            if (km != null && km.isKeyguardLocked()) km.requestDismissKeyguard(this, null);
        }

        Intent app = new Intent(this, MainActivity.class)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_SINGLE_TOP);
        startActivity(app);
        finish();
    }

    private void decline() {
        Log.i(TAG, "declined " + callId + " from the call screen");
        showing = null;
        CallNotifications.dismiss(this);
        PendingCallAction.put(this, PendingCallAction.REJECT, callId);
        KisyCallPlugin.announce(PendingCallAction.REJECT, callId);
        finish();
    }

    @Override
    protected void onDestroy() {
        super.onDestroy();
        if (showing == this) showing = null;
        // Swiped away rather than answered: stop the noise, leave the call to
        // the server's ring timeout.
        CallRinger.stopIf(callId);
    }
}
