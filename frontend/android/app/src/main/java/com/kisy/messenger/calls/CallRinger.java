package com.kisy.messenger.calls;

import android.content.Context;
import android.media.AudioAttributes;
import android.media.AudioManager;
import android.media.MediaPlayer;
import android.media.RingtoneManager;
import android.net.Uri;
import android.os.Build;
import android.os.Handler;
import android.os.Looper;
import android.os.VibrationEffect;
import android.os.Vibrator;
import android.os.VibratorManager;
import android.util.Log;

/**
 * The sound and the buzz of an incoming call.
 *
 * Deliberately not the notification channel's own sound: a channel sound plays
 * once and stops, while a phone should keep ringing until somebody deals with
 * it. It also has to survive the full-screen intent being denied — on
 * Android 14 a sideloaded build may not be allowed to take over the screen,
 * and then this ringing is the only thing that tells the user anything is
 * happening.
 *
 * Stops on its own after the server's ring timeout so a push that arrives
 * after the caller gave up cannot leave a phone ringing forever.
 */
public final class CallRinger {

    private static final String TAG = "KisyCall";

    /** Slightly longer than the server's 45s ring timeout, so the server decides, not this. */
    private static final long MAX_RING_MS = 50_000L;

    private static final long[] PATTERN = { 0L, 1000L, 1000L };

    private static MediaPlayer player;
    private static Vibrator vibrator;
    private static String ringingCallId;
    private static final Handler handler = new Handler(Looper.getMainLooper());
    private static final Runnable autoStop = () -> {
        stop();
        // The caller has long given up; the screen goes with the sound.
        IncomingCallActivity.closeIfShowing();
    };

    private CallRinger() {}

    /** The call currently ringing on this device, or null. */
    public static synchronized String current() {
        return ringingCallId;
    }

    public static synchronized void start(Context ctx, String callId) {
        if (callId != null && callId.equals(ringingCallId)) return; // already ringing this one
        stop();
        ringingCallId = callId;

        AudioManager audio = (AudioManager) ctx.getSystemService(Context.AUDIO_SERVICE);
        int mode = audio == null ? AudioManager.RINGER_MODE_NORMAL : audio.getRingerMode();

        if (mode == AudioManager.RINGER_MODE_NORMAL) {
            try {
                Uri tone = RingtoneManager.getActualDefaultRingtoneUri(ctx, RingtoneManager.TYPE_RINGTONE);
                if (tone == null) tone = RingtoneManager.getDefaultUri(RingtoneManager.TYPE_RINGTONE);
                MediaPlayer mp = new MediaPlayer();
                mp.setDataSource(ctx.getApplicationContext(), tone);
                mp.setAudioAttributes(
                    new AudioAttributes.Builder()
                        .setUsage(AudioAttributes.USAGE_NOTIFICATION_RINGTONE)
                        .setContentType(AudioAttributes.CONTENT_TYPE_SONIFICATION)
                        .build()
                );
                mp.setLooping(true);
                mp.prepare();
                mp.start();
                player = mp;
            } catch (Exception e) {
                // A missing or unreadable ringtone must not swallow the call:
                // the vibration and the notification still get through.
                Log.w(TAG, "ringtone failed: " + e.getMessage());
            }
        }

        if (mode != AudioManager.RINGER_MODE_SILENT) {
            Vibrator v = vibratorOf(ctx);
            if (v != null && v.hasVibrator()) {
                try {
                    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                        v.vibrate(VibrationEffect.createWaveform(PATTERN, 0));
                    } else {
                        // minSdk is 24, and VibrationEffect only arrives in 26.
                        v.vibrate(PATTERN, 0);
                    }
                    vibrator = v;
                } catch (Exception e) {
                    Log.w(TAG, "vibrate failed: " + e.getMessage());
                }
            }
        }

        handler.removeCallbacks(autoStop);
        handler.postDelayed(autoStop, MAX_RING_MS);
        Log.i(TAG, "ringing " + callId + " (ringerMode=" + mode + ")");
    }

    public static synchronized void stop() {
        handler.removeCallbacks(autoStop);
        ringingCallId = null;
        if (player != null) {
            try {
                player.stop();
            } catch (Exception ignored) {
                // Already stopped; release below is what matters.
            }
            player.release();
            player = null;
        }
        if (vibrator != null) {
            vibrator.cancel();
            vibrator = null;
        }
    }

    /** Stops only if the given call is the one ringing (a late cancel for an old call is ignored). */
    public static synchronized void stopIf(String callId) {
        if (callId == null || callId.equals(ringingCallId)) stop();
    }

    private static Vibrator vibratorOf(Context ctx) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            VibratorManager vm = (VibratorManager) ctx.getSystemService(Context.VIBRATOR_MANAGER_SERVICE);
            return vm == null ? null : vm.getDefaultVibrator();
        }
        return (Vibrator) ctx.getSystemService(Context.VIBRATOR_SERVICE);
    }
}
